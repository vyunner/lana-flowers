package auth

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

// initDataMaxAge — окно anti-replay для Telegram initData. Telegram-доки
// рекомендуют порядок часа; 3ч — компромисс между UX (юзер открыл мини-апп,
// отвлёкся, вернулся) и безопасностью (украденный initData нельзя реплеить
// сутками).
const initDataMaxAge = 3 * time.Hour

// Middleware — Gin-middleware:
//   - /health и /tg/webhook пропускаются без auth (последний защищён
//     secret_token Telegram'а в самом webhook handler'е)
//   - читаем initData из header'а X-Telegram-Init-Data или Authorization: tma <...>,
//     или из query ?tma= (для SSE-стрима, который не умеет custom headers)
//   - валидируем HMAC по bot token'у через ParseAndValidate
//   - upsert юзера в users (создаём если новый, обновляем кэш профайла)
//   - phone-gate: юзеры без phone_number блокируются на всех endpoint'ах
//     кроме /users/me (читать свой статус) и /upload (грузить аватарку
//     до телефона в онбординге)
//   - кладём user_id (string) и tg_user (*User) в Gin-context
func Middleware(db *sql.DB) gin.HandlerFunc {
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

		data, err := ParseAndValidate(raw, botToken, initDataMaxAge)
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

		// Auto-upsert юзера: создаём если новый (без phone_number), обновляем
		// кэшированный профиль из Telegram.
		if err := upsertUser(db, userID, data.User); err != nil {
			log.Printf("upsert user %s: %v", userID, err)
		}

		// Phone-gate: phone_number обязателен для всего кроме онбординг-эндпоинтов.
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

// upsertUser — INSERT...ON CONFLICT для кэширования профиля юзера из
// Telegram-данных. phone_number ОТДЕЛЬНО не пишется (он приходит через
// PATCH /users/me либо через webhook contact), чтобы не затереть номер
// при каждом проходе через middleware.
func upsertUser(db *sql.DB, userID string, u *User) error {
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
