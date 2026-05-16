// Package telegram — минимальный HTTP-клиент к Bot API.
// Используется для рассылки уведомлений и обработки callback-ов из webhook'а.
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// pendingNotifs — счётчик in-flight notify-горутин. Все Notify*-функции
// inc'ают перед стартом своей работы (или вызовом в горутине) и
// dec'ают по завершению. WaitNotifications() в shutdown даёт им долететь.
var pendingNotifs sync.WaitGroup

// trackStart должна вызываться в начале каждой Notify*-функции, чтобы
// WaitNotifications мог их дождаться при graceful shutdown. Используется
// `defer trackEnd()` сразу после.
//
// Реализация в notify.go каждая функция оборачивается:
//   func NotifyXxx(...) { defer trackEnd(trackStart()); ... }
func trackStart() struct{} {
	pendingNotifs.Add(1)
	return struct{}{}
}
func trackEnd(struct{}) { pendingNotifs.Done() }

// WaitNotifications ждёт окончания всех in-flight notify-горутин до
// timeout'а. Возвращает true если всё успели, false если по таймауту
// сдались (часть нотификаций потеряется при таком завершении).
func WaitNotifications(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		pendingNotifs.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

const apiBase = "https://api.telegram.org/bot"

var httpClient = &http.Client{Timeout: 10 * time.Second}

func token() string { return os.Getenv("TELEGRAM_BOT_TOKEN") }

// ---- Inline keyboard ----

type InlineKeyboardButton struct {
	Text         string      `json:"text"`
	CallbackData string      `json:"callback_data,omitempty"`
	URL          string      `json:"url,omitempty"`
	WebApp       *WebAppInfo `json:"web_app,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// ForceReply — следующее сообщение юзера автоматически станет reply на это.
type ForceReply struct {
	ForceReply bool   `json:"force_reply"`
	Selective  bool   `json:"selective,omitempty"`
	InputField string `json:"input_field_placeholder,omitempty"`
}

// KeyboardButton — обычная кнопка-клавиатура (под полем ввода).
// Если RequestContact=true — Telegram вернёт контакт юзера в message.contact.
type KeyboardButton struct {
	Text           string `json:"text"`
	RequestContact bool   `json:"request_contact,omitempty"`
}

type ReplyKeyboardMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard,omitempty"`
	OneTimeKeyboard bool               `json:"one_time_keyboard,omitempty"`
}

// ReplyKeyboardRemove — убрать клавиатуру у юзера.
type ReplyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}

// WebAppInfo — указатель на мини-апп (URL HTTPS).
type WebAppInfo struct {
	URL string `json:"url"`
}

// ---- SendMessage ----

type SendMessageReq struct {
	ChatID      any    `json:"chat_id"`
	Text        string `json:"text"`
	ParseMode   string `json:"parse_mode,omitempty"`
	ReplyMarkup any    `json:"reply_markup,omitempty"`
}

type SentMessage struct {
	MessageID int64 `json:"message_id"`
}

func SendMessage(req SendMessageReq) (*SentMessage, error) {
	var resp struct {
		OK     bool         `json:"ok"`
		Result *SentMessage `json:"result"`
		Desc   string       `json:"description"`
	}
	if err := call("sendMessage", req, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram: %s", resp.Desc)
	}
	return resp.Result, nil
}

// ---- SendPhoto ----

// SendPhotoReq — отправка фото с caption (текстом под картинкой).
// Photo может быть URL'ом — Telegram сам скачает; либо file_id ранее
// загруженной картинки. Caption поддерживает HTML до 1024 символов.
type SendPhotoReq struct {
	ChatID      any    `json:"chat_id"`
	Photo       string `json:"photo"`
	Caption     string `json:"caption,omitempty"`
	ParseMode   string `json:"parse_mode,omitempty"`
	ReplyMarkup any    `json:"reply_markup,omitempty"`
}

func SendPhoto(req SendPhotoReq) (*SentMessage, error) {
	var resp struct {
		OK     bool         `json:"ok"`
		Result *SentMessage `json:"result"`
		Desc   string       `json:"description"`
	}
	if err := call("sendPhoto", req, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram: %s", resp.Desc)
	}
	return resp.Result, nil
}

// EditMessageText — обновить текст уже отправленного сообщения (например, чтобы убрать кнопки после ответа).
//
// Работает ТОЛЬКО для текстовых сообщений. Для photo/video используй EditMessageCaption.
func EditMessageText(chatID any, messageID int64, text string, markup *InlineKeyboardMarkup) error {
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
		return fmt.Errorf("telegram: %s", resp.Desc)
	}
	return nil
}

// EditMessageCaption — обновить подпись у photo/video сообщения.
// Аналог EditMessageText для медиа-сообщений (у них нет text, есть caption).
func EditMessageCaption(chatID any, messageID int64, caption string, markup *InlineKeyboardMarkup) error {
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"caption":    caption,
		"parse_mode": "HTML",
	}
	if markup != nil {
		body["reply_markup"] = markup
	}
	var resp struct {
		OK   bool   `json:"ok"`
		Desc string `json:"description"`
	}
	if err := call("editMessageCaption", body, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram: %s", resp.Desc)
	}
	return nil
}

// AnswerCallbackQuery — обязательный ответ на тап inline-кнопки (иначе Telegram показывает loading).
func AnswerCallbackQuery(callbackID, text string, showAlert bool) error {
	body := map[string]any{
		"callback_query_id": callbackID,
	}
	if text != "" {
		body["text"] = text
	}
	if showAlert {
		body["show_alert"] = true
	}
	var resp struct {
		OK bool `json:"ok"`
	}
	return call("answerCallbackQuery", body, &resp)
}

// SetWebhook — регистрирует webhook URL у Telegram.
func SetWebhook(url, secretToken string) error {
	body := map[string]any{
		"url":          url,
		"secret_token": secretToken,
		"allowed_updates": []string{
			"message",
			"callback_query",
		},
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

// call с retry+backoff. Telegram изредка отдаёт 429/5xx — без retry'я мы
// просто теряем уведомления (notify-функции работают через `go ...` без
// очереди). Три попытки с 0/300/900ms — на нагрузку не повлияет, но
// поднимет deliverability ~до 99% (по нашему опыту нативное окно сбоев
// у telegram-api — единицы секунд).
const maxRetries = 3

func call(method string, body any, out any) error {
	tk := token()
	if tk == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 300ms, 900ms — экспонента с базой 3.
			time.Sleep(time.Duration(attempt*attempt) * 300 * time.Millisecond)
		}

		req, err := http.NewRequest("POST", apiBase+tk+"/"+method, bytes.NewReader(buf))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue // сетевая ошибка — retry
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 5xx и 429 — retry; всё остальное (4xx, 2xx) — финал, не пытаемся снова
		// потому что Telegram уже сказал «бизнесово неверно» (например невалидный chat_id).
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			lastErr = fmt.Errorf("telegram HTTP %d", resp.StatusCode)
			continue
		}

		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w (body: %s)", err, string(respBody))
		}
		return nil
	}
	return lastErr
}
