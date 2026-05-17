package webhook

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"lana-flowers-go/internal/adminbot"
	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/telegram"
)

// Хендлеры обычных сообщений: /start (welcome), Contact (поделился номером),
// Reply на наш ForceReply (ввод встречной цены).

// handleMessage — /start и обычные сообщения.
func handleMessage(db *sql.DB, m *Message) {
	if m.From == nil {
		return
	}
	if m.Text == "/start" {
		sendWelcome(m.Chat.ID)
		tgID := strconv.FormatInt(m.From.ID, 10)
		tgName := strings.TrimSpace(m.From.FirstName + " " + m.From.LastName)

		// Проверяем — уже ли юзер в users (т.е. он не первый раз тут).
		// Помечаем в нотификации «впервые» vs «вернулся». Полезно понимать
		// сколько новых vs повторных открытий.
		var existed bool
		_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)`, tgID).Scan(&existed)
		label := "впервые"
		if existed {
			label = "вернулся"
		}

		// Язык TG-клиента — полезно для понимания аудитории.
		lang := m.From.Username // placeholder if no LanguageCode field
		_ = lang

		go adminbot.Record(db, adminbot.EventUserStarted,
			map[string]any{
				"tg_id":   tgID,
				"name":    tgName,
				"username": m.From.Username,
				"is_new":  !existed,
			},
			fmt.Sprintf(
				"👋 <b>Новый /start</b> · %s\n\n%s\nID: <code>%s</code>",
				label,
				adminbot.FormatUser(tgID, tgName, m.From.Username),
				tgID,
			),
		)
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

	// RETURNING display_name, city — забираем то что юзер ввёл на онбординге
	// (display_name и city живут в users-таблице, кладутся через
	// PATCH /users/me из мини-аппа ДО шара телефона). Этого нет в
	// EXCLUDED-сете апсерта выше, поэтому существующие значения сохранятся.
	var displayName, city string
	err := db.QueryRow(`
		INSERT INTO users (user_id, first_name, last_name, username, phone_number, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			first_name   = EXCLUDED.first_name,
			last_name    = EXCLUDED.last_name,
			username     = EXCLUDED.username,
			phone_number = EXCLUDED.phone_number,
			last_seen_at = NOW()
		RETURNING display_name, city
	`, userID, m.From.FirstName, m.From.LastName, m.From.Username, phone).Scan(&displayName, &city)
	if err != nil {
		log.Printf("save phone for %s: %v", userID, err)
		_, _ = telegram.SendMessage(telegram.SendMessageReq{
			ChatID: m.Chat.ID,
			Text:   "Ошибка при сохранении. Попробуйте ещё раз.",
		})
		return
	}

	// Phone установлен → юзер прошёл регистрацию. Берём display_name+city
	// из users (это то что юзер сам ввёл в onboarding), плюс username из
	// TG, плюс phone из contact'а.
	if displayName == "" {
		displayName = strings.TrimSpace(m.From.FirstName + " " + m.From.LastName)
	}
	if city == "" {
		city = "—"
	}
	go adminbot.Record(db, adminbot.EventUserRegistered,
		map[string]any{
			"tg_id":        userID,
			"display_name": displayName,
			"username":     m.From.Username,
			"city":         city,
			"phone":        phone,
		},
		fmt.Sprintf(
			"✅ <b>Регистрация завершена</b>\n\n"+
				"Юзер: %s\n"+
				"Имя: <b>%s</b>\n"+
				"Город: %s\n"+
				"Телефон: <code>%s</code>",
			adminbot.FormatUser(userID, displayName, m.From.Username),
			adminbot.EscapeHTML(displayName),
			adminbot.EscapeHTML(city),
			phone,
		),
	)
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
