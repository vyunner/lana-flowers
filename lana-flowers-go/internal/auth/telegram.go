// Package auth — валидация Telegram WebApp initData по bot token.
// Алгоритм: https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// User — данные о юзере, как их шлёт Telegram внутри initData.user.
type User struct {
	ID              int64  `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	LanguageCode    string `json:"language_code"`
	IsPremium       bool   `json:"is_premium"`
	AllowsWriteToPM bool   `json:"allows_write_to_pm"`
	PhotoURL        string `json:"photo_url"`
}

// InitData — распарсенный и валидированный initData.
type InitData struct {
	QueryID  string
	User     *User
	AuthDate time.Time
	Hash     string
	Raw      string
}

// ParseAndValidate проверяет HMAC и возвращает структурированный InitData.
// Если maxAge > 0, отвергает initData старше этого срока (anti-replay).
func ParseAndValidate(rawInitData, botToken string, maxAge time.Duration) (*InitData, error) {
	if rawInitData == "" {
		return nil, fmt.Errorf("empty init data")
	}
	if botToken == "" {
		return nil, fmt.Errorf("bot token not configured")
	}

	values, err := url.ParseQuery(rawInitData)
	if err != nil {
		return nil, fmt.Errorf("parse query: %w", err)
	}

	hash := values.Get("hash")
	if hash == "" {
		return nil, fmt.Errorf("missing hash")
	}
	values.Del("hash")

	// Собираем data-check-string: ключи отсортированы, формат "key=value", разделитель \n.
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(values.Get(k))
	}
	dataCheckString := sb.String()

	// secret_key = HMAC_SHA256(message=bot_token, key="WebAppData")
	// hash       = HMAC_SHA256(message=data_check_string, key=secret_key)
	secretKey := hmacSHA256([]byte(botToken), []byte("WebAppData"))
	calc := hmacSHA256([]byte(dataCheckString), secretKey)
	calcHex := hex.EncodeToString(calc)

	// Сравниваем lowercase-байты. Telegram отдаёт hash в lowercase, но если
	// какой-то клиент / прокси нормализует регистр — не хотим отбивать валидные
	// запросы как 401.
	if !hmac.Equal([]byte(calcHex), []byte(strings.ToLower(hash))) {
		return nil, fmt.Errorf("hash mismatch")
	}

	out := &InitData{
		QueryID: values.Get("query_id"),
		Hash:    hash,
		Raw:     rawInitData,
	}

	if authDateStr := values.Get("auth_date"); authDateStr != "" {
		ts, err := strconv.ParseInt(authDateStr, 10, 64)
		if err == nil {
			out.AuthDate = time.Unix(ts, 0)
			if maxAge > 0 && time.Since(out.AuthDate) > maxAge {
				return nil, fmt.Errorf("init data expired")
			}
		}
	}

	if userStr := values.Get("user"); userStr != "" {
		var u User
		if err := json.Unmarshal([]byte(userStr), &u); err != nil {
			return nil, fmt.Errorf("parse user: %w", err)
		}
		out.User = &u
	}

	return out, nil
}

func hmacSHA256(message, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(message)
	return h.Sum(nil)
}
