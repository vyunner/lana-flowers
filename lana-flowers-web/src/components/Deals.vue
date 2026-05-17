<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { haptic, hapticNotify } from '../telegram'
import { getAllMyOffers, respondOffer } from '../api/offers'
import { getMyBouquets, deleteBouquet } from '../api/bouquets'
import { useApi } from '../composables/useApi'
import { usePolling } from '../composables/usePolling'
import { formatPrice, formatRemaining, OFFER_TTL_MS } from '../utils/format'
import { thumbUrl } from '../utils/image'
import { confirm, alert } from '../utils/dialog'
import CounterPriceModal from './CounterPriceModal.vue'
import ContactSheet from './ContactSheet.vue'
import PullToRefreshScroll from './base/PullToRefreshScroll.vue'

const emit = defineEmits(['deals-updated', 'switch-tab'])

// ---- API ----
const deals = useApi(getAllMyOffers)
const myBouquets = useApi(getMyBouquets)

async function load() {
  await Promise.all([deals.run(), myBouquets.run()])
  emit('deals-updated', sectionNeedAnswer.value.length)
}

onMounted(load)
defineExpose({ refresh: load })

// Polling — safety net. Основной канал — SSE через App.vue → handleEvent.
usePolling(load, 60_000)

// ---- Группировка по «что мне делать сейчас» ----
//
// Принцип: юзер сверху вниз читает экран и сразу понимает срочность.
// 1. Ответьте  — pending где я seller (мой ход)
// 2. Свяжитесь — accepted (надо договориться о встрече)
// 3. Ждёте    — pending где я buyer (мяч на той стороне)
// 4. На продаже — мои активные букеты БЕЗ офферов (просто витрина)
//
// Букет, на который есть активный оффер выше, не дублируется в «На продаже» —
// он уже представлен своими офферами в верхних секциях.

const sectionNeedAnswer = computed(() =>
  (deals.data.value || []).filter((o) => o.status === 'pending' && o.role === 'seller'),
)
const sectionNeedContact = computed(() =>
  (deals.data.value || []).filter((o) => o.status === 'accepted'),
)
const sectionWaiting = computed(() =>
  (deals.data.value || []).filter((o) => o.status === 'pending' && o.role === 'buyer'),
)
const sectionSelling = computed(() => {
  const busyBouquetIds = new Set(
    [
      ...sectionNeedAnswer.value,
      ...sectionNeedContact.value,
      ...sectionWaiting.value,
    ]
      .map((o) => o.bouquet?.id)
      .filter(Boolean),
  )
  return (myBouquets.data.value || []).filter(
    (b) => b.status === 'active' && !busyBouquetIds.has(b.id),
  )
})

const sectionHistoryOffers = computed(() =>
  (deals.data.value || []).filter((o) =>
    ['rejected', 'cancelled', 'expired', 'countered'].includes(o.status),
  ),
)
const sectionHistoryBouquets = computed(() =>
  (myBouquets.data.value || []).filter((b) => b.status !== 'active'),
)

const allEmpty = computed(
  () =>
    sectionNeedAnswer.value.length === 0 &&
    sectionNeedContact.value.length === 0 &&
    sectionWaiting.value.length === 0 &&
    sectionSelling.value.length === 0,
)
const historyEmpty = computed(
  () => sectionHistoryOffers.value.length === 0 && sectionHistoryBouquets.value.length === 0,
)
const historyOpen = ref(false)

// ---- Таймер истечения pending-офферов ----
const now = ref(Date.now())
let nowTimer = null
onMounted(() => {
  nowTimer = setInterval(() => {
    now.value = Date.now()
  }, 60_000)
})
onUnmounted(() => {
  if (nowTimer) clearInterval(nowTimer)
})

function offerDeadline(o) {
  if (!o.created_at) return 0
  const t = new Date(o.created_at).getTime()
  return Number.isFinite(t) ? t + OFFER_TTL_MS : 0
}
function expiresLabel(o) {
  if (o.status !== 'pending') return ''
  const d = offerDeadline(o)
  if (!d) return ''
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

// ---- Хелперы ----
function cpName(o) {
  return o.counterparty?.name || (o.role === 'buyer' ? 'Продавец' : 'Покупатель')
}

// Разница между запросной ценой букета и предложением покупателя в процентах.
// Положительное значение = покупатель просит скидку, отрицательное = предложил больше.
function priceDeltaPct(o) {
  const ask = o.bouquet?.price
  if (!ask || ask <= 0) return null
  const pct = Math.round(((ask - o.price) / ask) * 100)
  return Math.abs(pct) >= 1 ? pct : null
}

function historyStatusText(o) {
  if (o.status === 'rejected') return 'Отклонено'
  if (o.status === 'cancelled') return 'Сделка отменена'
  if (o.status === 'expired') return 'Букет ушёл другому'
  if (o.status === 'countered') return 'Был встречный ответ'
  return o.status
}
function bouquetStatusText(s) {
  if (s === 'sold') return 'Продано'
  if (s === 'archived') return 'Снято'
  return s
}

// ---- Действия с офферами ----
const counterModalOpen = ref(false)
const counterOffer = ref(null)
const contactSheetOpen = ref(false)
const contactCounterparty = ref(null)
const busyOfferId = ref(null)
const busyBouquetId = ref(null)

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
  if (!(await confirm(`Принять ${formatPrice(offer.price)} ₸ за «${offer.bouquet.title}»?`)))
    return
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
    `Отменить сделку? Букет «${offer.bouquet.title}» вернётся в продажу, ${cpName(offer)} получит уведомление.`,
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

// ---- Действия с букетами (мои объявления) ----
async function removeBouquet(b) {
  if (busyBouquetId.value) return
  if (!(await confirm(`Снять «${b.title}» с продажи?`))) return
  busyBouquetId.value = b.id
  try {
    await deleteBouquet(b.id)
    haptic('medium')
    await load()
  } catch (e) {
    await alert('Ошибка: ' + (e.message || e))
  } finally {
    busyBouquetId.value = null
  }
}

// ---- Empty-state CTA ----
function goCatalog() {
  haptic('light')
  emit('switch-tab', 'catalog')
}
</script>

<template>
  <PullToRefreshScroll :loader="load">
    <div class="deals-inner">
      <!-- Empty state: всё спокойно, идти в каталог -->
      <div
        v-if="!deals.loading.value && !myBouquets.loading.value && allEmpty"
        class="empty-state"
      >
        <p class="empty-title">Сейчас всё спокойно</p>
        <p class="empty-sub">Загляните в каталог — там новые букеты</p>
        <button class="empty-cta" type="button" @click="goCatalog">Открыть каталог →</button>
      </div>

      <!-- Loading -->
      <div v-else-if="deals.loading.value && !deals.data.value" class="status">Загружаю…</div>

      <template v-else>
        <!-- 1. ОТВЕТЬТЕ — pending, я seller -->
        <section v-if="sectionNeedAnswer.length" class="section">
          <h3 class="sec-title urgent">Ответьте · {{ sectionNeedAnswer.length }}</h3>
          <div class="cards">
            <article v-for="o in sectionNeedAnswer" :key="o.id" class="card">
              <div class="card-row">
                <div
                  class="photo"
                  :style="{ backgroundImage: `url(${thumbUrl(o.bouquet.photo || '')})` }"
                ></div>
                <div class="card-info">
                  <div class="card-title">{{ o.bouquet.title }}</div>
                  <div class="card-meta">
                    От {{ cpName(o) }}
                    <template v-if="expiresLabel(o)">
                      <span class="sep">·</span>
                      <span :class="['inline-time', expiresUrgency(o)]">{{ expiresLabel(o) }}</span>
                    </template>
                  </div>
                  <div
                    v-if="priceDeltaPct(o) != null"
                    :class="['delta-line', priceDeltaPct(o) > 0 ? 'down' : 'up']"
                  >
                    {{ priceDeltaPct(o) > 0 ? '−' : '+' }}{{ Math.abs(priceDeltaPct(o)) }}% от вашей цены ({{ formatPrice(o.bouquet.price) }} ₸)
                  </div>
                </div>
              </div>
              <div class="actions">
                <button
                  class="btn primary"
                  :disabled="busyOfferId === o.id"
                  @click="accept(o)"
                >
                  Согласиться на {{ formatPrice(o.price) }} ₸
                </button>
                <div class="btn-row">
                  <button class="btn secondary" type="button" @click="openCounter(o)">
                    Предложить свою
                  </button>
                  <button
                    class="btn secondary danger"
                    type="button"
                    :disabled="busyOfferId === o.id"
                    @click="reject(o)"
                  >
                    Отклонить
                  </button>
                </div>
              </div>
            </article>
          </div>
        </section>

        <!-- 2. СВЯЖИТЕСЬ — accepted -->
        <section v-if="sectionNeedContact.length" class="section">
          <h3 class="sec-title">Свяжитесь · {{ sectionNeedContact.length }}</h3>
          <div class="cards">
            <article v-for="o in sectionNeedContact" :key="o.id" class="card">
              <div class="card-row">
                <div
                  class="photo"
                  :style="{ backgroundImage: `url(${thumbUrl(o.bouquet.photo || '')})` }"
                ></div>
                <div class="card-info">
                  <div class="card-title">{{ o.bouquet.title }}</div>
                  <div class="card-meta">
                    Договорились с {{ cpName(o) }} —
                    <b>{{ formatPrice(o.price) }} ₸</b>
                  </div>
                </div>
              </div>
              <div class="actions">
                <button class="btn primary" type="button" @click="showContact(o)">
                  Связаться с {{ o.role === 'buyer' ? 'продавцом' : 'покупателем' }}
                </button>
                <button
                  class="btn secondary danger"
                  type="button"
                  :disabled="busyOfferId === o.id"
                  @click="cancelDeal(o)"
                >
                  Отменить сделку
                </button>
              </div>
            </article>
          </div>
        </section>

        <!-- 3. ЖДЁТЕ ОТВЕТА — pending, я buyer -->
        <section v-if="sectionWaiting.length" class="section">
          <h3 class="sec-title">Ждёте ответа · {{ sectionWaiting.length }}</h3>
          <div class="cards">
            <article v-for="o in sectionWaiting" :key="o.id" class="card">
              <div class="card-row">
                <div
                  class="photo"
                  :style="{ backgroundImage: `url(${thumbUrl(o.bouquet.photo || '')})` }"
                ></div>
                <div class="card-info">
                  <div class="card-title">{{ o.bouquet.title }}</div>
                  <div class="card-meta">
                    Вы предложили <b>{{ formatPrice(o.price) }} ₸</b>
                    <template v-if="expiresLabel(o)">
                      <span class="sep">·</span>
                      <span :class="['inline-time', expiresUrgency(o)]">{{ expiresLabel(o) }}</span>
                    </template>
                  </div>
                  <div class="hint">{{ cpName(o) }} ещё думает</div>
                </div>
              </div>
              <div class="actions">
                <button
                  class="btn secondary"
                  type="button"
                  :disabled="busyOfferId === o.id"
                  @click="withdrawOwn(o)"
                >
                  Отозвать предложение
                </button>
              </div>
            </article>
          </div>
        </section>

        <!-- 4. НА ПРОДАЖЕ — мои активные без офферов -->
        <section v-if="sectionSelling.length" class="section">
          <h3 class="sec-title">На продаже · {{ sectionSelling.length }}</h3>
          <ul class="compact-list">
            <li v-for="b in sectionSelling" :key="b.id" class="compact-row">
              <div
                class="compact-thumb"
                :style="{ backgroundImage: `url(${thumbUrl(b.photos?.[0] || '')})` }"
              ></div>
              <div class="compact-body">
                <div class="compact-title">{{ b.title }}</div>
                <div class="compact-sub">{{ formatPrice(b.price) }} ₸</div>
              </div>
              <button
                class="link danger compact-action"
                type="button"
                :disabled="busyBouquetId === b.id"
                @click="removeBouquet(b)"
              >
                Снять
              </button>
            </li>
          </ul>
        </section>

        <!-- История toggle -->
        <div v-if="!historyEmpty" class="history-toggle">
          <button class="link muted" type="button" @click="historyOpen = !historyOpen">
            {{ historyOpen ? 'Скрыть историю' : 'Показать историю →' }}
          </button>
        </div>

        <template v-if="historyOpen">
          <section v-if="sectionHistoryOffers.length" class="section">
            <h3 class="sec-title muted">Завершённые сделки</h3>
            <ul class="compact-list">
              <li
                v-for="o in sectionHistoryOffers"
                :key="o.id"
                class="compact-row history-row"
              >
                <div
                  class="compact-thumb"
                  :style="{ backgroundImage: `url(${thumbUrl(o.bouquet.photo || '')})` }"
                ></div>
                <div class="compact-body">
                  <div class="compact-title">{{ o.bouquet.title }}</div>
                  <div class="compact-sub">
                    {{ historyStatusText(o) }} · {{ formatPrice(o.price) }} ₸
                  </div>
                </div>
              </li>
            </ul>
          </section>
          <section v-if="sectionHistoryBouquets.length" class="section">
            <h3 class="sec-title muted">Проданные и снятые букеты</h3>
            <ul class="compact-list">
              <li
                v-for="b in sectionHistoryBouquets"
                :key="b.id"
                class="compact-row history-row"
              >
                <div
                  class="compact-thumb"
                  :style="{ backgroundImage: `url(${thumbUrl(b.photos?.[0] || '')})` }"
                ></div>
                <div class="compact-body">
                  <div class="compact-title">{{ b.title }}</div>
                  <div class="compact-sub">
                    {{ formatPrice(b.price) }} ₸ · {{ bouquetStatusText(b.status) }}
                  </div>
                </div>
              </li>
            </ul>
          </section>
        </template>
      </template>
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

.status {
  color: var(--text-muted);
  font-size: 14px;
  padding: 32px 0;
  text-align: center;
}

/* ---- Empty state ---- */
.empty-state {
  padding: 60px 24px;
  text-align: center;
}
.empty-title {
  font-size: 18px;
  font-weight: 700;
  margin: 0 0 6px;
  color: var(--text);
}
.empty-sub {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 18px;
}
.empty-cta {
  background: var(--primary);
  color: var(--primary-text);
  border: 0;
  border-radius: var(--radius-pill);
  padding: 11px 22px;
  font-weight: 600;
  font-size: 14px;
}

/* ---- Секции ---- */
.section {
  margin: 0 0 20px;
}
.sec-title {
  font-size: 13px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-muted);
  margin: 0 4px 8px;
}
.sec-title.urgent {
  color: var(--accent);
}
.sec-title.muted {
  color: var(--text-muted);
  font-weight: 600;
  text-transform: none;
  letter-spacing: 0;
  font-size: 14px;
}

/* ---- Карточка оффера ---- */
.cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.card-row {
  display: flex;
  gap: 14px;
}
.photo {
  width: 80px;
  height: 80px;
  border-radius: 12px;
  background: var(--surface-2);
  background-size: cover;
  background-position: center;
  flex-shrink: 0;
}
.card-info {
  flex: 1;
  min-width: 0;
}
.card-title {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* Meta-строка: «От Иван · через 1 ч» / «Вы предложили 9 500 ₸ · через 1 ч» —
   всё, что нужно увидеть с первого взгляда, в одну строку под названием. */
.card-meta {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 3px;
  line-height: 1.35;
}
.card-meta b {
  color: var(--text);
  font-weight: 700;
}
.card-meta .sep {
  margin: 0 6px;
  color: var(--text-muted);
}

/* Inline-таймер истечения (часть .card-meta). Цвет = срочность. */
.inline-time {
  color: var(--text-muted);
}
.inline-time.warn {
  color: var(--accent);
  font-weight: 600;
}
.inline-time.critical {
  color: #d6553f;
  font-weight: 700;
}

/* Дельта vs запрошенной цены — показывается ТОЛЬКО когда есть разница.
   Скидку (покупатель просит меньше) — красным, надбавку — зелёным. */
.delta-line {
  font-size: 12px;
  font-weight: 600;
  margin-top: 4px;
}
.delta-line.down {
  color: #d6553f;
}
.delta-line.up {
  color: #2c8a52;
}

.hint {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 4px;
  font-style: italic;
}

/* ---- Действия в карточке ---- */
/* Primary CTA сверху, под ней опционально ряд secondary-кнопок 50/50.
   Все три типа — это filled-кнопки с одинаковой высотой и border-radius,
   разница в фоне: primary = бордовый акцент, secondary = серый surface-2,
   danger-вариант secondary окрашивает только текст в красный. */
.actions {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.btn-row {
  display: flex;
  gap: 6px;
}
.btn-row > .btn {
  flex: 1;
  min-width: 0;
}
.btn {
  width: 100%;
  padding: 10px 12px;
  border-radius: 10px;
  border: 0;
  font-size: 13px;
  font-weight: 600;
  transition: background 0.15s, transform 0.1s;
}
.btn:active:not(:disabled) {
  transform: scale(0.98);
}
.btn:disabled {
  opacity: 0.5;
}
.btn.primary {
  background: var(--accent);
  color: var(--accent-text);
  font-weight: 700;
}
.btn.primary:active:not(:disabled) {
  background: var(--accent-hover);
}
.btn.secondary {
  background: var(--surface-2);
  color: var(--text);
}
.btn.secondary:active:not(:disabled) {
  background: var(--border);
}
.btn.secondary.danger {
  color: #d6553f;
}

/* ---- Text-links (для строк-списков «Снять», «Показать историю») ---- */
.link {
  background: transparent;
  border: 0;
  padding: 6px 4px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  cursor: pointer;
}
.link:active:not(:disabled) {
  color: var(--text);
}
.link:disabled {
  opacity: 0.5;
}
.link.danger {
  color: #d6553f;
}
.link.muted {
  color: var(--text-muted);
  font-weight: 500;
}

/* ---- Компактный список (мои букеты, история) ---- */
.compact-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.compact-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
}
.compact-row.history-row {
  border-color: transparent;
  background: var(--surface-2);
}
.compact-thumb {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  background: var(--surface-2);
  background-size: cover;
  background-position: center;
  flex-shrink: 0;
}
.compact-body {
  flex: 1;
  min-width: 0;
}
.compact-title {
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.compact-sub {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
}
.compact-action {
  flex-shrink: 0;
}

/* ---- История toggle ---- */
.history-toggle {
  text-align: center;
  padding: 8px 0 16px;
}
</style>
