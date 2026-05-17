import { onMounted, onUnmounted, ref } from 'vue'
import { formatRemaining, OFFER_TTL_MS } from '../utils/format'

/**
 * Реактивный таймер истечения для pending-офферов.
 *
 *   const { expiresLabel, expiresUrgency } = useOfferExpiry()
 *   expiresLabel(offer)    → 'через 1 ч' | 'скоро' | 'истёк' | ''
 *   expiresUrgency(offer)  → '' | 'muted' | 'warn' | 'critical'
 *
 * Тикаем раз в минуту через ref(now). Vue реактивно перерисует строку
 * без ручного refresh-а. Только для status === 'pending' — остальные
 * статусы возвращают пустую строку, шаблон может v-if-ить по этому.
 */
export function useOfferExpiry() {
  const now = ref(Date.now())
  let timer = null

  onMounted(() => {
    timer = setInterval(() => {
      now.value = Date.now()
    }, 60_000)
  })

  onUnmounted(() => {
    if (timer) clearInterval(timer)
  })

  function offerDeadline(o) {
    if (!o?.created_at) return 0
    const t = new Date(o.created_at).getTime()
    return Number.isFinite(t) ? t + OFFER_TTL_MS : 0
  }

  function expiresLabel(o) {
    if (!o || o.status !== 'pending') return ''
    const d = offerDeadline(o)
    if (!d) return ''
    void now.value // tracking
    return formatRemaining(d)
  }

  function expiresUrgency(o) {
    if (!o || o.status !== 'pending') return ''
    const d = offerDeadline(o)
    if (!d) return ''
    void now.value
    const diff = d - now.value
    if (diff <= 0) return 'critical'
    const hours = diff / 3_600_000
    if (hours < 2) return 'critical'
    if (hours < 24) return 'warn'
    return 'muted'
  }

  return { expiresLabel, expiresUrgency }
}
