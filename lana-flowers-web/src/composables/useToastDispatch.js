import { pushToast } from '../state/toasts'
import { formatPriceKzt } from '../utils/format'

/**
 * Превращает SSE-event от бэка в in-app toast по нашему стилю.
 *
 *   const dispatch = useToastDispatch({ onToastTap: () => switchToDealsTab() })
 *   useEventStream(dispatch, ...)
 *
 * Формат текста: «<действие> · <цена>\n<название>». Первая строка короткая
 * и предсказуемая, вторая — название букета (вариативная длина перенесётся
 * не ломая первую строку, см. white-space: pre-line в Toast.vue).
 *
 * Цвет (kind → background) несёт статус-сигнал: success/err/warn/info.
 * Эмодзи в тексте дублировали бы это семантически — не используем.
 *
 * onToastTap — необязательный action для тапа по тосту. Все наши event'ы
 * относятся к Сделкам, обычно сюда передают «переключение на таб deals».
 */
export function useToastDispatch({ onToastTap } = {}) {
  const action = typeof onToastTap === 'function' ? onToastTap : undefined

  return function dispatch(e) {
    if (!e || !e.type) return
    const priceStr = e.price ? formatPriceKzt(e.price) : ''
    const title = e.bouquet_title ? `«${e.bouquet_title}»` : ''

    switch (e.type) {
      case 'offer.created':
        pushToast(`Новое предложение · ${priceStr}\n${title}`, { kind: 'info', action })
        break
      case 'offer.accepted':
        pushToast(`Принято · ${priceStr}\n${title}`, { kind: 'success', ttl: 6000, action })
        break
      case 'offer.rejected':
        pushToast(`Отклонено · ${priceStr}\n${title}`, { kind: 'err', action })
        break
      case 'offer.countered':
        pushToast(`Встречное · ${priceStr}\n${title}`, { kind: 'warn', action })
        break
      case 'offer.cancelled':
        pushToast(`Сделка отменена · ${priceStr}\n${title}`, { kind: 'warn', action })
        break
      case 'offer.expired':
        pushToast(`Букет ушёл другому · ${priceStr}\n${title}`, { kind: 'err', action })
        break
      // Неизвестные event'ы игнорируем — лучше тихо чем мусорить.
    }
  }
}
