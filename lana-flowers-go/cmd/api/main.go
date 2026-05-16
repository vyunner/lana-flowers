package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"lana-flowers-go/internal/auth"
	dbpkg "lana-flowers-go/internal/db"
	"lana-flowers-go/internal/handler/bouquets"
	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/handler/upload"
	"lana-flowers-go/internal/handler/users"
	"lana-flowers-go/internal/handler/webhook"
	"lana-flowers-go/internal/response"

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

func main() {
	_ = godotenv.Load()

	if len(os.Args) >= 2 && os.Args[1] == "migrate" {
		runMigrateCmd(os.Args[2:])
		return
	}

	port := getenv("APP_PORT", "8080")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required in .env")
	}

	conn, err := dbpkg.ConnectDB(dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer conn.Close()

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies(nil)

	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"*"},
		AllowHeaders:    []string{"*"},
		ExposeHeaders:   []string{"*"},
		MaxAge:          12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Webhook регистрируется ДО auth middleware — защищён secret_token'ом самого Telegram.
	webhook.RegisterRoutes(r, conn)

	r.Use(authMiddleware(conn))

	users.RegisterRoutes(r, conn)
	bouquets.RegisterRoutes(r, conn)
	offers.RegisterRoutes(r, conn)
	upload.RegisterRoutes(r, conn)

	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// authMiddleware:
//   - /health — public
//   - X-Secret == API_SECRET — server-to-server (полный доступ, без user_id в контексте)
//   - X-Telegram-Init-Data — валидируем HMAC, апсертим юзера, кладём user_id в контекст
//   - иначе 401
func authMiddleware(db *sql.DB) gin.HandlerFunc {
	apiSecret := os.Getenv("API_SECRET")
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health" || path == "/tg/webhook" {
			c.Next()
			return
		}

		// 1) Server-to-server по shared secret
		if apiSecret != "" && c.GetHeader("X-Secret") == apiSecret {
			c.Set("auth_type", "secret")
			c.Next()
			return
		}

		// 2) Telegram WebApp initData
		raw := c.GetHeader("X-Telegram-Init-Data")
		if raw == "" {
			// fallback — некоторые SDK кладут в Authorization: tma <initData>
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "tma ") {
				raw = strings.TrimPrefix(authHeader, "tma ")
			}
		}

		if raw == "" {
			response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing init data")
			c.Abort()
			return
		}

		// 24h max age — anti-replay
		data, err := auth.ParseAndValidate(raw, botToken, 24*time.Hour)
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

		c.Set("auth_type", "telegram")
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
