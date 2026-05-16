import { api } from './client'

export function createOffer({ bouquetId, price, message }) {
  return api('/offers', {
    method: 'POST',
    body: { bouquet_id: bouquetId, price, message: message || '' },
  })
}

export function respondOffer(offerId, { action, price, message }) {
  return api('/offers/' + offerId + '/respond', {
    method: 'POST',
    body: { action, price, message: message || '' },
  })
}

export function getSentOffers(status) {
  return api('/offers/sent', { query: { status } })
}

export function getReceivedOffers(status) {
  return api('/offers/received', { query: { status } })
}
