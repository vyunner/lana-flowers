package webhook

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"lana-flowers-go/internal/events"
	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/telegram"
)

// Хендлер inline-кнопок (callback_query). Формат callback_data:
//
//	"offer:<id>:accept"   — продавец принимает
//	"offer:<id>:reject"   — продавец отклоняет
//	"offer:<id>:counter"  — продавец просит встречку (далее AskForCounterPrice)
//
// Любая другая команда → отвечаем «Неизвестная команда».

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
		// Зеркальное подтверждение продавцу (тому кто нажал accept) — отдельной
		// нотификацией, чтобы под рукой была кнопка «Открыть Сделки»: иначе
		// единственный фидбэк это appendStatusLine ниже на старом сообщении.
		go telegram.NotifyDealConfirmed(ctx.SellerID, ctx.BouquetTitle, ctx.Price)
		events.Default().Publish(ctx.BuyerID, events.Event{
			Type: events.TypeOfferAccepted, OfferID: offerID,
			BouquetTitle: ctx.BouquetTitle, Price: ctx.Price,
		})
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
		events.Default().Publish(ctx.BuyerID, events.Event{
			Type: events.TypeOfferRejected, OfferID: offerID,
			BouquetTitle: ctx.BouquetTitle, Price: ctx.Price,
		})
		appendStatusLine(q.Message, "\n\n❌ <b>Отклонено</b>")

	case "counter":
		_ = telegram.AnswerCallbackQuery(q.ID, "Введите встречную цену в чате", false)
		if q.Message != nil {
			telegram.AskForCounterPrice(q.Message.Chat.ID, offerID, ctx.BouquetTitle)
		}

	default:
		_ = telegram.AnswerCallbackQuery(q.ID, "Неизвестное действие", true)
	}
}

// notifyExpiredFromWebhook — DM + SSE покупателю что его pending-оффер
// на этом букете заэкспайрился (продавец принял другой оффер).
// Дублирует логику respond.go → notifyExpired, но webhook-сторона не
// может импортить handler/offers/respond (был бы цикл), так что копия.
func notifyExpiredFromWebhook(db *sql.DB, offerID int64) {
	ctx, err := offers.LoadContext(db, offerID)
	if err != nil {
		return
	}
	telegram.NotifyOfferExpired(ctx.BuyerID, ctx.BouquetTitle, ctx.Price)
	events.Default().Publish(ctx.BuyerID, events.Event{
		Type: events.TypeOfferExpired, OfferID: offerID,
		BouquetTitle: ctx.BouquetTitle, Price: ctx.Price,
	})
}

// appendStatusLine — дорисовать строку «принято/отклонено» в сообщение
// продавца, удалив inline-кнопки (передаём nil markup, тем самым стирая
// клавиатуру).
//
// Telegram даёт editMessageText для текстовых сообщений и editMessageCaption
// для photo/video — у них разные поля (text vs caption), и попытка
// отредактить чужой тип молча возвращает ошибку. Выбираем по тому, что
// Telegram прислал нам в callback: если есть Caption — это photo/video,
// иначе text.
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
