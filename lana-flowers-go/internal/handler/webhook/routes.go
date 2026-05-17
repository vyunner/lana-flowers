// Package webhook — Telegram Bot API webhook endpoint.
//
// Структура пакета:
//   - routes.go   (этот файл) — RegisterRoutes + диспетчер Handle
//   - types.go    — типы Update/Message/CallbackQuery/etc
//   - message.go  — handleMessage (/start), handleContact, handleReply (counter via ForceReply)
//   - callback.go — handleCallback (inline-кнопки accept/reject/counter) + appendStatusLine
package webhook

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const miniAppURL = "https://lana-flowers.vercel.app"

// webhookSecret кэшируется на старте — в проде должен быть задан, иначе
// /tg/webhook открыт миру и любой может слать поддельные callback_query
// от лица любого user_id.
var webhookSecret string

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	webhookSecret = os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	if webhookSecret == "" && os.Getenv("APP_ENV") == "production" {
		log.Fatal("TELEGRAM_WEBHOOK_SECRET is required in production (.env)")
	}
	if webhookSecret == "" {
		log.Print("WARN: TELEGRAM_WEBHOOK_SECRET is empty — webhook unauthenticated (OK only in development)")
	}
	r.POST("/tg/webhook", func(c *gin.Context) { Handle(c, db) })
}

// Handle — диспетчер update'ов от Telegram. Проверяет секретный header,
// парсит JSON, кидает в нужный handler (callback / contact / reply / message).
func Handle(c *gin.Context, db *sql.DB) {
	// Если secret задан — header обязан совпадать. Если не задан (только в
	// dev по логу выше) — пропускаем без проверки.
	if webhookSecret != "" && c.GetHeader("X-Telegram-Bot-Api-Secret-Token") != webhookSecret {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var u Update
	if err := c.ShouldBindJSON(&u); err != nil {
		log.Printf("webhook: bad json: %v", err)
		c.Status(http.StatusOK)
		return
	}

	switch {
	case u.CallbackQuery != nil:
		handleCallback(db, u.CallbackQuery)
	case u.Message != nil && u.Message.Contact != nil:
		handleContact(db, u.Message)
	case u.Message != nil && u.Message.ReplyToMessage != nil:
		handleReply(db, u.Message)
	case u.Message != nil:
		handleMessage(db, u.Message)
	}

	// Telegram ждёт 2xx — иначе будет ретраить тот же update до 24 часов.
	c.Status(http.StatusOK)
}
