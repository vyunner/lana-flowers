package adminbot

// Минимальные типы Telegram update'ов для админ-бота. Дублируют формы из
// internal/handler/webhook/types.go — но вынесены отдельно потому что
// адмбот изолирован, своя версия типов даёт независимый темп изменений
// (например, в админ-боте можно добавить inline_query без затрагивания
// основного бота).

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"`
}

// InlineKeyboardButton/InlineKeyboardMarkup дублируют формы из telegram
// пакета — но без WebApp/URL полей, нам в админ-боте они не нужны.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}
