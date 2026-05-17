package telegram

// Типы клавиатур и интерактивных элементов Telegram Bot API.
// Inline-кнопки прикрепляются к сообщениям, Reply-клавиатура заменяет
// системную клавиатуру юзера, ForceReply форсит ответ-режим.

// ---- Inline keyboard (кнопки под сообщением) ----

// InlineKeyboardButton — одна inline-кнопка. Заполнить ОДИН из:
// CallbackData (handler в webhook ловит тап), URL (открыть в браузере)
// или WebApp (открыть мини-апп прямо из чата). Остальные null.
type InlineKeyboardButton struct {
	Text         string      `json:"text"`
	CallbackData string      `json:"callback_data,omitempty"`
	URL          string      `json:"url,omitempty"`
	WebApp       *WebAppInfo `json:"web_app,omitempty"`
}

// InlineKeyboardMarkup — сетка inline-кнопок. Внешний slice — ряды,
// внутренний — кнопки в ряду.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// WebAppInfo — указатель на мини-апп (URL HTTPS).
type WebAppInfo struct {
	URL string `json:"url"`
}

// ---- ForceReply ----

// ForceReply — следующее сообщение юзера автоматически станет reply на
// сообщение бота с этим markup'ом. Используется для приёма текстового
// ввода (например, цены) без отдельной inline-клавиатуры.
type ForceReply struct {
	ForceReply bool   `json:"force_reply"`
	Selective  bool   `json:"selective,omitempty"`
	InputField string `json:"input_field_placeholder,omitempty"`
}

// ---- Reply keyboard (обычная клавиатура под полем ввода) ----

// KeyboardButton — обычная кнопка-клавиатура. Если RequestContact=true
// — Telegram вернёт контакт юзера в message.contact.
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
