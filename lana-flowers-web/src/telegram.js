/**
 * Минимальная интеграция с Telegram WebApp.
 * SDK подключается через <script> в index.html — здесь только обёртка.
 */

const tg = typeof window !== 'undefined' ? window.Telegram?.WebApp : null

export function initTelegram() {
  if (!tg) return null

  // Сообщаем Telegram что мы готовы — убирает loader
  tg.ready()

  // Раскрываем на полную высоту (по умолчанию ~50%)
  tg.expand()

  // Чтобы свайп вниз не закрывал app случайно
  tg.disableVerticalSwipes?.()

  // Подменяем 100dvh на реальную высоту мини-аппа от Telegram
  // (без неё контент не центрируется — 100dvh включает шапку Telegram).
  syncViewportHeight()
  tg.onEvent?.('viewportChanged', syncViewportHeight)

  // Отступ под Telegram-шапкой / системным notch.
  // Bot API 7.10+ отдаёт contentSafeAreaInset, на старых клиентах берём
  // safeAreaInset, на совсем древних — 0 (фолбэк в CSS).
  syncSafeArea()
  tg.onEvent?.('safeAreaChanged', syncSafeArea)
  tg.onEvent?.('contentSafeAreaChanged', syncSafeArea)

  return tg
}

function syncViewportHeight() {
  const h =
    (tg && tg.viewportHeight) ||
    (typeof window !== 'undefined' ? window.innerHeight : 0)
  if (h) {
    document.documentElement.style.setProperty('--tg-vh', h + 'px')
  }
}

function syncSafeArea() {
  const top = Math.round(
    Math.max(
      tg?.contentSafeAreaInset?.top || 0,
      tg?.safeAreaInset?.top || 0,
    ),
  )
  document.documentElement.style.setProperty('--tg-safe-top', top + 'px')
}

/** Информация о юзере из initData (id, имя, аватар, премиум статус). */
export function getTelegramUser() {
  return tg?.initDataUnsafe?.user ?? null
}

/** Тактильный feedback на действие. */
export function haptic(style = 'light') {
  // styles: light, medium, heavy, rigid, soft
  tg?.HapticFeedback?.impactOccurred?.(style)
}

/** Уведомление-фидбэк (success / warning / error). */
export function hapticNotify(type = 'success') {
  tg?.HapticFeedback?.notificationOccurred?.(type)
}

/** Закрыть мини-апп. */
export function closeApp() {
  tg?.close?.()
}

export { tg }
