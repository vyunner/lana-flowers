// Package preview — dev-эндпоинт для проверки in-app toast'ов (SSE-events).
//
// CLI ./app preview шлёт DM-нотификации напрямую через Telegram API — это
// работает из отдельного процесса. А SSE-event'ы публикуются в in-memory
// events.Hub живого сервера, поэтому отдельный процесс до подписчиков не
// дотянется. Решение: HTTP-эндпоинт, который сам сервер вызывает в свой
// же events.Default().Publish().
//
// Аутентификация — заголовок X-Admin-Secret должен совпадать с env
// ADMIN_PREVIEW_SECRET. Если env не задан — эндпоинт fail-closed (403).
// Это намеренно безопаснее чем no-auth: если кто-то найдёт URL, он не
// сможет спамить toast'ы юзерам без знания секрета.
package preview

import (
	"database/sql"
	"net/http"
	"os"

	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, _ *sql.DB) {
	r.POST("/admin/preview-event", PreviewEvent)
}

type previewReq struct {
	UserID       string `json:"user_id" binding:"required"`
	Type         string `json:"type" binding:"required"` // напр. "offer.created"
	BouquetTitle string `json:"bouquet_title"`
	Price        int64  `json:"price"`
	OfferID      int64  `json:"offer_id"`
}

func PreviewEvent(c *gin.Context) {
	secret := os.Getenv("ADMIN_PREVIEW_SECRET")
	if secret == "" || c.GetHeader("X-Admin-Secret") != secret {
		response.Err(c, http.StatusForbidden, "FORBIDDEN", "")
		return
	}
	var req previewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	events.Default().Publish(req.UserID, events.Event{
		Type:         req.Type,
		OfferID:      req.OfferID,
		BouquetTitle: req.BouquetTitle,
		Price:        req.Price,
	})
	response.OK(c, gin.H{"ok": true})
}
