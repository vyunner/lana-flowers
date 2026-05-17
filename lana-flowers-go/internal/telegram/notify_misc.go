package telegram

import (
	"fmt"
	"log"
	"strconv"
)

// Прочие нотификации: отклонено / отозвано / опоздал + ForceReply
// для ввода встречной цены.

// NotifyOfferRejected — покупателю: его оффер отклонён. CTA в каталог.
func NotifyOfferRejected(buyerTGID string, bouquetTitle string, offeredPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"❌ Предложение отклонено\n\n<b>%s</b>\n\nВаша цена:"+nbsp+"<b>%s"+nbsp+"₸</b>\n\nПопробуйте другую цену или посмотрите похожие букеты.",
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
// как продавец ответил. Сделки не было, букет с продажи не уходил —
// отдельная функция от NotifyDealCancelled (другой ассум о состоянии
// букета). CTA нет: букет остался активным, действий не требуется.
func NotifyOfferWithdrawn(sellerTGID string, bouquetTitle string, offeredPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(sellerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"↩️ Покупатель отозвал предложение\n\n<b>%s</b>\n\nПредложение:"+nbsp+"<b>%s"+nbsp+"₸</b>\n\nБукет остаётся в продаже.",
		escapeHTML(bouquetTitle),
		formatPrice(offeredPrice),
	)
	sendStructured(chatID, text, nil, "OfferWithdrawn")
}

// NotifyOfferExpired — покупателю: его pending заэкспайрился, продавец
// принял другое предложение на тот же букет. CTA в каталог за похожими.
func NotifyOfferExpired(buyerTGID, bouquetTitle string, offerPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"⏱ Букет ушёл другому покупателю\n\n<b>%s</b>\n\nВаше предложение"+nbsp+"<b>%s"+nbsp+"₸</b> отменено.",
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

// AskForCounterPrice — после тапа «Предложить свою» бот шлёт ForceReply
// «Укажите встречную цену», юзер пишет число в чат, handleReply парсит.
//
// В тексте ОБЯЗАТЕЛЬНО оставляем «#N» в конце — handleReply парсит его
// regex'ом `#(\d+)` чтобы восстановить offer_id из reply-контекста.
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
