package telegram

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// Форматирование цены, дельты, deeplink-кнопки и HTML-escape — общее
// для всех Notify*-функций. Логика держится синхронной с фронтом
// (lana-flowers-web/src/utils/format.js и utils/offers.js): иначе DM и
// in-app карточки разъедутся в цифрах/процентах.

// nbsp — неразрывный пробел (U+00A0). Используем между группами цифр в
// цене и перед «₸», чтобы Telegram-рендерер не разрывал «30 000» или
// «27 000 ₸» по обычному space'у при wrap'е сообщения в узком пузыре чата.
const nbsp = " "

// formatPrice превращает 12500 → "12<NBSP>500". NBSP вместо обычного
// пробела чтобы число не расщеплялось на перенос строки в Telegram-чате
// (см. nbsp выше). На рендеринг визуально не влияет — та же ширина.
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
			sb.WriteString(nbsp)
		}
	}
	for i := pre; i < len(s); i += 3 {
		sb.WriteString(s[i : i+3])
		if i+3 < len(s) {
			sb.WriteString(nbsp)
		}
	}
	return sb.String()
}

// priceDeltaPct — % разница ask vs offer. Положительное = покупатель
// просит меньше (скидка), отрицательное = предложил больше. Возвращает
// (0, false) если цены равны или ask некорректный. Логика та же что во
// фронте (lana-flowers-web/src/utils/offers.js → priceDeltaPct).
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

// formatPriceWithDelta — multi-line блок «Цена: NN ₸ (±%) / Ваша: XX ₸».
//
// Inline-формат («NN ₸ · −10% от вашей цены (XX ₸)») в DM не лезет в
// узкий пузырь — рвётся в неудобных местах. Multi-line c label:value
// предсказуем по ширине, каждая строка ≤ ~25 символов, не wrap'ится.
//
//	formatPriceWithDelta(25000, 30000) →
//	    "Цена: <b>25 000 ₸</b> (−17%)\nВаша: 30 000 ₸"
//	formatPriceWithDelta(30000, 30000) →
//	    "Цена: <b>30 000 ₸</b>"     ← когда равно, вторая строка не нужна
//	formatPriceWithDelta(31000, 30000) →
//	    "Цена: <b>31 000 ₸</b> (+3%)\nВаша: 30 000 ₸"
func formatPriceWithDelta(offerPrice, askPrice int64) string {
	base := fmt.Sprintf("Цена:"+nbsp+"<b>%s"+nbsp+"₸</b>", formatPrice(offerPrice))
	pct, ok := priceDeltaPct(askPrice, offerPrice)
	if !ok {
		return base
	}
	sign := "−"
	abs := pct
	if pct < 0 {
		sign = "+"
		abs = -pct
	}
	return fmt.Sprintf("%s"+nbsp+"(%s%d%%)\nВаша:"+nbsp+"%s"+nbsp+"₸",
		base, sign, abs, formatPrice(askPrice))
}

// miniAppButton — кнопка-deeplink в мини-апп на нужный экран.
// screen: "catalog" | "deals" | "profile" — должен совпадать с парсером
// ?screen=... в App.vue. WebApp inline-кнопки работают в DM с ботом.
// WEBAPP_URL берётся из env, fallback — prod alias.
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

// escapeHTML — минимальный escape для HTML parse_mode. Только & < >;
// никаких полноценных entity-таблиц не нужно, Telegram парсит только
// `<b> <i> <u> <s> <code> <pre> <a>`, в значениях остальные символы
// безопасны.
func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
