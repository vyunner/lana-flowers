// Package events handler — SSE-стрим событий клиенту.
// GET /events — держит соединение открытым, шлёт data: <json>\n\n
// при каждом publish'е для текущего юзера.
package events

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, _ *sql.DB) {
	r.GET("/events", Stream)
}

// Stream — SSE-эндпоинт. Auth уже сделан middleware'ом (initData валиден),
// user_id в context. Соединение держим открытым до клиентского
// disconnect'а или server shutdown'а; шлём data-frames при event'ах +
// ping каждые 25с чтобы Caddy/Telegram не закрыл idle-connection.
func Stream(c *gin.Context) {
	uid, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	userID := uid.(string)

	// SSE-заголовки. X-Accel-Buffering=no — отключает буферизацию у nginx
	// (Caddy без этого тоже норм, но не мешает на всякий).
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	ch, unsub := events.Default().Subscribe(userID)
	defer unsub()

	// «Готов» — первый event сразу при подключении, чтобы клиент знал
	// что стрим живой (без него EventSource считает соединение установленным,
	// но фронту приятнее иметь явный sentinel).
	_, _ = fmt.Fprintf(c.Writer, "event: ready\ndata: {}\n\n")
	c.Writer.Flush()

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			// SSE-комментарий (начинается с :) — не доходит до onmessage,
			// но держит TCP-соединение живым.
			if _, err := fmt.Fprintf(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		case e, ok := <-ch:
			if !ok {
				// Hub закрыл канал (graceful shutdown) — стрим завершаем,
				// клиент сам переподключится через EventSource auto-retry.
				return
			}
			data, _ := json.Marshal(e)
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}
