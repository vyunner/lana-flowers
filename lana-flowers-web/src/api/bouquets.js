import { api } from './client'

export function listBouquets({ city, category, limit } = {}) {
  return api('/bouquets', { query: { city, category, limit } })
}

export function getBouquet(id) {
  return api('/bouquets/' + id)
}

export function createBouquet(payload) {
  return api('/bouquets', { method: 'POST', body: payload })
}

export function getMyBouquets() {
  return api('/bouquets/my')
}

export function deleteBouquet(id) {
  return api('/bouquets/' + id, { method: 'DELETE' })
}
