import { api } from './client'

export function createOffer({ bouquetId, price }) {
  return api('/offers', {
    method: 'POST',
    body: { bouquet_id: bouquetId, price },
  })
}

export function respondOffer(offerId, { action, price }) {
  return api('/offers/' + offerId + '/respond', {
    method: 'POST',
    body: { action, price },
  })
}

export function getSentOffers(status) {
  return api('/offers/sent', { query: { status } })
}

export function getReceivedOffers(status) {
  return api('/offers/received', { query: { status } })
}

// getAllMyOffers — единый список «всё что меня касается» (отправленные +
// входящие), отсортированный по recency. Каждый item уже содержит role и
// counterparty — фронту не нужно сопоставлять с me.user_id.
export async function getAllMyOffers() {
  const [sent, received] = await Promise.all([getSentOffers(), getReceivedOffers()])
  const all = [...(sent || []), ...(received || [])]
  // sort: новые сверху
  all.sort((a, b) => (a.created_at < b.created_at ? 1 : -1))
  return all
}
