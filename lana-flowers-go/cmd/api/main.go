package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"lana-flowers-go/internal/auth"
	dbpkg "lana-flowers-go/internal/db"
	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/handler/bouquets"
	eventshandler "lana-flowers-go/internal/handler/events"
	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/handler/upload"
	"lana-flowers-go/internal/handler/users"
	"lana-flowers-go/internal/handler/webhook"
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

	// Background-воркер: ставит pending-офферы старше TTL в expired и шлёт
	// DM/SSE покупателям. Без него inbox продавца раздувается мёртвыми
	// предложениями. Останавливается через workerCtx при shutdown.
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()
	offers.StartPendingExpireWorker(workerCtx, conn)

	// Webhook регистрируется ДО auth middleware — защищён secret_token'ом самого Telegram.
	webhook.RegisterRoutes(r, conn)

	r.Use(auth.Middleware(conn))

	users.RegisterRoutes(r, conn)
	bouquets.RegisterRoutes(r, conn)
	offers.RegisterRoutes(r, conn)
	upload.RegisterRoutes(r, conn)
	eventshandler.RegisterRoutes(r, conn)

	// Graceful shutdown: ловим SIGTERM/SIGINT, даём 10s in-flight запросам
	// нормально закончиться вместо kill -9. Без этого деплой обрывает
	// горутины notify'ев и in-flight POST'ы.
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
