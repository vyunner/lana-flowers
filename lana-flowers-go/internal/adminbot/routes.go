// Package adminbot — отдельный Telegram-бот для админ-панели маркетплейса.
//
// Отделён от основного бота (internal/handler/webhook): другой
// ADMIN_BOT_TOKEN, другой webhook URL, своя БД (admins, admin_requests,
// event_log). Это исключает риск что баг в админ-боте уронит основной
// flow (онбординг/торг) и наоборот.
//
// Поток событий маркетплейса:
//
//	hookpoint в основном коде → adminbot.Record(db, eventType, payload, text)
//	   ↓                              ↓
//	  event_log.INSERT       AdminsForEvent → DM каждому enabled админу
package adminbot

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// adminWebhookSecret кэшируется на старте. ОБЯЗАТЕЛЕН в проде —
// без него /tg/admin-webhook открыт миру.
var adminWebhookSecret string

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	adminWebhookSecret = os.Getenv("ADMIN_WEBHOOK_SECRET")
	if adminWebhookSecret == "" && os.Getenv("APP_ENV") == "production" {
		log.Fatal("ADMIN_WEBHOOK_SECRET is required in production (.env)")
	}
	if adminWebhookSecret == "" {
		log.Print("WARN: ADMIN_WEBHOOK_SECRET is empty — admin webhook unauthenticated")
	}
	r.POST("/tg/admin-webhook", func(c *gin.Context) { Handle(c, db) })
}

func Handle(c *gin.Context, db *sql.DB) {
	if adminWebhookSecret != "" && c.GetHeader("X-Telegram-Bot-Api-Secret-Token") != adminWebhookSecret {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	var u Update
	if err := c.ShouldBindJSON(&u); err != nil {
		log.Printf("adminbot webhook: bad json: %v", err)
		c.Status(http.StatusOK)
		return
	}
	handleUpdate(db, &u)
	c.Status(http.StatusOK) // Telegram ждёт 2xx иначе retry'ит
}
