package adminbot

import (
	"database/sql"
	"strconv"
)

// Notify — DM-рассылка по всем enabled админам. Текст HTML.
// Fire-and-forget: ошибки доставки одному админу не блокируют рассылку
// остальным и не возвращаются caller'у.
func Notify(db *sql.DB, eventType, text string) {
	ids, err := AdminsForEvent(db, eventType)
	if err != nil {
		logSendError("AdminsForEvent", 0, err)
		return
	}
	for _, id := range ids {
		chatID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			continue
		}
		if err := SendMessage(chatID, text, nil); err != nil {
			logSendError("Notify "+eventType, chatID, err)
		}
	}
}

// Record — комбинированный helper: append в event_log + Notify(text)
// одним вызовом. Используется во всех хук-точках основного бэка —
// чтобы caller-сторона не делала «лог + рассылка» парой строк.
//
// payload — структурированные данные события (для будущей UI-страницы
// истории), text — готовое человеческое сообщение для DM админам.
func Record(db *sql.DB, eventType string, payload any, text string) {
	LogEvent(db, eventType, payload)
	Notify(db, eventType, text)
}
