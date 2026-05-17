package telegram

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

// formatPrice превращает 12500 → "12 500".
func formatPrice(n int64) string {
	s := strconv.FormatInt(n, 10)
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

// priceDeltaPct — % разница ask vs offer. Положительное = покупатель просит
// меньше (скидка), отрицательное = предложил больше. Возвращает (0, false)
// если цены равны или ask некорректный. Логика та же что во фронте
// (lana-flowers-web/src/components/Deals.vue → priceDeltaPct).
func priceDeltaPct(ask, offer int64) (int, bool) {
	if ask <= 0 {
		return 0, false
	}
	pct := int(math.Round(float64(ask-offer) / float64(ask) * 100))
	if pct == 0 {
		return 0, false
	}
	return pct, true
}

// formatPriceWithDelta — строка цены с инлайн-контекстом разницы.
//
//	formatPriceWithDelta(25000, 30000) → "<b>25 000 ₸</b> · −17% от вашей цены (30 000 ₸)"
//	formatPriceWithDelta(30000, 30000) → "<b>30 000 ₸</b>"  (без избыточной "вашей цены")
//	formatPriceWithDelta(31000, 30000) → "<b>31 000 ₸</b> · +3% к вашей цене (30 000 ₸)"
func formatPriceWithDelta(offerPrice, askPrice int64) string {
	base := fmt.Sprintf("<b>%s ₸</b>", formatPrice(offerPrice))
	pct, ok := priceDeltaPct(askPrice, offerPrice)
	if !ok {
		return base
	}
	if pct > 0 {
		return fmt.Sprintf("%s · −%d%% от вашей цены (%s ₸)", base, pct, formatPrice(askPrice))
	}
	return fmt.Sprintf("%s · +%d%% к вашей цене (%s ₸)", base, -pct, formatPrice(askPrice))
}

// miniAppButton — кнопка-deeplink в мини-апп на нужный экран.
// screen: "catalog" | "deals" | "profile" — должен совпадать с парсером
// ?screen=... в App.vue. WebApp inline-кнопки работают в DM с ботом.
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
// ParseMode и лог-формат для всех итоговых уведомлений без фото.
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
// Шаблон карточки в DM повторяет визуальную структуру in-app Сделок
// (lana-flowers-web/src/components/Deals.vue):
//
//   <emoji-статус> <Заголовок-действие>
//
//   <b>Название букета</b>
//
//   <b>NNN ₸</b> [· инлайн контекст вроде «−17% от вашей цены»]
//   [От: Имя]
//
//   [Инструкция/что делать]
//
//   [Inline-кнопки]
//
// Эмодзи в заголовке ОСТАВЛЕНЫ (в отличие от тостов и in-app): в чат-листе
// Telegram превью сообщения это единственный визуальный якорь чтобы
// различить «принято» / «отклонено» / «новое» не открывая бот. Без эмодзи
// все нотификации сливаются в одинаковый «Lana Flowers · текст…».
//
// А вот на callback-кнопках (Согласиться/Предложить/Отклонить) эмодзи
// убраны — текст совпадает с тем, что юзер видит в карточке Сделок.

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

// NotifyOfferCountered — контрагенту: пришла встречная цена. Photo + callback.
//
// У встречки контекст разницы не % (получатель сам делал предыдущее
// предложение, ему важна абсолютная разница), а просто «вы предлагали X».
func NotifyOfferCountered(buyerTGID string, newOfferID, bouquetID int64, bouquetTitle, bouquetPhoto string, oldPrice, newPrice int64, sellerName string) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}

	text := fmt.Sprintf(
		"🔄 Встречная цена\n\n<b>%s</b>\n\n<b>%s ₸</b> (вы предлагали %s ₸)",
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
		"✅ Предложение принято\n\n<b>%s</b>\n\n<b>%s ₸</b>\n\nДоговоритесь о деталях с продавцом.",
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
		"✅ Сделка состоялась\n\n<b>%s</b>\n\n<b>%s ₸</b>\n\nДоговоритесь о деталях с покупателем.",
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

// NotifyOfferRejected — покупателю: его оффер отклонён. CTA в каталог.
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
// как продавец ответил. Сделки не было, букет с продажи не уходил — отдельная
// функция от NotifyDealCancelled (другой ассум о состоянии букета).
// CTA нет: букет остался активным, действий не требуется.
func NotifyOfferWithdrawn(sellerTGID string, bouquetTitle string, offeredPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(sellerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"↩️ Покупатель отозвал предложение\n\n<b>%s</b>\n\n<b>%s ₸</b>\n\nБукет остаётся в продаже.",
		escapeHTML(bouquetTitle),
		formatPrice(offeredPrice),
	)
	sendStructured(chatID, text, nil, "OfferWithdrawn")
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
		"⚠️ Сделка отменена\n\n<b>%s</b>\n\n<b>%s ₸</b>\n\n%s",
		escapeHTML(bouquetTitle),
		formatPrice(finalPrice),
		body,
	)
	markup := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{{btn}},
	}
	sendStructured(chatID, text, markup, "DealCancelled")
}

// NotifyOfferExpired — покупателю: его pending заэкспайрился, продавец принял
// другое предложение на тот же букет. CTA в каталог за похожими.
func NotifyOfferExpired(buyerTGID, bouquetTitle string, offerPrice int64) {
	defer trackEnd(trackStart())
	chatID, err := strconv.ParseInt(buyerTGID, 10, 64)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		"⏱ Букет ушёл другому покупателю\n\n<b>%s</b>\n\nВаше предложение <b>%s ₸</b> отменено.",
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

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
