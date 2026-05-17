package telegram

import (
	"fmt"
	"log"
	"strconv"
)

// Нотификации про сами ОФФЕРЫ (входящий новый / встречная цена) —
// карточка с фото букета + callback-кнопками accept/counter/reject.
//
// callback_data формат: "offer:<id>:accept" / ":reject" / ":counter" —
// разбирается в webhook handleCallback.

// NotifyNewOffer — продавцу: пришёл новый оффер от покупателя.
//
// @username покупателя в текст НЕ показываем — приватные данные,
// продавцу хватает первого имени.
func NotifyNewOffer(sellerTGID string, offerID, bouquetID int64, bouquetTitle, bouquetPhoto string, sellerPrice, offerPrice int64, buyerName string) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(sellerTGID, 10, 64)
	if err != nil {
		log.Printf("notify: bad sellerTGID %q: %v", sellerTGID, err)
		return
	}

	text := fmt.Sprintf(
		"💌 Новое предложение\n\n<b>%s</b>\n\n%s",
		escapeHTML(bouquetTitle),
		formatPriceWithDelta(offerPrice, sellerPrice),
	)
	if buyerName != "" {
		text += "\nОт: " + escapeHTML(buyerName)
	}

	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:         fmt.Sprintf("Согласиться на %s ₸", formatPrice(offerPrice)),
					CallbackData: fmt.Sprintf("offer:%d:accept", offerID),
				},
			},
			{
				{Text: "Предложить свою", CallbackData: fmt.Sprintf("offer:%d:counter", offerID)},
				{Text: "Отклонить", CallbackData: fmt.Sprintf("offer:%d:reject", offerID)},
			},
		},
	}

	if err := sendPhotoOrText(chatID, bouquetPhoto, text, markup); err != nil {
		log.Printf("notify NewOffer chat=%d: %v", chatID, err)
	}
}

// NotifyOfferCountered — контрагенту: пришла встречная цена.
//
// У встречки контекст разницы НЕ %, а просто «вы предлагали X»: получатель
// сам делал предыдущее предложение, ему важна абсолютная разница против
// собственной цены, а не % от запросной (asking price там не релевантна).
//
// @username отправителя не показываем — приватные данные.
func NotifyOfferCountered(buyerTGID string, newOfferID, bouquetID int64, bouquetTitle, bouquetPhoto string, oldPrice, newPrice int64, sellerName string) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}

	text := fmt.Sprintf(
		"🔄 Встречная цена\n\n<b>%s</b>\n\nЦена:"+nbsp+"<b>%s"+nbsp+"₸</b>\nВы предлагали:"+nbsp+"%s"+nbsp+"₸",
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
				{
					Text:         fmt.Sprintf("Согласиться на %s ₸", formatPrice(newPrice)),
					CallbackData: fmt.Sprintf("offer:%d:accept", newOfferID),
				},
			},
			{
				{Text: "Предложить свою", CallbackData: fmt.Sprintf("offer:%d:counter", newOfferID)},
				{Text: "Отклонить", CallbackData: fmt.Sprintf("offer:%d:reject", newOfferID)},
			},
		},
	}

	if err := sendPhotoOrText(chatID, bouquetPhoto, text, markup); err != nil {
		log.Printf("notify Countered chat=%d: %v", chatID, err)
	}
}
