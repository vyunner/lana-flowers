package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"lana-flowers-go/internal/auth"
	dbpkg "lana-flowers-go/internal/db"
	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/handler/bouquets"
	eventshandler "lana-flowers-go/internal/handler/events"
	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/handler/preview"
	"lana-flowers-go/internal/handler/upload"
	"lana-flowers-go/internal/handler/users"
	"lana-flowers-go/internal/handler/webhook"
	"lana-flowers-go/internal/response"
	"lana-flowers-go/internal/telegram"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// initDataMaxAge — окно anti-replay для Telegram initData. Telegram-доки рекомендуют
// порядок часа; 3ч — компромисс между UX (юзер открыл мини-апп, отвлёкся, вернулся)
// и безопасностью (украденный initData нельзя реплеить сутками).
const initDataMaxAge = 3 * time.Hour

func main() {
	_ = godotenv.Load()

	if len(os.Args) >= 2 && os.Args[1] == "migrate" {
		runMigrateCmd(os.Args[2:])
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "thumbs" {
		runThumbsCmd(os.Args[2:])
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "preview" {
		runPreviewCmd(os.Args[2:])
		return
	}

	port := getenv("APP_PORT", "8080")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required in .env")
	}

	// Production-режим — без debug-логов от Gin и без warning'ов в journal.
	// Локально: оставьте APP_ENV пустым / development.
	appEnv := getenv("APP_ENV", "development")
	if appEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	conn, err := dbpkg.ConnectDB(dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer conn.Close()

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies(nil)

	// CORS-whitelist: список через запятую в CORS_ORIGINS. Дефолт — продовый
	// vercel-домен. AllowAllOrigins:true опасно даже для mini-app: любой сайт
	// может слать запросы от лица юзера если получит initData.
	allowedOrigins := strings.Split(getenv("CORS_ORIGINS", "https://lana-flowers.vercel.app"), ",")
	for i, o := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(o)
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:  allowedOrigins,
		AllowMethods:  []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-Telegram-Init-Data", "X-Secret"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Background-воркер: ставит pending-офферы старше 7 дней в expired
	// и шлёт DM/SSE покупателям. Без него inbox продавца раздувается
	// мёртвыми предложениями годами. Останавливается через workerCtx
	// при shutdown.
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()
	offers.StartPendingExpireWorker(workerCtx, conn)

	// Webhook регистрируется ДО auth middleware — защищён secret_token'ом самого Telegram.
	webhook.RegisterRoutes(r, conn)

	// Preview-эндпоинт для dev-проверки in-app toast'ов — тоже ДО middleware,
	// защищён собственным X-Admin-Secret header. См. internal/handler/preview.
	preview.RegisterRoutes(r, conn)

	r.Use(authMiddleware(conn))

	users.RegisterRoutes(r, conn)
	bouquets.RegisterRoutes(r, conn)
	offers.RegisterRoutes(r, conn)
	upload.RegisterRoutes(r, conn)
	eventshandler.RegisterRoutes(r, conn)

	// Graceful shutdown: ловим SIGTERM/SIGINT, даём 10s in-flight запросам
	// нормально закончиться вместо kill -9. Без этого деплой обрывает
	// горутины nofity'ев и in-flight POST'ы.
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Print("shutdown signal received")

	// Останавливаем background-воркеров (expire-sweep) ДО srv.Shutdown
	// чтобы они не открыли новые DB-транзакции в момент остановки.
	stopWorkers()

	// Закрываем все SSE-стримы — иначе srv.Shutdown висит весь таймаут
	// в ожидании пока long-lived соединения закроются сами.
	events.Default().CloseAll()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	// Notify-горутины (telegram.Notify*) могут ещё лететь — дадим им
	// шанс долететь до Telegram API ещё пару секунд.
	telegram.WaitNotifications(3 * time.Second)
	log.Print("server stopped")
}

// authMiddleware:
//   - /health и /tg/webhook — public (последний защищён secret_token Telegram)
//   - X-Telegram-Init-Data — валидируем HMAC, апсертим юзера, кладём user_id в контекст
//   - иначе 401
//
// X-Secret-бэкдор удалён — нигде не использовался, риск утечки переменной
// превышал пользу. Если когда-то понадобится server-to-server — пишите
// отдельный middleware с явным whitelist'ом эндпоинтов.
func authMiddleware(db *sql.DB) gin.HandlerFunc {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required in .env")
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health" || path == "/tg/webhook" {
			c.Next()
			return
		}

		// Telegram WebApp initData
		raw := c.GetHeader("X-Telegram-Init-Data")
		if raw == "" {
			// fallback — некоторые SDK кладут в Authorization: tma <initData>
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "tma ") {
				raw = strings.TrimPrefix(authHeader, "tma ")
			}
		}
		if raw == "" {
			// SSE-fallback: EventSource в браузере НЕ умеет custom headers,
			// поэтому /events приходит с initData в query (?tma=...).
			raw = c.Query("tma")
		}

		if raw == "" {
			response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing init data")
			c.Abort()
			return
		}

		data, err := auth.ParseAndValidate(raw, botToken, initDataMaxAge)
		if err != nil {
			response.Err(c, http.StatusUnauthorized, "INVALID_INIT_DATA", err.Error())
			c.Abort()
			return
		}
		if data.User == nil {
			response.Err(c, http.StatusUnauthorized, "MISSING_USER", "init data has no user")
			c.Abort()
			return
		}

		userID := strconv.FormatInt(data.User.ID, 10)

		// Auto-upsert юзера: создаём если новый (без phone_number), обновляем кэшированный
		// профиль из Telegram.
		if err := upsertUser(db, userID, data.User); err != nil {
			log.Printf("upsert user %s: %v", userID, err)
		}

		// Гейт: phone_number обязателен. Исключения — /users/me (читаем статус)
		// и /upload (юзер грузит аватарку во время онбординга до телефона).
		if path != "/users/me" && path != "/upload" {
			var phone string
			if err := db.QueryRow(`SELECT phone_number FROM users WHERE user_id = $1`, userID).Scan(&phone); err == nil && phone == "" {
				response.Err(c, http.StatusForbidden, "NOT_REGISTERED",
					"Сначала пройдите регистрацию")
				c.Abort()
				return
			}
		}

		c.Set("user_id", userID)
		c.Set("tg_user", data.User)
		c.Next()
	}
}

func upsertUser(db *sql.DB, userID string, u *auth.User) error {
	_, err := db.Exec(`
		INSERT INTO users (user_id, first_name, last_name, username, photo_url, is_premium, language_code, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			first_name    = EXCLUDED.first_name,
			last_name     = EXCLUDED.last_name,
			username      = EXCLUDED.username,
			photo_url     = EXCLUDED.photo_url,
			is_premium    = EXCLUDED.is_premium,
			language_code = EXCLUDED.language_code,
			last_seen_at  = NOW()
	`, userID, u.FirstName, u.LastName, u.Username, u.PhotoURL, u.IsPremium, u.LanguageCode)
	return err
}
