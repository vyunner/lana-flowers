<script setup>
import { computed, onMounted, ref } from 'vue'
import { getAllMyOffers } from '../api/offers'
import { getMyBouquets } from '../api/bouquets'
import { useApi } from '../composables/useApi'
import { usePolling } from '../composables/usePolling'
import { useOfferExpiry } from '../composables/useOfferExpiry'
import { useOfferActions } from '../composables/useOfferActions'
import { formatPrice } from '../utils/format'
import { offerHistoryStatusText, bouquetHistoryStatusText } from '../utils/offers'
import { haptic } from '../telegram'
import CounterPriceModal from './CounterPriceModal.vue'
import ContactSheet from './ContactSheet.vue'
import PullToRefreshScroll from './base/PullToRefreshScroll.vue'
import OfferCardSeller from './deals/OfferCardSeller.vue'
import OfferCardBuyer from './deals/OfferCardBuyer.vue'
import OfferCardAccepted from './deals/OfferCardAccepted.vue'
import BouquetRowSelling from './deals/BouquetRowSelling.vue'
import HistoryRow from './deals/HistoryRow.vue'

const emit = defineEmits(['deals-updated', 'switch-tab'])

// ---- Data ----
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
// 1. Ответьте       — pending где я seller (мой ход, есть таймер)
// 2. Ждёте ответа   — pending где я buyer  (мяч на той стороне, тоже таймер)
// 3. Свяжитесь      — accepted (договариваться, без таймера)
// 4. На продаже     — мои активные букеты БЕЗ офферов (просто витрина)
//
// Букет с активным оффером выше НЕ дублируется в «На продаже» — он уже
// представлен своими офферами в верхних секциях.
const sectionNeedAnswer = computed(() =>
  (deals.data.value || []).filter((o) => o.status === 'pending' && o.role === 'seller'),
)
const sectionWaiting = computed(() =>
  (deals.data.value || []).filter((o) => o.status === 'pending' && o.role === 'buyer'),
)
const sectionNeedContact = computed(() =>
  (deals.data.value || []).filter((o) => o.status === 'accepted'),
)
const sectionSelling = computed(() => {
  const busy = new Set(
    [...sectionNeedAnswer.value, ...sectionNeedContact.value, ...sectionWaiting.value]
      .map((o) => o.bouquet?.id)
      .filter(Boolean),
  )
  return (myBouquets.data.value || []).filter(
    (b) => b.status === 'active' && !busy.has(b.id),
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
    sectionWaiting.value.length === 0 &&
    sectionNeedContact.value.length === 0 &&
    sectionSelling.value.length === 0,
)
const historyEmpty = computed(
  () => sectionHistoryOffers.value.length === 0 && sectionHistoryBouquets.value.length === 0,
)
const historyOpen = ref(false)

// ---- Таймер и действия ----
const expiry = useOfferExpiry()
const a = useOfferActions({ onChange: load })

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
        <!-- 1. ОТВЕТЬТЕ -->
        <section v-if="sectionNeedAnswer.length" class="section">
          <h3 class="sec-title urgent">Ответьте · {{ sectionNeedAnswer.length }}</h3>
          <div class="cards">
            <OfferCardSeller
              v-for="o in sectionNeedAnswer"
              :key="o.id"
              :offer="o"
              :busy="a.busyOfferId.value === o.id"
              :expires-label="expiry.expiresLabel"
              :expires-urgency="expiry.expiresUrgency"
              @accept="a.accept"
              @counter="a.openCounter"
              @reject="a.reject"
            />
          </div>
        </section>

        <!-- 2. ЖДЁТЕ ОТВЕТА -->
        <section v-if="sectionWaiting.length" class="section">
          <h3 class="sec-title">Ждёте ответа · {{ sectionWaiting.length }}</h3>
          <div class="cards">
            <OfferCardBuyer
              v-for="o in sectionWaiting"
              :key="o.id"
              :offer="o"
              :busy="a.busyOfferId.value === o.id"
              :expires-label="expiry.expiresLabel"
              :expires-urgency="expiry.expiresUrgency"
              @withdraw="a.withdrawOwn"
            />
          </div>
        </section>

        <!-- 3. СВЯЖИТЕСЬ -->
        <section v-if="sectionNeedContact.length" class="section">
          <h3 class="sec-title">Свяжитесь · {{ sectionNeedContact.length }}</h3>
          <div class="cards">
            <OfferCardAccepted
              v-for="o in sectionNeedContact"
              :key="o.id"
              :offer="o"
              :busy="a.busyOfferId.value === o.id"
              @contact="a.showContact"
              @cancel="a.cancelDeal"
            />
          </div>
        </section>

        <!-- 4. НА ПРОДАЖЕ -->
        <section v-if="sectionSelling.length" class="section">
          <h3 class="sec-title">На продаже · {{ sectionSelling.length }}</h3>
          <ul class="compact-list">
            <BouquetRowSelling
              v-for="b in sectionSelling"
              :key="b.id"
              :bouquet="b"
              :busy="a.busyBouquetId.value === b.id"
              @remove="a.removeBouquet"
            />
          </ul>
        </section>

        <!-- История toggle -->
        <div v-if="!historyEmpty" class="history-toggle">
          <button class="history-toggle-btn" type="button" @click="historyOpen = !historyOpen">
            {{ historyOpen ? 'Скрыть историю' : 'Показать историю →' }}
          </button>
        </div>

        <template v-if="historyOpen">
          <section v-if="sectionHistoryOffers.length" class="section">
            <h3 class="sec-title muted">Завершённые сделки</h3>
            <ul class="compact-list">
              <HistoryRow
                v-for="o in sectionHistoryOffers"
                :key="o.id"
                :title="o.bouquet.title"
                :sub="`${offerHistoryStatusText(o)} · ${formatPrice(o.price)} ₸`"
                :photo="o.bouquet.photo || ''"
              />
            </ul>
          </section>
          <section v-if="sectionHistoryBouquets.length" class="section">
            <h3 class="sec-title muted">Проданные и снятые букеты</h3>
            <ul class="compact-list">
              <HistoryRow
                v-for="b in sectionHistoryBouquets"
                :key="b.id"
                :title="b.title"
                :sub="`${formatPrice(b.price)} ₸ · ${bouquetHistoryStatusText(b.status)}`"
                :photo="b.photos?.[0] || ''"
              />
            </ul>
          </section>
        </template>
      </template>
    </div>

    <CounterPriceModal
      :open="a.counterModalOpen.value"
      :offer="a.counterOffer.value"
      @close="a.closeCounter"
      @confirm="a.confirmCounter"
    />

    <ContactSheet
      :open="a.contactSheetOpen.value"
      :counterparty="a.contactCounterparty.value"
      @close="a.closeContact"
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

/* Section heading дублируется в _shared.css (карточки/строки видят свой
   scoped-вариант), но Deals.vue рендерит <h3 class="sec-title"> вне
   subcomponent'ов — нужна копия здесь же. */
.sec-title {
  font-size: 13px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-muted);
  margin: 0 4px 8px;
}
.sec-title.urgent { color: var(--accent); }
.sec-title.muted {
  color: var(--text-muted);
  font-weight: 600;
  text-transform: none;
  letter-spacing: 0;
  font-size: 14px;
}

.cards { display: flex; flex-direction: column; gap: 10px; }
.compact-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* Empty state */
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

.section { margin: 0 0 20px; }

.history-toggle {
  text-align: center;
  padding: 8px 0 16px;
}
.history-toggle-btn {
  background: transparent;
  border: 0;
  padding: 6px 4px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
  cursor: pointer;
}
</style>
