package webhook

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/telegram"
)

// Хендлеры обычных сообщений: /start (welcome), Contact (поделился номером),
// Reply на наш ForceReply (ввод встречной цены).

// handleMessage — /start и обычные сообщения.
//
// _ db / m.From сейчас не нужны: единственная команда — /start, без
// привязки к user_id (welcome не персонализирован). Если когда-то добавим
// /myorders или persistent state — вернём userID.
func handleMessage(_ *sql.DB, m *Message) {
	if m.From == nil {
		return
	}
	if m.Text == "/start" {
		sendWelcome(m.Chat.ID)
	}
}

// sendWelcome — единое приветствие в боте: текст + кнопка открыть мини-апп.
// Используется на /start. Регистрация (телефон/имя/аватар) — целиком в
// мини-аппе через WebApp.requestContact + PATCH /users/me.
func sendWelcome(chatID int64) {
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: "🛍 Открыть каталог", WebApp: &telegram.WebAppInfo{URL: miniAppURL}}},
		},
	}
	_, err := telegram.SendMessage(telegram.SendMessageReq{
		ChatID:      chatID,
		Text:        "🌸 Добро пожаловать в <b>Lana Flowers</b>!",
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})
	if err != nil {
		log.Printf("sendWelcome: %v", err)
	}
}

// handleContact — юзер тапнул «Поделиться номером». Сохраняем номер в
// users-таблицу и больше ничего не делаем — мини-апп сам поллит /users/me
// и продолжит онбординг-флоу.
func handleContact(db *sql.DB, m *Message) {
	if m.From == nil || m.Contact == nil {
		return
	}
	userID := strconv.FormatInt(m.From.ID, 10)

	// Защита: контакт должен быть СВОИМ (нельзя зарегистрировать чужой телефон).
	if m.Contact.UserID != 0 && m.Contact.UserID != m.From.ID {
		_, _ = telegram.SendMessage(telegram.SendMessageReq{
			ChatID: m.Chat.ID,
			Text:   "Можно поделиться только своим номером.",
		})
		return
	}

	phone := strings.TrimSpace(m.Contact.PhoneNumber)
	if phone == "" {
		return
	}

	_, err := db.Exec(`
		INSERT INTO users (user_id, first_name, last_name, username, phone_number, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			first_name   = EXCLUDED.first_name,
			last_name    = EXCLUDED.last_name,
			username     = EXCLUDED.username,
			phone_number = EXCLUDED.phone_number,
			last_seen_at = NOW()
	`, userID, m.From.FirstName, m.From.LastName, m.From.Username, phone)
	if err != nil {
		log.Printf("save phone for %s: %v", userID, err)
		_, _ = telegram.SendMessage(telegram.SendMessageReq{
			ChatID: m.Chat.ID,
			Text:   "Ошибка при сохранении. Попробуйте ещё раз.",
		})
	}
}

// counterRe — регексп для извлечения offer_id из reply-контекста.
// AskForCounterPrice оставляет «#N» в конце сообщения как технический
// маркер, чтобы handleReply мог восстановить какому офферу адресован ввод.
var counterRe = regexp.MustCompile(`#(\d+)`)

// handleReply — юзер ответил на наш ForceReply «Укажите встречную цену».
// Извлекаем offer_id из текста ОРИГИНАЛЬНОГО сообщения (через counterRe),
// парсим число из ответа, создаём counter-оффер, рассылаем нотификации.
func handleReply(db *sql.DB, m *Message) {
	if m.From == nil || m.ReplyToMessage == nil {
		return
	}

	matches := counterRe.FindStringSubmatch(m.ReplyToMessage.Text)
	if len(matches) < 2 {
		return
	}
	offerID, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return
	}

	priceStr := stripNonDigits(m.Text)
	if priceStr == "" {
		_, _ = telegram.SendMessage(telegram.SendMessageReq{
			ChatID: m.Chat.ID,
			Text:   "Не понял число. Пришли цену в тенге, например: 11500",
		})
		return
	}
	price, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil || price <= 0 {
		_, _ = telegram.SendMessage(telegram.SendMessageReq{
			ChatID: m.Chat.ID,
			Text:   "Цена должна быть положительным числом",
		})
		return
	}

	userID := strconv.FormatInt(m.From.ID, 10)

	origCtx, err := offers.LoadContext(db, offerID)
	if err != nil {
		_, _ = telegram.SendMessage(telegram.SendMessageReq{
			ChatID: m.Chat.ID,
			Text:   "Оффер не найден или уже закрыт",
		})
		return
	}

	newID, err := offers.CounterOffer(db, offerID, userID, price)
	if err != nil {
		_, _ = telegram.SendMessage(telegram.SendMessageReq{
			ChatID: m.Chat.ID,
			Text:   "Ошибка: " + err.Error(),
		})
		return
	}

	_, _ = telegram.SendMessage(telegram.SendMessageReq{
		ChatID:    m.Chat.ID,
		Text:      fmt.Sprintf("🔄 Встречное предложение <b>%d ₸</b> отправлено покупателю", price),
		ParseMode: "HTML",
	})

	go telegram.NotifyOfferCountered(
		origCtx.BuyerID, newID, origCtx.BouquetID, origCtx.BouquetTitle, origCtx.BouquetPhoto,
		origCtx.Price, price,
		origCtx.SellerName,
	)
	events.Default().Publish(origCtx.BuyerID, events.Event{
		Type: events.TypeOfferCountered, OfferID: newID,
		BouquetTitle: origCtx.BouquetTitle, Price: price,
	})
}

// stripNonDigits оставляет только цифры — для парса цены, юзер мог
// написать «11 500», «11500 ₸», «~11500» итд.
func stripNonDigits(s string) string {
	var sb strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}
