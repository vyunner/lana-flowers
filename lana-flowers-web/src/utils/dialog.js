// Обёртки над нативными confirm/alert, которые используют Telegram-UI
// если мини-апп открыт изнутри Telegram (нативные popup'ы выглядят
// как в Telegram, не как браузерный модал из 2010-х).
//
// Везде где раньше был window.confirm() / alert() — теперь эти promise-based
// функции. Если Telegram SDK недоступен (открыли в браузере для отладки) —
// фолбэк на нативные диалоги.

import { tg } from '../telegram'

/**
 * Показать confirm-диалог. Возвращает Promise<boolean>.
 *
 * await confirm('Точно отменить сделку?') // true/false
 */
export function confirm(message) {
  if (tg && typeof tg.showConfirm === 'function') {
    return new Promise((resolve) => {
      tg.showConfirm(message, (ok) => resolve(!!ok))
    })
  }
  return Promise.resolve(window.confirm(message))
}

/**
 * Показать alert. Promise резолвится когда юзер закрыл попап.
 *
 * await alert('Ошибка: ' + e.message)
 */
export function alert(message) {
  if (tg && typeof tg.showAlert === 'function') {
    return new Promise((resolve) => {
      tg.showAlert(message, () => resolve())
    })
  }
  window.alert(message)
  return Promise.resolve()
}
