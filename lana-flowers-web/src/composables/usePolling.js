import { onBeforeUnmount, onMounted } from 'vue'
import { sseConnected } from '../state/realtime'

/**
 * Поллинг с паузой когда вкладка/мини-апп ушёл в фон.
 *
 * Использование:
 *   usePolling(loadData, 15000)
 *
 * Логика:
 * - setInterval каждые intervalMs мс запускает callback
 * - НЕ запускает следующий тик пока предыдущий не завершился (анти-overlap)
 * - На document.visibilitychange:
 *     hidden → останавливаем таймер (юзер в другом приложении)
 *     visible → СРАЗУ один tick + restart таймера (свежие данные при возврате)
 *
 * Почему такой polling, а не WebSocket: для C2C-маркетплейса масштаба MVP
 * не оправдано держать тысячи long-lived соединений. Опрос раз в 15-30с
 * даёт UI-консистентность дешево; пуш-канал для важных событий — уже есть
 * (Telegram-бот шлёт DM в чат).
 */
export function usePolling(callback, intervalMs = 15000) {
  let timer = null
  let running = false

  async function tick() {
    if (running) return
    // Если SSE-стрим живой — апдейты прилетят мгновенно через push,
    // дёргать REST каждые 60с впустую не имеет смысла. Когда SSE
    // упадёт (connected → false) — polling сам возобновится.
    if (sseConnected.value) return
    running = true
    try {
      await callback()
    } catch (e) {
      // молча — polling не должен сыпать модалками
      if (typeof console !== 'undefined') console.warn('polling tick failed:', e)
    } finally {
      running = false
    }
  }

  function start() {
    if (timer) return
    timer = setInterval(tick, intervalMs)
  }

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  function onVisibilityChange() {
    if (typeof document === 'undefined') return
    if (document.hidden) {
      stop()
    } else {
      // Сразу подтянуть свежие данные при возврате — иначе юзер видит
      // stale до следующего тика.
      tick()
      start()
    }
  }

  onMounted(() => {
    start()
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', onVisibilityChange)
    }
  })

  onBeforeUnmount(() => {
    stop()
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  })

  return { start, stop, tick }
}
