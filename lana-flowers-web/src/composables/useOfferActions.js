import { ref } from 'vue'
import { haptic, hapticNotify } from '../telegram'
import { respondOffer } from '../api/offers'
import { deleteBouquet } from '../api/bouquets'
import { formatPrice } from '../utils/format'
import { cpName } from '../utils/offers'
import { confirm, alert } from '../utils/dialog'

/**
 * Хук для всех действий со сделками: accept / reject / counter (через
 * модалку) / cancel / withdraw / контакт / снять букет.
 *
 *   const a = useOfferActions({ onChange: load })
 *   a.accept(offer); a.reject(offer); ...
 *   <CounterPriceModal :open="a.counterModalOpen" :offer="a.counterOffer"
 *     @close="a.closeCounter" @confirm="a.confirmCounter" />
 *   <ContactSheet :open="a.contactSheetOpen"
 *     :counterparty="a.contactCounterparty" @close="a.closeContact" />
 *
 * busyOfferId / busyBouquetId — блокируем повторные тапы по той же строке
 * пока запрос летит, иначе двойной accept или удаление в гонку.
 *
 * onChange — колбэк после успешной мутации, обычно load() из родителя
 * чтобы перетянуть данные. Если не передан — caller должен сам слушать
 * SSE-event и обновлять (но удобнее передать).
 */
export function useOfferActions({ onChange } = {}) {
  const busyOfferId = ref(null)
  const busyBouquetId = ref(null)

  const counterModalOpen = ref(false)
  const counterOffer = ref(null)

  const contactSheetOpen = ref(false)
  const contactCounterparty = ref(null)

  async function reload() {
    if (typeof onChange === 'function') {
      try { await onChange() } catch {}
    }
  }

  // ---- Counter (через модалку) ----
  function openCounter(offer) {
    haptic('light')
    counterOffer.value = offer
    counterModalOpen.value = true
  }
  function closeCounter() {
    counterModalOpen.value = false
  }
  async function confirmCounter({ offerId, price }) {
    try {
      await respondOffer(offerId, { action: 'counter', price })
      hapticNotify('success')
      counterModalOpen.value = false
      await reload()
    } catch (e) {
      await alert('Не удалось: ' + (e.message || e))
    }
  }

  // ---- Accept / Reject (продавец) ----
  async function accept(offer) {
    if (busyOfferId.value) return
    if (!(await confirm(`Принять ${formatPrice(offer.price)} ₸ за «${offer.bouquet.title}»?`))) {
      return
    }
    busyOfferId.value = offer.id
    try {
      await respondOffer(offer.id, { action: 'accept' })
      hapticNotify('success')
      await reload()
    } catch (e) {
      await alert('Ошибка: ' + (e.message || e))
    } finally {
      busyOfferId.value = null
    }
  }

  async function reject(offer) {
    if (busyOfferId.value) return
    if (!(await confirm(`Отклонить предложение ${formatPrice(offer.price)} ₸?`))) return
    busyOfferId.value = offer.id
    try {
      await respondOffer(offer.id, { action: 'reject' })
      hapticNotify('warning')
      await reload()
    } catch (e) {
      await alert('Ошибка: ' + (e.message || e))
    } finally {
      busyOfferId.value = null
    }
  }

  // ---- Cancel (отменить принятую сделку) ----
  async function cancelDeal(offer) {
    if (busyOfferId.value) return
    const ok = await confirm(
      `Отменить сделку? Букет «${offer.bouquet.title}» вернётся в продажу, ${cpName(offer)} получит уведомление.`,
    )
    if (!ok) return
    busyOfferId.value = offer.id
    try {
      await respondOffer(offer.id, { action: 'cancel' })
      hapticNotify('warning')
      await reload()
    } catch (e) {
      await alert('Ошибка: ' + (e.message || e))
    } finally {
      busyOfferId.value = null
    }
  }

  // ---- Withdraw (покупатель отзывает свой pending) ----
  async function withdrawOwn(offer) {
    if (busyOfferId.value) return
    const ok = await confirm(
      `Отозвать предложение ${formatPrice(offer.price)} ₸ за «${offer.bouquet.title}»?`,
    )
    if (!ok) return
    busyOfferId.value = offer.id
    try {
      await respondOffer(offer.id, { action: 'withdraw' })
      hapticNotify('warning')
      await reload()
    } catch (e) {
      await alert('Ошибка: ' + (e.message || e))
    } finally {
      busyOfferId.value = null
    }
  }

  // ---- Контакт после accepted ----
  function showContact(offer) {
    haptic('light')
    contactCounterparty.value = offer.counterparty
    contactSheetOpen.value = true
  }
  function closeContact() {
    contactSheetOpen.value = false
  }

  // ---- Снять букет с продажи ----
  async function removeBouquet(b) {
    if (busyBouquetId.value) return
    if (!(await confirm(`Снять «${b.title}» с продажи?`))) return
    busyBouquetId.value = b.id
    try {
      await deleteBouquet(b.id)
      haptic('medium')
      await reload()
    } catch (e) {
      await alert('Ошибка: ' + (e.message || e))
    } finally {
      busyBouquetId.value = null
    }
  }

  return {
    busyOfferId,
    busyBouquetId,
    counterModalOpen,
    counterOffer,
    contactSheetOpen,
    contactCounterparty,
    openCounter,
    closeCounter,
    confirmCounter,
    accept,
    reject,
    cancelDeal,
    withdrawOwn,
    showContact,
    closeContact,
    removeBouquet,
  }
}
