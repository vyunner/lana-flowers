// Форматирование чисел и телефонов. Единое место — раньше formatPrice
// был склонирован в 6+ компонентах с лёгкими различиями.

const ruFmt = new Intl.NumberFormat('ru-RU')

/** «12500» → «12 500» — только число, без валюты. */
export function formatPrice(n) {
  return ruFmt.format(Number(n) || 0)
}

/** «12500» → «12 500 ₸» — с валютой, для отображения в UI. */
export function formatPriceKzt(n) {
  return formatPrice(n) + ' ₸'
}

/** «77026207447» → «+7 702 620 74 47». Если не похоже на казахский 11-значный
 *  номер — возвращаем как есть, чтобы не сломать рендер. */
export function formatPhoneKz(raw) {
  if (!raw) return ''
  const d = String(raw).replace(/\D/g, '')
  if (d.length === 11 && (d[0] === '7' || d[0] === '8')) {
    return `+7 ${d.slice(1, 4)} ${d.slice(4, 7)} ${d.slice(7, 9)} ${d.slice(9, 11)}`
  }
  return String(raw)
}

/** «12 500 ₸» / «abc» → 12500 / 0. Чтобы парсить input цены. */
export function parsePrice(str) {
  const n = parseInt(String(str).replace(/\D/g, ''), 10)
  return Number.isFinite(n) ? n : 0
}

/** Минимальная цена оффера. Меньше не имеет смысла предлагать (продавцу
 *  не нужны 5-рублёвые офферы, и в спам-фильтр идёт). Согласовать с UI. */
export const MIN_OFFER_PRICE = 100
