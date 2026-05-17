package adminbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Минимальный HTTP-клиент к Bot API для админ-бота. Изолирован от
// internal/telegram (тот пакет использует TELEGRAM_BOT_TOKEN, мы тут —
// ADMIN_BOT_TOKEN). Дубль части кода терпимо: интерфейс маленький
// (SendMessage / EditMessageText / AnswerCallbackQuery / SetWebhook),
// меняется редко.

const apiBase = "https://api.telegram.org/bot"

var httpClient = &http.Client{Timeout: 10 * time.Second}

func token() string { return os.Getenv("ADMIN_BOT_TOKEN") }

// call с одной попыткой — для админ-бота нет retry-логики основного
// telegram пакета. Админских event'ов мало, потеря одного DM не
// критична, лучше быстро вернуть ошибку и сложить в лог чем городить
// очередь retry'ев.
func call(method string, body any, out any) error {
	tk := token()
	if tk == "" {
		return fmt.Errorf("ADMIN_BOT_TOKEN not set")
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", apiBase+tk+"/"+method, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode: %w (body: %s)", err, string(respBody))
		}
	}
	return nil
}

// SendMessage — отправить HTML-сообщение. markup может быть nil.
func SendMessage(chatID int64, text string, markup *InlineKeyboardMarkup) error {
	body := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if markup != nil {
		body["reply_markup"] = markup
	}
	var resp struct {
		OK   bool   `json:"ok"`
		Desc string `json:"description"`
	}
	if err := call("sendMessage", body, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram: %s", resp.Desc)
	}
	return nil
}

// EditMessageText — обновить текст/клавиатуру существующего сообщения
// (используется в меню чтобы при тапе на toggle страница не плодилась).
func EditMessageText(chatID int64, messageID int64, text string, markup *InlineKeyboardMarkup) error {
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if markup != nil {
		body["reply_markup"] = markup
	}
	var resp struct {
		OK   bool   `json:"ok"`
		Desc string `json:"description"`
	}
	if err := call("editMessageText", body, &resp); err != nil {
		return err
	}
	if !resp.OK {
		// «message is not modified» — норм, бывает если юзер дважды тапнул
		// одну и ту же кнопку. Не считаем ошибкой.
		if resp.Desc == "Bad Request: message is not modified" {
			return nil
		}
		return fmt.Errorf("telegram: %s", resp.Desc)
	}
	return nil
}

// AnswerCallbackQuery — обязательный ответ на тап inline-кнопки
// (иначе Telegram показывает loading-индикатор на кнопке вечно).
func AnswerCallbackQuery(callbackID, text string, alert bool) error {
	body := map[string]any{"callback_query_id": callbackID}
	if text != "" {
		body["text"] = text
	}
	if alert {
		body["show_alert"] = true
	}
	var resp struct {
		OK bool `json:"ok"`
	}
	return call("answerCallbackQuery", body, &resp)
}

// SetWebhook — регистрирует webhook URL у Telegram. Вызывается из CLI
// `./app admin-webhook` один раз при первичной настройке бота.
func SetWebhook(url, secret string) error {
	body := map[string]any{
		"url":             url,
		"secret_token":    secret,
		"allowed_updates": []string{"message", "callback_query"},
	}
	var resp struct {
		OK   bool   `json:"ok"`
		Desc string `json:"description"`
	}
	if err := call("setWebhook", body, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram: %s", resp.Desc)
	}
	return nil
}

// logSendError — обёртка для fire-and-forget Notify, чтобы ошибку
// доставки одного DM не теряли молча, но и не возвращали наверх (хуки
// в основном коде не должны падать из-за админ-бота).
func logSendError(tag string, chatID int64, err error) {
	if err != nil {
		log.Printf("adminbot %s chat=%d: %v", tag, chatID, err)
	}
}
