/**
 * Базовый HTTP-клиент к Lana Flowers API.
 *
 * При ответе NOT_REGISTERED — поднимаем needsRegistration, App.vue покажет гейт.
 */

import { needsRegistration } from '../state/auth'

const BASE_URL =
  import.meta.env.VITE_API_URL || 'https://64-226-107-161.nip.io'

function getInitData() {
  return window.Telegram?.WebApp?.initData || ''
}

export class ApiError extends Error {
  constructor(message, status, code) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export async function api(path, { method = 'GET', body, query } = {}) {
  let url = BASE_URL + path
  if (query && Object.keys(query).length) {
    const sp = new URLSearchParams()
    for (const [k, v] of Object.entries(query)) {
      if (v != null && v !== '') sp.set(k, v)
    }
    const qs = sp.toString()
    if (qs) url += '?' + qs
  }

  const headers = {
    'X-Telegram-Init-Data': getInitData(),
  }
  if (body != null) headers['Content-Type'] = 'application/json'

  const res = await fetch(url, {
    method,
    headers,
    body: body != null ? JSON.stringify(body) : undefined,
  })

  let json
  try {
    json = await res.json()
  } catch {
    throw new ApiError(`HTTP ${res.status} (no JSON body)`, res.status, 'BAD_JSON')
  }

  if (!res.ok || json.code !== 'OK') {
    if (json.code === 'NOT_REGISTERED') {
      needsRegistration.value = true
    }
    const msg = json.error || `HTTP ${res.status}`
    throw new ApiError(msg, res.status, json.code || 'UNKNOWN')
  }

  return json.data
}
