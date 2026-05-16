package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

const testToken = "1234567890:TEST-bot-token-for-unit-tests"

// makeInitData собирает валидный initData из набора пар + подписывает hash
// по Telegram-протоколу. Используется как «фабрика валидных payload'ов»
// в тестах.
func makeInitData(t *testing.T, fields map[string]string, botToken string) string {
	t.Helper()
	keys := make([]string, 0, len(fields))
	for k := range fields {
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
		sb.WriteString(fields[k])
	}
	dataCheckString := sb.String()

	mac := hmac.New(sha256.New, hmacSHA256([]byte(botToken), []byte("WebAppData")))
	mac.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(mac.Sum(nil))

	v := url.Values{}
	for k, val := range fields {
		v.Set(k, val)
	}
	v.Set("hash", hash)
	return v.Encode()
}

func TestParseAndValidate_HappyPath(t *testing.T) {
	raw := makeInitData(t, map[string]string{
		"auth_date": fmt.Sprintf("%d", time.Now().Unix()),
		"query_id":  "AAH123",
		"user":      `{"id":42,"first_name":"Vasya"}`,
	}, testToken)

	d, err := ParseAndValidate(raw, testToken, time.Hour)
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	if d.User == nil || d.User.ID != 42 || d.User.FirstName != "Vasya" {
		t.Fatalf("user not parsed: %+v", d.User)
	}
	if d.QueryID != "AAH123" {
		t.Fatalf("query_id mismatch: %q", d.QueryID)
	}
}

func TestParseAndValidate_HashUppercase(t *testing.T) {
	// Регрессия на исторический баг: hmac.Equal сравнивает байты строго.
	// Если SDK когда-нибудь пришлёт hash в верхнем регистре, валидация
	// должна всё равно пройти.
	raw := makeInitData(t, map[string]string{
		"auth_date": fmt.Sprintf("%d", time.Now().Unix()),
		"user":      `{"id":1,"first_name":"A"}`,
	}, testToken)

	v, _ := url.ParseQuery(raw)
	v.Set("hash", strings.ToUpper(v.Get("hash")))
	raw2 := v.Encode()

	if _, err := ParseAndValidate(raw2, testToken, time.Hour); err != nil {
		t.Fatalf("uppercase hash should pass: %v", err)
	}
}

func TestParseAndValidate_WrongToken(t *testing.T) {
	raw := makeInitData(t, map[string]string{
		"auth_date": fmt.Sprintf("%d", time.Now().Unix()),
		"user":      `{"id":1,"first_name":"A"}`,
	}, testToken)

	if _, err := ParseAndValidate(raw, "DIFFERENT-token", time.Hour); err == nil {
		t.Fatal("expected hash mismatch with wrong token")
	}
}

func TestParseAndValidate_TamperedField(t *testing.T) {
	raw := makeInitData(t, map[string]string{
		"auth_date": fmt.Sprintf("%d", time.Now().Unix()),
		"user":      `{"id":1,"first_name":"A"}`,
	}, testToken)

	// Подменяем user после подписи — хэш должен перестать сходиться.
	v, _ := url.ParseQuery(raw)
	v.Set("user", `{"id":2,"first_name":"Evil"}`)
	raw2 := v.Encode()

	if _, err := ParseAndValidate(raw2, testToken, time.Hour); err == nil {
		t.Fatal("expected hash mismatch after field tampering")
	}
}

func TestParseAndValidate_Expired(t *testing.T) {
	old := time.Now().Add(-25 * time.Hour).Unix()
	raw := makeInitData(t, map[string]string{
		"auth_date": fmt.Sprintf("%d", old),
		"user":      `{"id":1,"first_name":"A"}`,
	}, testToken)

	_, err := ParseAndValidate(raw, testToken, 24*time.Hour)
	if err == nil {
		t.Fatal("expected expired error")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected 'expired' in error, got %v", err)
	}
}

func TestParseAndValidate_NoMaxAge(t *testing.T) {
	// maxAge=0 → проверка возраста отключена, очень старый initData валиден.
	old := time.Now().Add(-100 * time.Hour).Unix()
	raw := makeInitData(t, map[string]string{
		"auth_date": fmt.Sprintf("%d", old),
		"user":      `{"id":1,"first_name":"A"}`,
	}, testToken)

	if _, err := ParseAndValidate(raw, testToken, 0); err != nil {
		t.Fatalf("maxAge=0 must skip age check: %v", err)
	}
}

func TestParseAndValidate_MissingHash(t *testing.T) {
	raw := "user=%7B%22id%22%3A1%7D&auth_date=1"
	if _, err := ParseAndValidate(raw, testToken, time.Hour); err == nil {
		t.Fatal("missing hash should error")
	}
}

func TestParseAndValidate_EmptyInputs(t *testing.T) {
	if _, err := ParseAndValidate("", testToken, time.Hour); err == nil {
		t.Fatal("empty initData should error")
	}
	if _, err := ParseAndValidate("foo=bar&hash=deadbeef", "", time.Hour); err == nil {
		t.Fatal("empty token should error")
	}
}
