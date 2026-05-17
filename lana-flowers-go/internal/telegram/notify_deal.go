package telegram

import (
	"fmt"
	"strconv"
)

// Нотификации про ЖИЗНЕННЫЙ ЦИКЛ СДЕЛКИ: принято, продавцу подтверждение,
// сделка отменена. Все три — без фото, с CTA-кнопкой в мини-апп
// (Сделки / Мои букеты / Похожие).

// NotifyOfferAccepted — покупателю: его оффер принят. CTA в Сделки.
//
// Контакт продавца в текст НЕ кладём — он раскрывается только в Сделках
// (там же где договариваются о деталях).
func NotifyOfferAccepted(buyerTGID string, bouquetTitle string, finalPrice int64, sellerName string) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}
	_ = sellerName // имя контрагента доступно в Сделках, в DM избыточно

	text := fmt.Sprintf(
		"✅ Предложение принято\n\n<b>%s</b>\n\nЦена:"+nbsp+"<b>%s"+nbsp+"₸</b>\n\nДоговоритесь о деталях с продавцом.",
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

// NotifyDealConfirmed — продавцу: он принял оффер, сделка состоялась.
// Зеркальная функция к NotifyOfferAccepted (для покупателя). Шлётся даже
// если accept был сделан через inline-кнопку в DM-боте, который обновляет
// старое сообщение — отдельное явное подтверждение полезнее, плюс кнопка
// «Открыть Сделки» сразу под рукой без лазания по чату.
func NotifyDealConfirmed(sellerTGID string, bouquetTitle string, finalPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(sellerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"✅ Сделка состоялась\n\n<b>%s</b>\n\nЦена:"+nbsp+"<b>%s"+nbsp+"₸</b>\n\nДоговоритесь о деталях с покупателем.",
		escapeHTML(bouquetTitle),
		formatPrice(finalPrice),
	)
	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{miniAppButton("💬 Открыть Сделки", "deals")},
		},
	}
	sendStructured(chatID, text, markup, "DealConfirmed")
}

// NotifyDealCancelled — одна из сторон отменила уже принятую сделку.
// Получатель — противоположная сторона. recipientIsBuyer определяет тон и CTA:
//   - recipientIsBuyer=false (продавцу, отменил buyer): «Покупатель передумал» + «Мои букеты»
//   - recipientIsBuyer=true  (покупателю, отменил seller): «Продавец отменил» + «Похожие»
//
// «Мои букеты» ведёт на ?screen=deals — там новая секция «На продаже»
// (раньше эта секция жила в Профиле, но мы её перенесли в Сделки).
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
		btn = miniAppButton("🛒 Мои букеты", "deals")
	}

	text := fmt.Sprintf(
		"⚠️ Сделка отменена\n\n<b>%s</b>\n\nЦена:"+nbsp+"<b>%s"+nbsp+"₸</b>\n\n%s",
		escapeHTML(bouquetTitle),
		formatPrice(finalPrice),
		body,
	)
	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{{btn}},
	}
	sendStructured(chatID, text, markup, "DealCancelled")
}
