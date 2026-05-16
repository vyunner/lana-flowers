// Глобальный стейт тоастов. Любой компонент может pushToast(...).
// Toast.vue рендерит стек и сам убирает через таймер.

import { ref } from 'vue'

export const toasts = ref([]) // [{ id, text, kind, action, createdAt }]

let nextId = 1
const DEFAULT_TTL_MS = 4500
const MAX_VISIBLE = 3

/**
 * Добавить тост.
 *
 *   pushToast('Текст', { kind: 'success', action: () => switchTab('deals') })
 *
 * kind = 'info' | 'success' | 'warn' | 'err'
 * action — необязательный колбэк, вызывается при тапе на тост ДО его dismiss'а.
 *          Подходит для «toast → переход в нужный экран».
 */
export function pushToast(text, { kind = 'info', ttl = DEFAULT_TTL_MS, action = null } = {}) {
  const id = nextId++
  toasts.value.push({ id, text, kind, action, createdAt: Date.now() })
  if (toasts.value.length > MAX_VISIBLE) {
    toasts.value.splice(0, toasts.value.length - MAX_VISIBLE)
  }
  setTimeout(() => dismissToast(id), ttl)
}

export function dismissToast(id) {
  const i = toasts.value.findIndex((t) => t.id === id)
  if (i >= 0) toasts.value.splice(i, 1)
}
