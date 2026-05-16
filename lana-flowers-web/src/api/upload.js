/**
 * Upload фото на бэк. Возвращает абсолютный URL картинки.
 *
 * Используем raw fetch (не client.js) потому что multipart/form-data —
 * fetch сам выставит boundary, нельзя задавать Content-Type вручную.
 */

const BASE_URL =
  import.meta.env.VITE_API_URL || 'https://64-226-107-161.nip.io'

function getInitData() {
  return window.Telegram?.WebApp?.initData || ''
}

export async function uploadPhoto(file) {
  const fd = new FormData()
  fd.append('file', file)

  const res = await fetch(BASE_URL + '/upload', {
    method: 'POST',
    headers: { 'X-Telegram-Init-Data': getInitData() },
    body: fd,
  })

  const json = await res.json().catch(() => null)
  if (!res.ok || !json || json.code !== 'OK') {
    const msg = json?.error || `HTTP ${res.status}`
    throw new Error(msg)
  }
  // Бэк возвращает относительный путь /uploads/abc.jpg → префиксим базой.
  return BASE_URL + json.data.url
}
