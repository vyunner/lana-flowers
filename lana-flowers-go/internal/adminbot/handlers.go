package adminbot

import (
	"database/sql"
	"log"
	"strconv"
	"strings"
)

// handleUpdate — корневой диспетчер update'а. Зовётся из routes.go.
// Молча игнорирует update'ы которые нам не интересны (не падаем, не
// шумим — Telegram повторит при ошибке).
func handleUpdate(db *sql.DB, u *Update) {
	switch {
	case u.CallbackQuery != nil:
		handleCallback(db, u.CallbackQuery)
	case u.Message != nil:
		handleMessage(db, u.Message)
	}
}

// handleMessage — единственная поддерживаемая команда: /start.
// Любые другие сообщения игнорируем (админ-бот общается через кнопки).
func handleMessage(db *sql.DB, m *Message) {
	if m.From == nil {
		return
	}
	if m.Text != "/start" {
		return
	}
	handleStart(db, m)
}

// handleStart — две ветки:
//
//	юзер уже админ → показываем главное меню
//	юзер не админ → записываем заявку в admin_requests + краткий ответ
func handleStart(db *sql.DB, m *Message) {
	tgID := strconv.FormatInt(m.From.ID, 10)

	isAdmin, err := IsAdmin(db, tgID)
	if err != nil {
		log.Printf("adminbot IsAdmin %s: %v", tgID, err)
		_ = SendMessage(m.Chat.ID, "Внутренняя ошибка, попробуйте позже.", nil)
		return
	}

	if isAdmin {
		_ = SendMessage(m.Chat.ID, mainText, mainMenu())
		return
	}

	// Не админ — пишем в заявки и сообщаем что отправили на рассмотрение.
	// Имя берём из first_name + last_name (display_name тут нет — это
	// другой бот, у него своя БД-связь с юзерами нет).
	displayName := strings.TrimSpace(m.From.FirstName + " " + m.From.LastName)
	if err := AddRequest(db, tgID, displayName, m.From.Username); err != nil {
		log.Printf("adminbot AddRequest %s: %v", tgID, err)
	}
	_ = SendMessage(
		m.Chat.ID,
		"<b>Доступ ограничен.</b>\n\nЗаявка отправлена. Если её одобрят, "+
			"вернитесь и нажмите /start ещё раз.",
		nil,
	)
}

// handleCallback — роутер по callback_data. Формат строк описан в menu.go.
// На любую неизвестную data — silently ack чтобы у юзера loading
// перестал крутиться.
func handleCallback(db *sql.DB, q *CallbackQuery) {
	if q.From == nil || q.Message == nil {
		_ = AnswerCallbackQuery(q.ID, "", false)
		return
	}
	tgID := strconv.FormatInt(q.From.ID, 10)

	// Все callback'и требуют админ-прав. Не админ — отвечаем «нет доступа».
	a, err := GetAdmin(db, tgID)
	if err != nil {
		log.Printf("adminbot GetAdmin %s: %v", tgID, err)
		_ = AnswerCallbackQuery(q.ID, "Ошибка", true)
		return
	}
	if a == nil {
		_ = AnswerCallbackQuery(q.ID, "Нет доступа", true)
		return
	}

	data := q.Data
	switch {
	case data == cbMain:
		_ = EditMessageText(q.Message.Chat.ID, q.Message.MessageID, mainText, mainMenu())
		_ = AnswerCallbackQuery(q.ID, "", false)

	case data == cbStats:
		s, err := GetStats(db)
		if err != nil {
			log.Printf("adminbot GetStats: %v", err)
			_ = AnswerCallbackQuery(q.ID, "Ошибка", true)
			return
		}
		_ = EditMessageText(q.Message.Chat.ID, q.Message.MessageID, statsText(s), statsMenu())
		_ = AnswerCallbackQuery(q.ID, "", false)

	case data == cbNotif:
		_ = EditMessageText(q.Message.Chat.ID, q.Message.MessageID, notifText, notifMenu(a))
		_ = AnswerCallbackQuery(q.ID, "", false)

	case strings.HasPrefix(data, cbNotifToggle):
		ev := strings.TrimPrefix(data, cbNotifToggle)
		newEnabled, err := ToggleEvent(db, tgID, ev)
		if err != nil {
			log.Printf("adminbot ToggleEvent %s %s: %v", tgID, ev, err)
			_ = AnswerCallbackQuery(q.ID, "Ошибка", true)
			return
		}
		// Перечитываем админа чтобы свежие настройки попали в меню.
		a2, _ := GetAdmin(db, tgID)
		_ = EditMessageText(q.Message.Chat.ID, q.Message.MessageID, notifText, notifMenu(a2))
		ackText := "Выключено"
		if newEnabled {
			ackText = "Включено"
		}
		_ = AnswerCallbackQuery(q.ID, ackText, false)

	case data == cbAccess:
		showAccessMenu(db, q.Message.Chat.ID, q.Message.MessageID)
		_ = AnswerCallbackQuery(q.ID, "", false)

	case strings.HasPrefix(data, cbAccessGrant):
		target := strings.TrimPrefix(data, cbAccessGrant)
		if err := AddAdmin(db, target, tgID); err != nil {
			log.Printf("adminbot AddAdmin %s: %v", target, err)
			_ = AnswerCallbackQuery(q.ID, "Ошибка", true)
			return
		}
		_ = RemoveRequest(db, target)
		// Уведомим самого юзера что доступ получен.
		if targetID, err := strconv.ParseInt(target, 10, 64); err == nil {
			_ = SendMessage(targetID,
				"✅ Вам выдан доступ к админ-панели. Нажмите /start.", nil)
		}
		showAccessMenu(db, q.Message.Chat.ID, q.Message.MessageID)
		_ = AnswerCallbackQuery(q.ID, "Доступ выдан", false)

	case strings.HasPrefix(data, cbAccessDeny):
		target := strings.TrimPrefix(data, cbAccessDeny)
		_ = RemoveRequest(db, target)
		showAccessMenu(db, q.Message.Chat.ID, q.Message.MessageID)
		_ = AnswerCallbackQuery(q.ID, "Отказано", false)

	default:
		// "m:noop" и неизвестные — просто ack.
		_ = AnswerCallbackQuery(q.ID, "", false)
	}
}

func showAccessMenu(db *sql.DB, chatID, messageID int64) {
	requests, err := ListRequests(db)
	if err != nil {
		log.Printf("adminbot ListRequests: %v", err)
		_ = EditMessageText(chatID, messageID, "Ошибка загрузки заявок.", mainMenu())
		return
	}
	text := accessText
	if len(requests) == 0 {
		text = accessEmptyText()
	}
	_ = EditMessageText(chatID, messageID, text, accessMenu(requests))
}
