package webhook

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/telegram"

	"github.com/gin-gonic/gin"
)

const miniAppURL = "https://lana-flowers.vercel.app"

// webhookSecret кэшируется на старте — в проде должен быть задан, иначе
// /tg/webhook открыт миру и любой может слать поддельные callback_query
// от лица любого user_id.
var webhookSecret string

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	webhookSecret = os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	if webhookSecret == "" && os.Getenv("APP_ENV") == "production" {
		log.Fatal("TELEGRAM_WEBHOOK_SECRET is required in production (.env)")
	}
	if webhookSecret == "" {
		log.Print("WARN: TELEGRAM_WEBHOOK_SECRET is empty — webhook unauthenticated (OK only in development)")
	}
	r.POST("/tg/webhook", func(c *gin.Context) { Handle(c, db) })
}

// Update — минимальный набор полей из Telegram update.
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

func Handle(c *gin.Context, db *sql.DB) {
	// Если secret задан — header обязан совпадать. Если не задан (только в
	// dev по логу выше) — пропускаем без проверки.
	if webhookSecret != "" && c.GetHeader("X-Telegram-Bot-Api-Secret-Token") != webhookSecret {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var u Update
	if err := c.ShouldBindJSON(&u); err != nil {
		log.Printf("webhook: bad json: %v", err)
		c.Status(http.StatusOK)
		return
	}

	switch {
	case u.CallbackQuery != nil:
		handleCallback(db, u.CallbackQuery)
	case u.Message != nil && u.Message.Contact != nil:
		handleContact(db, u.Message)
	case u.Message != nil && u.Message.ReplyToMessage != nil:
		handleReply(db, u.Message)
	case u.Message != nil:
		handleMessage(db, u.Message)
	}

	c.Status(http.StatusOK)
}

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
// Используется и на /start, и после привязки номера. Регистрация (телефон/имя/аватар)
// — целиком в мини-аппе через WebApp.requestContact + PATCH /users/me.
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

// handleContact — юзер тапнул «Поделиться номером». Сохраняем + welcome.
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
		return
	}

	// Номер сохранён. Если юзер шарил через WebApp.requestContact — он сидит в мини-аппе,
	// никакого подтверждения слать не надо (там сам поллит /users/me и продолжит флоу).
}

// handleCallback — нажатие inline-кнопки. callback_data = "offer:<id>:<action>".
func handleCallback(db *sql.DB, q *CallbackQuery) {
	log.Printf("callback: data=%q from=%v", q.Data, q.From)
	parts := strings.Split(q.Data, ":")
	if len(parts) < 3 || parts[0] != "offer" {
		_ = telegram.AnswerCallbackQuery(q.ID, "Неизвестная команда", false)
		return
	}

	offerID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		_ = telegram.AnswerCallbackQuery(q.ID, "Битый ID оффера", true)
		return
	}
	action := parts[2]

	if q.From == nil {
		return
	}
	userID := strconv.FormatInt(q.From.ID, 10)

	ctx, err := offers.LoadContext(db, offerID)
	if err != nil {
		_ = telegram.AnswerCallbackQuery(q.ID, "Оффер не найден", true)
		return
	}

	if ctx.SellerID != userID {
		_ = telegram.AnswerCallbackQuery(q.ID, "Это не ваш оффер", true)
		return
	}

	if ctx.Status != "pending" {
		_ = telegram.AnswerCallbackQuery(q.ID, "Уже обработан", true)
		if q.Message != nil {
			_ = telegram.EditMessageText(q.Message.Chat.ID, q.Message.MessageID,
				q.Message.Text+"\n\n— уже обработано —", nil)
		}
		return
	}

	switch action {
	case "accept":
		expired, err := offers.AcceptOffer(db, offerID, userID)
		if err != nil {
			_ = telegram.AnswerCallbackQuery(q.ID, "Ошибка: "+err.Error(), true)
			return
		}
		_ = telegram.AnswerCallbackQuery(q.ID, "✅ Принято", false)
		go telegram.NotifyOfferAccepted(ctx.BuyerID, ctx.BouquetTitle, ctx.Price,
			ctx.SellerName)
		for _, id := range expired {
			go notifyExpiredFromWebhook(db, id)
		}
		appendStatusLine(q.Message, fmt.Sprintf("\n\n✅ <b>Принято за %d ₸</b>", ctx.Price))

	case "reject":
		if err := offers.RejectOffer(db, offerID, userID); err != nil {
			_ = telegram.AnswerCallbackQuery(q.ID, "Ошибка: "+err.Error(), true)
			return
		}
		_ = telegram.AnswerCallbackQuery(q.ID, "❌ Отклонено", false)
		go telegram.NotifyOfferRejected(ctx.BuyerID, ctx.BouquetTitle, ctx.Price)
		appendStatusLine(q.Message, "\n\n❌ <b>Отклонено</b>")

	case "counter":
		_ = telegram.AnswerCallbackQuery(q.ID, "Введите встречную цену в чате", false)
		if q.Message != nil {
			telegram.AskForCounterPrice(q.Message.Chat.ID, offerID, ctx.Price)
		}

	default:
		_ = telegram.AnswerCallbackQuery(q.ID, "Неизвестное действие", true)
	}
}

// handleReply — юзер ответил на сообщение бота. Используется для ввода встречной цены.
var counterRe = regexp.MustCompile(`#(\d+)`)

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

	// m.Text — это обычно просто число «11500», смысла хранить в БД-поле
	// `message` нет (получается бессмысленная запись с цифрами). Если
	// юзер захочет писать комментарий — добавим UI-поле в OfferSheet и
	// прокинем через respondOffer, тогда message будет осмысленный.
	newID, err := offers.CounterOffer(db, offerID, userID, price, "")
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
		origCtx.SellerName, "", // см. CounterOffer выше — message пустой
	)
}

func stripNonDigits(s string) string {
	var sb strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}

// notifyExpiredFromWebhook — DM покупателю что его pending-оффер на этом букете
// заэкспайрился (продавец принял другой оффер). Тот же смысл что
// offers.notifyExpired, но в webhook'е нет к нему доступа (private).
func notifyExpiredFromWebhook(db *sql.DB, offerID int64) {
	ctx, err := offers.LoadContext(db, offerID)
	if err != nil {
		return
	}
	telegram.NotifyOfferExpired(ctx.BuyerID, ctx.BouquetTitle, ctx.Price)
}

// appendStatusLine — дорисовать строку «принято/отклонено» в сообщение продавца,
// удалив inline-кнопки (передаём nil markup, тем самым стирая клавиатуру).
//
// Telegram даёт editMessageText для текстовых сообщений и editMessageCaption
// для photo/video — у них разные поля (text vs caption), и попытка отредактить
// чужой тип молча возвращает ошибку. Выбираем по тому, что Telegram прислал
// нам в callback: если есть Caption — это photo/video, иначе text.
func appendStatusLine(m *Message, line string) {
	if m == nil {
		return
	}
	if m.Caption != "" {
		newCap := m.Caption + line
		if err := telegram.EditMessageCaption(m.Chat.ID, m.MessageID, newCap, nil); err != nil {
			log.Printf("appendStatusLine caption: %v", err)
		}
		return
	}
	newText := m.Text + line
	if err := telegram.EditMessageText(m.Chat.ID, m.MessageID, newText, nil); err != nil {
		log.Printf("appendStatusLine text: %v", err)
	}
}
