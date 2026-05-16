package telegram

import (
	"fmt"
	"log"
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

// NotifyNewOffer — новый оффер от покупателя продавцу.
//
// callback_data формат: "offer:<id>:accept" / ":reject" / ":counter"
//
// Внимание: @username / t.me-ссылки на юзеров НЕ показываем. Это приватные
// данные, продавцу хватает имени. Если нужен контакт — он зашит в Telegram-ID
// и в БД-телефоне, но в текст ботом не льём.
//
// bouquetPhoto — URL первой картинки. Если пустой, шлём просто текст;
// иначе картинка с caption (так выглядит как нормальная карточка маркетплейса).
func NotifyNewOffer(sellerTGID string, offerID, bouquetID int64, bouquetTitle, bouquetPhoto string, sellerPrice, offerPrice int64, buyerName, message string) {
	chatID, err := strconv.ParseInt(sellerTGID, 10, 64)
	if err != nil {
		log.Printf("notify: bad sellerTGID %q: %v", sellerTGID, err)
		return
	}

	text := fmt.Sprintf(
		"🌸 <b>Новое предложение</b>\n\n"+
			"<b>%s</b>\n"+
			"Ваша цена: <b>%s ₸</b>\n"+
			"Покупатель: <b>%s ₸</b>",
		escapeHTML(bouquetTitle),
		formatPrice(sellerPrice),
		formatPrice(offerPrice),
	)
	if message != "" {
		text += fmt.Sprintf("\n\n<i>«%s»</i>", escapeHTML(message))
	}
	if buyerName != "" {
		text += "\n\nОт: " + escapeHTML(buyerName)
	}

	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("✅ Принять %s ₸", formatPrice(offerPrice)), CallbackData: fmt.Sprintf("offer:%d:accept", offerID)},
			},
			{
				// «Встречно» → открывает мини-апп c deeplink'ом на нужный оффер,
				// чтобы юзер юзал красивый ползунок в app, а не ForceReply в чате.
				{Text: "🔄 Встречно", WebApp: &WebAppInfo{URL: counterDeepLink(offerID)}},
				{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("offer:%d:reject", offerID)},
			},
		},
	}

	if err := sendOfferNotification(chatID, bouquetPhoto, text, markup); err != nil {
		log.Printf("notify NewOffer chat=%d: %v", chatID, err)
	}
}

// counterDeepLink — URL мини-аппа с query-параметром, который фронт распарсит
// и сразу откроет counter-модалку на нужном оффере. Менять URL — только синхронно
// с фронтом (App.vue читает ?counter=...).
const miniAppOrigin = "https://lana-flowers.vercel.app"

func counterDeepLink(offerID int64) string {
	return fmt.Sprintf("%s/?counter=%d", miniAppOrigin, offerID)
}

// sendOfferNotification — общая отправка: с фоткой через sendPhoto если URL есть,
// иначе обычное текстовое сообщение. Если sendPhoto упал (например, Telegram не
// смог скачать картинку), фолбэчимся на текст — лучше доставить без превью, чем
// потерять уведомление.
func sendOfferNotification(chatID int64, photoURL, text string, markup *InlineKeyboardMarkup) error {
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

// NotifyOfferAccepted — продавец принял оффер.
//
// Контакт продавца НЕ шлём в текст (ни @username, ни t.me-ссылку).
// Координация по доставке — внутри приложения (раздел «Сообщения»).
func NotifyOfferAccepted(buyerTGID string, bouquetTitle string, finalPrice int64, sellerName string) {
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}

	text := fmt.Sprintf(
		"✅ <b>Сделка состоялась!</b>\n\n"+
			"<b>%s</b> за <b>%s ₸</b>\n\n"+
			"Договоритесь о доставке с продавцом",
		escapeHTML(bouquetTitle),
		formatPrice(finalPrice),
	)
	if sellerName != "" {
		text += " " + escapeHTML(sellerName)
	}

	if _, err := SendMessage(SendMessageReq{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}); err != nil {
		log.Printf("notify Accepted chat=%d: %v", chatID, err)
	}
}

// NotifyDealCancelled — одна из сторон отменила уже принятую сделку.
// Получатель — противоположная сторона. iAmBuyer = true означает: инициатор
// отмены был покупатель, значит DM летит ПРОДАВЦУ (получатель не он сам).
func NotifyDealCancelled(toTGID string, bouquetTitle string, finalPrice int64, initiatedByBuyer bool) {
	chatID, err := strconv.ParseInt(toTGID, 10, 64)
	if err != nil {
		return
	}
	who := "Покупатель"
	if !initiatedByBuyer {
		who = "Продавец"
	}
	text := fmt.Sprintf(
		"⚠️ <b>%s отменил сделку</b>\n\n"+
			"<b>%s</b> за <b>%s ₸</b>\n\n"+
			"Букет снова в продаже — если ещё актуально, можно договориться заново.",
		who, escapeHTML(bouquetTitle), formatPrice(finalPrice),
	)
	if _, err := SendMessage(SendMessageReq{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}); err != nil {
		log.Printf("notify DealCancelled chat=%d: %v", chatID, err)
	}
}

// NotifyOfferExpired — покупателю что его pending-оффер заэкспайрился,
// потому что продавец принял другое предложение на тот же букет.
func NotifyOfferExpired(buyerTGID, bouquetTitle string, offerPrice int64) {
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"К сожалению, букет <b>%s</b> уже продан другому покупателю.\n\n"+
			"Ваше предложение <b>%s ₸</b> отменено.",
		escapeHTML(bouquetTitle), formatPrice(offerPrice),
	)
	if _, err := SendMessage(SendMessageReq{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}); err != nil {
		log.Printf("notify Expired chat=%d: %v", chatID, err)
	}
}

// NotifyOfferRejected — продавец отклонил.
func NotifyOfferRejected(buyerTGID string, bouquetTitle string, offeredPrice int64) {
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}

	text := fmt.Sprintf(
		"❌ Продавец отклонил вашу цену <b>%s ₸</b>\n\nза <b>%s</b>",
		formatPrice(offeredPrice),
		escapeHTML(bouquetTitle),
	)

	if _, err := SendMessage(SendMessageReq{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}); err != nil {
		log.Printf("notify Rejected chat=%d: %v", chatID, err)
	}
}

// NotifyOfferCountered — продавец дал встречную цену. Покупатель решает: принять / отклонить / встречно.
//
// @username продавца не показываем — приватные данные.
func NotifyOfferCountered(buyerTGID string, newOfferID, bouquetID int64, bouquetTitle, bouquetPhoto string, oldPrice, newPrice int64, sellerName, message string) {
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}

	text := fmt.Sprintf(
		"🔄 <b>Встречное предложение</b>\n\n"+
			"<b>%s</b>\n"+
			"Вы предлагали: %s ₸\n"+
			"Продавец просит: <b>%s ₸</b>",
		escapeHTML(bouquetTitle),
		formatPrice(oldPrice),
		formatPrice(newPrice),
	)
	if message != "" {
		text += fmt.Sprintf("\n\n<i>«%s»</i>", escapeHTML(message))
	}
	if sellerName != "" {
		text += "\n\nОт: " + escapeHTML(sellerName)
	}

	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("✅ Принять %s ₸", formatPrice(newPrice)), CallbackData: fmt.Sprintf("offer:%d:accept", newOfferID)},
			},
			{
				{Text: "🔄 Встречно", WebApp: &WebAppInfo{URL: counterDeepLink(newOfferID)}},
				{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("offer:%d:reject", newOfferID)},
			},
		},
	}

	if err := sendOfferNotification(chatID, bouquetPhoto, text, markup); err != nil {
		log.Printf("notify Countered chat=%d: %v", chatID, err)
	}
}

// AskForCounterPrice — после тапа "Встречно" просим юзера ввести сумму через ForceReply.
// В тексте сообщения зашиваем offer_id, чтобы при ответе мы могли его восстановить.
func AskForCounterPrice(chatID any, offerID int64, currentPrice int64) {
	text := fmt.Sprintf(
		"💬 Введите вашу <b>встречную цену</b> в тенге для оффера #%d.\nТекущая цена: %s ₸",
		offerID, formatPrice(currentPrice),
	)
	markup := &ForceReply{
		ForceReply: true,
		Selective:  true,
		InputField: "Например: 11500",
	}
	if _, err := SendMessage(SendMessageReq{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	}); err != nil {
		log.Printf("ask counter: %v", err)
	}
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
