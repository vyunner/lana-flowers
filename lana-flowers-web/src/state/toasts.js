// Глобальный стейт тоастов. Любой компонент может pushToast(...).
// Toast.vue рендерит стек и сам убирает через таймер.

import { ref } from 'vue'

export const toasts = ref([]) // [{ id, text, kind, createdAt }]

let nextId = 1
const DEFAULT_TTL_MS = 4500
const MAX_VISIBLE = 3

/**
 * Добавить тост. kind = 'info' | 'success' | 'warn' | 'err'
 */
export function pushToast(text, { kind = 'info', ttl = DEFAULT_TTL_MS } = {}) {
  const id = nextId++
  toasts.value.push({ id, text, kind, createdAt: Date.now() })
  // Ограничиваем стек — старые выпадают
  if (toasts.value.length > MAX_VISIBLE) {
    toasts.value.splice(0, toasts.value.length - MAX_VISIBLE)
  }
  setTimeout(() => dismissToast(id), ttl)
}

export function dismissToast(id) {
  const i = toasts.value.findIndex((t) => t.id === id)
  if (i >= 0) toasts.value.splice(i, 1)
}
