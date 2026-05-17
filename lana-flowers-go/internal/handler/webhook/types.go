package webhook

// Минимальный набор полей из Telegram update'ов, которые мы реально
// читаем. Полная схема Bot API больше — но дописываем по мере того как
// что-то понадобится, чтобы json.Unmarshal не падал на неизвестных полях
// (он их просто игнорирует) и чтобы наш код не зависел от полной модели.

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	MessageID      int64    `json:"message_id"`
	From           *User    `json:"from"`
	Chat           Chat     `json:"chat"`
	Text           string   `json:"text"`
	Caption        string   `json:"caption,omitempty"` // для photo/video — текст под медиа
	Contact        *Contact `json:"contact,omitempty"`
	ReplyToMessage *Message `json:"reply_to_message,omitempty"`
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

type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	UserID      int64  `json:"user_id"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"`
}
