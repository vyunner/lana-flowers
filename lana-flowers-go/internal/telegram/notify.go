package telegram

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// formatPrice превращает 12500 → "12 500".
func formatPrice(n int64) string {
	s := strconv.FormatInt(n, 10)
	// Вставляем пробел каждые 3 цифры справа.
	if len(s) <= 3 {
		return s
	}
	var sb strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		sb.WriteString(s[:pre])
		if len(s) > pre {
			sb.WriteByte(' ')
		}
	}
	for i := pre; i < len(s); i += 3 {
		sb.WriteString(s[i : i+3])
		if i+3 < len(s) {
			sb.WriteByte(' ')
		}
	}
	return sb.String()
}

// miniAppButton — кнопка-deeplink в мини-апп на конкретный экран.
// screen: "catalog" | "deals" | "profile" — должен совпадать с парсером
// ?screen=... в App.vue. WebApp inline-кнопки работают только в DM с ботом
// (а нотификации именно туда и шлются), что нам подходит.
func miniAppButton(text, screen string) InlineKeyboardButton {
	base := os.Getenv("WEBAPP_URL")
	if base == "" {
		base = "https://lana-flowers.vercel.app"
	}
	return InlineKeyboardButton{
		Text:   text,
		WebApp: &WebAppInfo{URL: base + "/?screen=" + screen},
	}
}

// sendPhotoOrText — общая отправка с фоткой, фолбэк на текст. Если sendPhoto
// упал (Telegram не смог скачать картинку), отправляем без превью — лучше
// доставить нотификацию без фото, чем потерять её совсем.
func sendPhotoOrText(chatID int64, photoURL, text string, markup *InlineKeyboardMarkup) error {
	if photoURL != "" {
		_, err := SendPhoto(SendPhotoReq{
			ChatID:      chatID,
			Photo:       photoURL,
			Caption:     text,
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		if err == nil {
			return nil
		}
		log.Printf("notify: sendPhoto failed (%v), falling back to text", err)
	}
	_, err := SendMessage(SendMessageReq{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})
	return err
}

// sendStructured — HTML-сообщение с опциональной разметкой кнопок. Единый
// ParseMode и лог-формат для всех итоговых уведомлений (без фото).
func sendStructured(chatID int64, text string, markup *InlineKeyboardMarkup, tag string) {
	if _, err := SendMessage(SendMessageReq{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	}); err != nil {
		log.Printf("notify %s chat=%d: %v", tag, chatID, err)
	}
}

// ---- Формат сообщений ----
//
// Все нотификации построены одинаково:
//
//   <emoji> <Заголовок-действие>
//
//   <b>Название букета</b>
//
//   Метка: <b>Значение</b>
//   Метка: Значение
//   ...
//
//   [Инструкция/что делать дальше]
//
//   [Inline-кнопки]
//
// Метки слева, значения справа — глаз сканирует один столбец и сразу
// понимает что есть что. Эмодзи в заголовке статусный, ровно один.
// Декоративных эмодзи в строках полей нет.

// NotifyNewOffer — продавцу: пришёл новый оффер. Photo + callback-кнопки.
//
// callback_data формат: "offer:<id>:accept" / ":reject" / ":counter"
// @username покупателя не показываем — приватные данные.
func NotifyNewOffer(sellerTGID string, offerID, bouquetID int64, bouquetTitle, bouquetPhoto string, sellerPrice, offerPrice int64, buyerName string) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(sellerTGID, 10, 64)
	if err != nil {
		log.Printf("notify: bad sellerTGID %q: %v", sellerTGID, err)
		return
	}

	text := fmt.Sprintf(
		"💌 Новое предложение\n\n<b>%s</b>\n\nЦена: <b>%s ₸</b>\nВаша: %s ₸",
		escapeHTML(bouquetTitle),
		formatPrice(offerPrice),
		formatPrice(sellerPrice),
	)
	if buyerName != "" {
		text += "\nОт: " + escapeHTML(buyerName)
	}

	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("✅ Принять %s ₸", formatPrice(offerPrice)), CallbackData: fmt.Sprintf("offer:%d:accept", offerID)},
			},
			{
				// «Встречно» → бот шлёт ForceReply «Введите сумму…», юзер
				// просто пишет число в чат. Без deep-link в app.
				{Text: "🔄 Встречно", CallbackData: fmt.Sprintf("offer:%d:counter", offerID)},
				{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("offer:%d:reject", offerID)},
			},
		},
	}

	if err := sendPhotoOrText(chatID, bouquetPhoto, text, markup); err != nil {
		log.Printf("notify NewOffer chat=%d: %v", chatID, err)
	}
}

// NotifyOfferCountered — контрагенту: пришла встречная цена. Photo + callback-кнопки.
//
// @username продавца не показываем — приватные данные.
func NotifyOfferCountered(buyerTGID string, newOfferID, bouquetID int64, bouquetTitle, bouquetPhoto string, oldPrice, newPrice int64, sellerName string) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}

	text := fmt.Sprintf(
		"🔄 Встречная цена\n\n<b>%s</b>\n\nНовая цена: <b>%s ₸</b>\nВы предлагали: %s ₸",
		escapeHTML(bouquetTitle),
		formatPrice(newPrice),
		formatPrice(oldPrice),
	)
	if sellerName != "" {
		text += "\nОт: " + escapeHTML(sellerName)
	}

	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("✅ Принять %s ₸", formatPrice(newPrice)), CallbackData: fmt.Sprintf("offer:%d:accept", newOfferID)},
			},
			{
				{Text: "🔄 Встречно", CallbackData: fmt.Sprintf("offer:%d:counter", newOfferID)},
				{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("offer:%d:reject", newOfferID)},
			},
		},
	}

	if err := sendPhotoOrText(chatID, bouquetPhoto, text, markup); err != nil {
		log.Printf("notify Countered chat=%d: %v", chatID, err)
	}
}

// NotifyOfferAccepted — покупателю: его оффер принят. + CTA в Сделки.
//
// Контакт продавца в текст НЕ кладём — он раскрывается только в Сделках
// (там же, где договариваются о встрече).
func NotifyOfferAccepted(buyerTGID string, bouquetTitle string, finalPrice int64, sellerName string) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}
	_ = sellerName // имя контрагента уже доступно в Сделках, в DM избыточно

	text := fmt.Sprintf(
		"✅ Предложение принято\n\n<b>%s</b>\n\nЦена: <b>%s ₸</b>\n\nДоговоритесь о деталях с продавцом в Сделках.",
		escapeHTML(bouquetTitle),
		formatPrice(finalPrice),
	)
	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{miniAppButton("💬 Открыть Сделки", "deals")},
		},
	}
	sendStructured(chatID, text, markup, "Accepted")
}

// NotifyOfferRejected — покупателю: его оффер отклонён. + CTA в каталог.
func NotifyOfferRejected(buyerTGID string, bouquetTitle string, offeredPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"❌ Предложение отклонено\n\n<b>%s</b>\n\nВаша цена: <b>%s ₸</b>\n\nПопробуйте другую цену или посмотрите похожие букеты.",
		escapeHTML(bouquetTitle),
		formatPrice(offeredPrice),
	)
	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{miniAppButton("🔍 Похожие букеты", "catalog")},
		},
	}
	sendStructured(chatID, text, markup, "Rejected")
}

// NotifyOfferWithdrawn — продавцу: покупатель отозвал свой pending до того,
// как продавец на него ответил. Сделки не было, букет с продажи не уходил —
// поэтому формулировка отличается от NotifyDealCancelled. Кнопки нет:
// букет остался активным в каталоге, продавец и так в курсе.
func NotifyOfferWithdrawn(sellerTGID string, bouquetTitle string, offeredPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(sellerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"↩️ Покупатель отозвал предложение\n\n<b>%s</b>\n\nЦена: <b>%s ₸</b>\n\nБукет остаётся в продаже.",
		escapeHTML(bouquetTitle),
		formatPrice(offeredPrice),
	)
	sendStructured(chatID, text, nil, "OfferWithdrawn")
}

// NotifyDealCancelled — одна из сторон отменила уже принятую сделку.
// Получатель — противоположная сторона. recipientIsBuyer определяет тон и CTA:
//   - получатель=продавец (отменил buyer): «Покупатель передумал…» + «Мои букеты»
//   - получатель=покупатель (отменил seller): «Продавец отменил…» + «Похожие букеты»
func NotifyDealCancelled(toTGID string, bouquetTitle string, finalPrice int64, recipientIsBuyer bool) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(toTGID, 10, 64)
	if err != nil {
		return
	}

	var body string
	var btn InlineKeyboardButton
	if recipientIsBuyer {
		body = "Продавец отменил сделку. Посмотрите похожие букеты."
		btn = miniAppButton("🔍 Похожие букеты", "catalog")
	} else {
		body = "Покупатель передумал. Букет снова в каталоге."
		btn = miniAppButton("🛒 Мои букеты", "profile")
	}

	text := fmt.Sprintf(
		"⚠️ Сделка отменена\n\n<b>%s</b>\n\nЦена: <b>%s ₸</b>\n\n%s",
		escapeHTML(bouquetTitle),
		formatPrice(finalPrice),
		body,
	)
	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{{btn}},
	}
	sendStructured(chatID, text, markup, "DealCancelled")
}

// NotifyOfferExpired — покупателю: его pending заэкспайрился, потому что
// продавец принял другое предложение на тот же букет. + CTA в каталог.
func NotifyOfferExpired(buyerTGID, bouquetTitle string, offerPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"⏱ Букет ушёл другому покупателю\n\n<b>%s</b>\n\nВаше предложение: <b>%s ₸</b>\n\nПредложение отменено. Посмотрите похожие.",
		escapeHTML(bouquetTitle),
		formatPrice(offerPrice),
	)
	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{miniAppButton("🔍 Похожие букеты", "catalog")},
		},
	}
	sendStructured(chatID, text, markup, "Expired")
}

// AskForCounterPrice — после тапа "Встречно" просим юзера ввести сумму через ForceReply.
//
// В тексте ОБЯЗАТЕЛЬНО оставляем «#N» в конце — handleReply парсит его
// regex'ом `#(\d+)` чтобы восстановить offer_id из reply-контекста. Без
// этого не поймём какому офферу цена адресована.
func AskForCounterPrice(chatID any, offerID int64, bouquetTitle string) {
	text := fmt.Sprintf("Укажите встречную цену за «%s». #%d", bouquetTitle, offerID)
	markup := &ForceReply{
		ForceReply: true,
		Selective:  true,
		InputField: "Например: 11500",
	}
	if _, err := SendMessage(SendMessageReq{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: markup,
	}); err != nil {
		log.Printf("ask counter: %v", err)
	}
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
