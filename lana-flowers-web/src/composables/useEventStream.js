import { onBeforeUnmount, onMounted, ref } from 'vue'

// useEventStream — подписка на SSE-стрим бэка /events. EventSource в
// браузере не даёт ставить custom headers, поэтому initData передаём
// query-параметром ?tma=...; бэк-middleware читает из обоих мест.
//
// Auto-reconnect: EventSource сам пытается переподключиться, но если
// браузер отдал error без авто-reconnect'а (бывает в WebView'ах) — мы
// форсируем retry через exponential backoff с потолком 30с.

const BASE_URL = import.meta.env.VITE_API_URL || 'https://64-226-107-161.nip.io'

export function useEventStream(onEvent) {
  const connected = ref(false)
  let es = null
  let retryAttempt = 0
  let retryTimer = null

  function connect() {
    const initData = window.Telegram?.WebApp?.initData || ''
    if (!initData) {
      // SSR / нет Telegram-окружения → молча выходим, polling доделает.
      return
    }
    // encodeURIComponent — initData содержит &, = и прочее, что нужно эскейпить.
    const url = `${BASE_URL}/events?tma=${encodeURIComponent(initData)}`

    try {
      es = new EventSource(url)
    } catch (e) {
      scheduleRetry()
      return
    }

    es.addEventListener('ready', () => {
      connected.value = true
      retryAttempt = 0 // успешный коннект сбрасывает backoff
    })

    es.addEventListener('message', (msg) => {
      try {
        const data = JSON.parse(msg.data)
        if (data && typeof onEvent === 'function') onEvent(data)
      } catch {}
    })

    es.addEventListener('error', () => {
      connected.value = false
      // EventSource по spec'у переподключается сам, но WebView'ы Telegram
      // иногда «висят» без auto-retry. Подстраховываемся: явный close +
      // backoff. Если EventSource всё-таки в processing — close его остановит,
      // не утечёт.
      try { es?.close() } catch {}
      es = null
      scheduleRetry()
    })
  }

  function scheduleRetry() {
    if (retryTimer) return
    // 1s, 2s, 4s, 8s, 16s, 30s, 30s, ...
    const delay = Math.min(30000, 1000 * Math.pow(2, retryAttempt))
    retryAttempt++
    retryTimer = setTimeout(() => {
      retryTimer = null
      connect()
    }, delay)
  }

  function disconnect() {
    if (retryTimer) {
      clearTimeout(retryTimer)
      retryTimer = null
    }
    if (es) {
      try { es.close() } catch {}
      es = null
    }
    connected.value = false
  }

  onMounted(connect)
  onBeforeUnmount(disconnect)

  return { connected, disconnect, reconnect: connect }
}
