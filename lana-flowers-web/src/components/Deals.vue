<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { haptic, hapticNotify } from '../telegram'
import { getAllMyOffers, respondOffer } from '../api/offers'
import { useApi } from '../composables/useApi'
import { usePolling } from '../composables/usePolling'
import { formatPrice, formatRemaining, OFFER_TTL_MS } from '../utils/format'
import { thumbUrl } from '../utils/image'
import { confirm, alert } from '../utils/dialog'
import EmptyState from './EmptyState.vue'
import CounterPriceModal from './CounterPriceModal.vue'
import ContactSheet from './ContactSheet.vue'
import PullToRefreshScroll from './base/PullToRefreshScroll.vue'

const emit = defineEmits(['deals-updated'])

const deals = useApi(getAllMyOffers)

async function load() {
  await deals.run()
  emit('deals-updated', actionableCount.value)
}

onMounted(load)
defineExpose({ refresh: load })

// Polling — safety net на случай если SSE-соединение лопнуло. Основной
// канал апдейтов теперь SSE (мгновенно через App.vue → handleEvent).
// 60с достаточно: даже если SSE завис, юзер увидит свежее в течение минуты.
usePolling(load, 60000)

// ---- Фильтр ----
const filter = ref('active') // 'active' | 'history'

const items = computed(() => {
  const list = deals.data.value || []
  if (filter.value === 'active') {
    return list.filter((o) => o.status === 'pending' || o.status === 'accepted')
  }
  return list.filter((o) => o.status !== 'pending' && o.status !== 'accepted')
})

// Сколько офферов ждут МОЕГО ответа (я seller, статус pending) — для бейджа на табе.
const actionableCount = computed(() => {
  return (deals.data.value || []).filter(
    (o) => o.status === 'pending' && o.role === 'seller',
  ).length
})

// ---- Действия ----
const counterModalOpen = ref(false)
const counterOffer = ref(null)

const contactSheetOpen = ref(false)
const contactCounterparty = ref(null)

const busyOfferId = ref(null) // блокируем повторные тапы по строке во время запроса

function openCounter(offer) {
  haptic('light')
  counterOffer.value = offer
  counterModalOpen.value = true
}

async function confirmCounter({ offerId, price }) {
  try {
    await respondOffer(offerId, { action: 'counter', price })
    hapticNotify('success')
    counterModalOpen.value = false
    await load()
  } catch (e) {
    await alert('Не удалось: ' + (e.message || e))
  }
}

async function accept(offer) {
  if (busyOfferId.value) return
  if (!(await confirm(`Принять ${formatPrice(offer.price)} ₸ за «${offer.bouquet.title}»?`))) return
  busyOfferId.value = offer.id
  try {
    await respondOffer(offer.id, { action: 'accept' })
    hapticNotify('success')
    await load()
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
    await load()
  } catch (e) {
    await alert('Ошибка: ' + (e.message || e))
  } finally {
    busyOfferId.value = null
  }
}

async function cancelDeal(offer) {
  if (busyOfferId.value) return
  const ok = await confirm(
    `Отменить сделку? Букет «${offer.bouquet.title}» вернётся в продажу, ${offer.counterparty.name} получит уведомление.`,
  )
  if (!ok) return
  busyOfferId.value = offer.id
  try {
    await respondOffer(offer.id, { action: 'cancel' })
    hapticNotify('warning')
    await load()
  } catch (e) {
    await alert('Ошибка: ' + (e.message || e))
  } finally {
    busyOfferId.value = null
  }
}

// Покупатель забирает свой pending-оффер (передумал). Букет не трогаем,
// продавец получит уведомление об отзыве — оффер не повисит в его inbox'е.
async function withdrawOwn(offer) {
  if (busyOfferId.value) return
  const ok = await confirm(`Отозвать предложение ${formatPrice(offer.price)} ₸ за «${offer.bouquet.title}»?`)
  if (!ok) return
  busyOfferId.value = offer.id
  try {
    await respondOffer(offer.id, { action: 'withdraw' })
    hapticNotify('warning')
    await load()
  } catch (e) {
    await alert('Ошибка: ' + (e.message || e))
  } finally {
    busyOfferId.value = null
  }
}

function showContact(offer) {
  haptic('light')
  contactCounterparty.value = offer.counterparty
  contactSheetOpen.value = true
}

// ---- Таймер истечения для pending-офферов ----
// Бэк auto-expire'ит pending старше OFFER_TTL_MS (см. expire.go). Фронт
// сам считает оставшееся от created_at и обновляет надпись раз в минуту
// — это и есть «таймер», без отдельного API.
const now = ref(Date.now())
let nowTimer = null
onMounted(() => {
  nowTimer = setInterval(() => { now.value = Date.now() }, 60_000)
})
onUnmounted(() => {
  if (nowTimer) clearInterval(nowTimer)
})

function offerDeadline(o) {
  if (!o.created_at) return 0
  const created = new Date(o.created_at).getTime()
  return Number.isFinite(created) ? created + OFFER_TTL_MS : 0
}

function expiresLabel(o) {
  if (o.status !== 'pending') return ''
  const d = offerDeadline(o)
  if (!d) return ''
  // Чтение now.value — для трекинга Vue: при тике перерисовка строки.
  void now.value
  return formatRemaining(d)
}

function expiresUrgency(o) {
  if (o.status !== 'pending') return ''
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

// ---- Хелперы статусов/ролей ----
function statusLabel(o) {
  if (o.status === 'accepted') return 'Принято'
  if (o.status === 'rejected') return 'Отклонено'
  if (o.status === 'countered') return 'Был встречный ответ'
  if (o.status === 'cancelled') return 'Сделка отменена'
  if (o.status === 'expired') return 'Букет ушёл другому'
  // pending
  return o.role === 'seller' ? 'Ждёт вашего ответа' : 'Ожидание ответа'
}

function statusClass(o) {
  if (o.status === 'accepted') return 'ok'
  if (o.status === 'rejected') return 'err'
  if (o.status === 'cancelled' || o.status === 'expired') return 'err'
  if (o.status === 'countered') return 'mute'
  return o.role === 'seller' ? 'warn' : 'mute'
}

function roleLabel(o) {
  return o.role === 'buyer' ? 'Вы предложили' : 'Покупатель предложил'
}
</script>

<template>
  <PullToRefreshScroll :loader="load">
    <div class="deals-inner">
      <div class="filter">
        <button
          type="button"
          class="chip"
          :class="{ active: filter === 'active' }"
          @click="haptic('light'); filter = 'active'"
        >
          Активные
        </button>
        <button
          type="button"
          class="chip"
          :class="{ active: filter === 'history' }"
          @click="haptic('light'); filter = 'history'"
        >
          История
        </button>
      </div>

      <div v-if="deals.loading.value && !deals.data.value" class="status">Загружаю…</div>

      <EmptyState
        v-else-if="items.length === 0"
        :text="filter === 'active' ? 'Активных сделок нет' : 'История пуста'"
      />

      <ul v-else class="list">
        <li v-for="o in items" :key="o.id" class="row">
          <div
            class="thumb"
            :style="{ backgroundImage: `url(${thumbUrl(o.bouquet.photo || '')})` }"
          ></div>

          <div class="body">
            <div class="title">{{ o.bouquet.title }}</div>
            <div class="sub">
              {{ roleLabel(o) }} <strong>{{ formatPrice(o.price) }} ₸</strong>
            </div>
            <div class="meta">
              <span :class="['st', statusClass(o)]">{{ statusLabel(o) }}</span>
              <span class="sep">·</span>
              <span class="cp">{{ o.counterparty.name }}</span>
            </div>

            <div
              v-if="o.status === 'pending'"
              :class="['expires', expiresUrgency(o)]"
            >
              {{ expiresLabel(o) }}
            </div>

            <!-- Действия -->
            <div v-if="o.status === 'pending' && o.role === 'seller'" class="actions">
              <button
                class="act primary"
                :disabled="busyOfferId === o.id"
                @click="accept(o)"
              >
                Принять
              </button>
              <button class="act" @click="openCounter(o)">Встречно</button>
              <button
                class="act danger"
                :disabled="busyOfferId === o.id"
                @click="reject(o)"
              >
                Отклонить
              </button>
            </div>
            <div v-else-if="o.status === 'pending' && o.role === 'buyer'" class="actions">
              <button
                class="act danger ghost"
                :disabled="busyOfferId === o.id"
                @click="withdrawOwn(o)"
              >
                Отозвать предложение
              </button>
            </div>
            <div v-else-if="o.status === 'accepted'" class="actions">
              <button class="act primary" @click="showContact(o)">
                Связаться с {{ o.role === 'buyer' ? 'продавцом' : 'покупателем' }}
              </button>
              <button
                class="act danger ghost"
                :disabled="busyOfferId === o.id"
                @click="cancelDeal(o)"
                title="Если сделка сорвалась — букет вернётся в продажу"
              >
                Не состоялась
              </button>
            </div>
          </div>
        </li>
      </ul>
    </div>

    <CounterPriceModal
      :open="counterModalOpen"
      :offer="counterOffer"
      @close="counterModalOpen = false"
      @confirm="confirmCounter"
    />

    <ContactSheet
      :open="contactSheetOpen"
      :counterparty="contactCounterparty"
      @close="contactSheetOpen = false"
    />
  </PullToRefreshScroll>
</template>

<style scoped>
.deals-inner {
  padding: 8px 16px calc(120px + env(safe-area-inset-bottom, 0px));
}

.filter {
  display: flex;
  gap: 6px;
  padding: 4px 0 14px;
}
.chip {
  flex-shrink: 0;
  font-size: 12.5px;
  padding: 7px 14px;
  border-radius: var(--radius-pill);
  border: 1px solid var(--secondary-border);
  color: var(--secondary-text);
  font-weight: 500;
  background: transparent;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}
.chip.active {
  background: var(--primary);
  color: var(--primary-text);
  border-color: var(--primary);
}

.status, .empty {
  color: var(--text-muted);
  font-size: 14px;
  padding: 32px 0;
  text-align: center;
}

.list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.row {
  display: flex;
  gap: 12px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 12px;
}
.thumb {
  width: 64px;
  height: 64px;
  border-radius: 10px;
  background: var(--surface-2);
  background-size: cover;
  background-position: center;
  flex-shrink: 0;
}
.body {
  flex: 1;
  min-width: 0;
}
.title {
  font-size: 15px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  letter-spacing: -0.005em;
}
.sub {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 2px;
}
.sub strong { color: var(--text); font-weight: 600; }
.meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: var(--text-muted);
  margin-top: 4px;
}
.sep { opacity: 0.6; }
.cp { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.st { font-weight: 600; }
.st.ok { color: #2c8a52; }
.st.err { color: #d6553f; }
.st.warn { color: var(--accent); }
.st.mute { color: var(--text-muted); }

/* Таймер истечения pending-оффера. Цвет = срочность:
   muted (>24ч) → warn бордо (<24ч) → critical красный (<2ч / истёк). */
.expires {
  font-size: 12px;
  margin-top: 4px;
  color: var(--text-muted);
}
.expires.warn {
  color: var(--accent);
  font-weight: 600;
}
.expires.critical {
  color: #d6553f;
  font-weight: 700;
}

.actions {
  display: flex;
  gap: 6px;
  margin-top: 10px;
  flex-wrap: wrap;
}
.act {
  flex: 1;
  min-width: 0;
  padding: 9px 10px;
  border-radius: 8px;
  background: var(--surface-2);
  color: var(--text);
  border: 0;
  font-size: 13px;
  font-weight: 600;
  transition: background 0.15s, transform 0.1s;
}
.act:active:not(:disabled) { transform: scale(0.97); background: var(--border); }
.act:disabled { opacity: 0.5; }
.act.primary {
  background: var(--accent);
  color: var(--accent-text);
}
.act.primary:active:not(:disabled) { background: var(--accent-hover); }
.act.danger {
  color: #d6553f;
}
/* Ghost = «менее заметная» — для деструктивных-но-не-страшных действий
   типа «отменить сделку». Без фона, тонкая обводка. */
.act.danger.ghost {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  font-weight: 500;
}
.act.danger.ghost:active:not(:disabled) {
  background: var(--surface-2);
  color: #d6553f;
  border-color: #d6553f;
}
</style>
