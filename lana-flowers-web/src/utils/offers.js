// Pure-функции для офферов и букетов: имена сторон, формулировки статусов,
// арифметика дельты цены. Без зависимостей от Vue/store — переиспользуется
// из карточек Сделок, тостов, и любых будущих мест.

import { formatPrice } from './format'

/**
 * Имя контрагента в человеческом виде. Если бэк ничего не отдал —
 * фолбэк по роли («Покупатель»/«Продавец»), чтобы UI не лопался пустотой.
 */
export function cpName(offer) {
  return offer.counterparty?.name || (offer.role === 'buyer' ? 'Продавец' : 'Покупатель')
}

/**
 * Разница между запросной ценой букета и предложением покупателя в %.
 *  > 0 — скидка (покупатель просит меньше), < 0 — надбавка.
 *  null — цены равны, нет смысла показывать «0%», или нет данных.
 */
export function priceDeltaPct(offer) {
  const ask = offer.bouquet?.price
  if (!ask || ask <= 0) return null
  const pct = Math.round(((ask - offer.price) / ask) * 100)
  return Math.abs(pct) >= 1 ? pct : null
}

/**
 * Человеческое название статуса оффера для секции «История».
 * Возвращает сырое значение если статус неизвестный — лучше показать
 * непонятный лейбл чем пустую строку, проще диагностировать.
 */
export function offerHistoryStatusText(offer) {
  switch (offer.status) {
    case 'rejected':
      return 'Отклонено'
    case 'cancelled':
      return 'Сделка отменена'
    case 'expired':
      return 'Букет ушёл другому'
    case 'countered':
      return 'Был встречный ответ'
    default:
      return offer.status
  }
}

/** Лейбл для статуса букета в истории — только закрытые состояния. */
export function bouquetHistoryStatusText(status) {
  if (status === 'sold') return 'Продано'
  if (status === 'archived') return 'Снято'
  return status
}

/** Удобный price+₸ из числа. Реэкспорт чтобы не таскать formatPrice отдельно. */
export function priceWithCurrency(n) {
  return formatPrice(n) + ' ₸'
}
