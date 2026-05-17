package telegram

import "fmt"

// Тонкие обёртки над Bot API методами. Все возвращают (*SentMessage, error)
// или (error) и используют общий call() с retry.

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

// ---- EditMessageText / Caption ----

// EditMessageText — обновить текст уже отправленного сообщения
// (например, чтобы убрать кнопки после ответа). Работает ТОЛЬКО для
// текстовых сообщений. Для photo/video используй EditMessageCaption.
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

// ---- Callback / Webhook admin ----

// AnswerCallbackQuery — обязательный ответ на тап inline-кнопки
// (иначе Telegram показывает loading-индикатор на кнопке вечно).
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
