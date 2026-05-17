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

/** TTL pending-оффера до auto-expire. ДОЛЖЕН совпадать с PendingTTL в
 *  lana-flowers-go/internal/handler/offers/expire.go — иначе фронт будет
 *  показывать неправильное оставшееся время. */
export const OFFER_TTL_MS = 2 * 60 * 60 * 1000

/** Русское склонение: pluralRu(2, ['день','дня','дней']) → 'дня'. */
function pluralRu(n, forms) {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return forms[0]
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return forms[1]
  return forms[2]
}

/** Короткая фраза «через 5 ч» / «через 23 мин» / «через 2 дня» / «скоро» / «истёк».
 *  Идёт inline в meta-строку оффера — поэтому без избыточного «истекает».
 *  Минуты/часы — аббревиатурой (мин/ч), дни — с полным склонением. */
export function formatRemaining(deadlineMs) {
  const diff = deadlineMs - Date.now()
  if (diff <= 0) return 'истёк'
  const min = Math.floor(diff / 60000)
  if (min < 5) return 'скоро'
  if (min < 60) return `через ${min} мин`
  const hours = Math.floor(min / 60)
  if (hours < 24) return `через ${hours} ч`
  const days = Math.floor(hours / 24)
  return `через ${days} ${pluralRu(days, ['день', 'дня', 'дней'])}`
}
