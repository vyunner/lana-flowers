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
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(sellerTGID, 10, 64)
	if err != nil {
		log.Printf("notify: bad sellerTGID %q: %v", sellerTGID, err)
		return
	}

	// Короткий формат: «<Бук> — <BUYER ₸> (вы: <SELLER ₸>)»
	// одной строкой. Если есть buyerName — добавляем «От: …».
	// Раньше было 4 строки с дублями «Ваша цена / Покупатель» — мусор,
	// продавец и так видит свою цену в карточке.
	text := fmt.Sprintf(
		"🌸 <b>%s</b> — <b>%s ₸</b> (вы: %s ₸)",
		escapeHTML(bouquetTitle),
		formatPrice(offerPrice),
		formatPrice(sellerPrice),
	)
	if buyerName != "" {
		text += "\nОт: " + escapeHTML(buyerName)
	}
	if message != "" {
		text += "\n<i>«" + escapeHTML(message) + "»</i>"
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

	if err := sendOfferNotification(chatID, bouquetPhoto, text, markup); err != nil {
		log.Printf("notify NewOffer chat=%d: %v", chatID, err)
	}
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
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}

	// Короткий формат — детали и кнопка «Связаться» в «Сделках».
	text := fmt.Sprintf(
		"✅ Принято! <b>%s</b> за <b>%s ₸</b>. Контакты — в Сделках.",
		escapeHTML(bouquetTitle),
		formatPrice(finalPrice),
	)
	_ = sellerName // имя контрагента доступно в Сделках, в DM избыточно

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
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(toTGID, 10, 64)
	if err != nil {
		return
	}
	who := "Покупатель"
	if !initiatedByBuyer {
		who = "Продавец"
	}
	text := fmt.Sprintf(
		"⚠️ %s отменил сделку <b>%s</b> (%s ₸). Букет снова в продаже.",
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

// NotifyOfferWithdrawn — продавцу, что покупатель отозвал своё pending-предложение
// до того, как продавец на него ответил. Сделки не было, букет с продажи не уходил —
// поэтому формулировка отличается от NotifyDealCancelled («снова в продаже» врало бы).
func NotifyOfferWithdrawn(sellerTGID string, bouquetTitle string, offeredPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(sellerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"↩️ Покупатель отозвал предложение <b>%s</b> (%s ₸).",
		escapeHTML(bouquetTitle), formatPrice(offeredPrice),
	)
	if _, err := SendMessage(SendMessageReq{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}); err != nil {
		log.Printf("notify OfferWithdrawn chat=%d: %v", chatID, err)
	}
}

// NotifyOfferExpired — покупателю что его pending-оффер заэкспайрился,
// потому что продавец принял другое предложение на тот же букет.
func NotifyOfferExpired(buyerTGID, bouquetTitle string, offerPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"Букет <b>%s</b> уже продан другому. Ваше %s ₸ отменено.",
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
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}

	text := fmt.Sprintf(
		"❌ Отклонено: <b>%s</b> — %s ₸",
		escapeHTML(bouquetTitle),
		formatPrice(offeredPrice),
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
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}

	// Короткая встречка: «<Бук> — <NEW ₸> (было: <OLD ₸>)».
	text := fmt.Sprintf(
		"🔄 <b>%s</b> — <b>%s ₸</b> (было: %s ₸)",
		escapeHTML(bouquetTitle),
		formatPrice(newPrice),
		formatPrice(oldPrice),
	)
	if sellerName != "" {
		text += "\nОт: " + escapeHTML(sellerName)
	}
	if message != "" {
		text += "\n<i>«" + escapeHTML(message) + "»</i>"
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

	if err := sendOfferNotification(chatID, bouquetPhoto, text, markup); err != nil {
		log.Printf("notify Countered chat=%d: %v", chatID, err)
	}
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
