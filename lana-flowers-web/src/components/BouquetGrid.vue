<script setup>
import { ref, watch, onMounted, computed } from 'vue'
import { listBouquets } from '../api/bouquets'
import { useApi } from '../composables/useApi'
import { usePolling } from '../composables/usePolling'
import { me } from '../state/auth'
import { formatPriceKzt } from '../utils/format'
import BouquetCard from './BouquetCard.vue'
import EmptyState from './EmptyState.vue'
import PullToRefreshScroll from './base/PullToRefreshScroll.vue'

const props = defineProps({
  category: { type: String, required: true },
  city: { type: String, required: true },
})
const emit = defineEmits(['scroll', 'offer', 'open'])

const fading = ref(false)
const ptrRef = ref(null)

const { data, loading, error, run } = useApi(() =>
  listBouquets({ city: props.city, category: props.category }),
)

async function reload() {
  fading.value = true
  try {
    await run()
  } catch (e) {
    // Ошибка уже в state, рендерится в .status
  } finally {
    requestAnimationFrame(() => (fading.value = false))
  }
}

defineExpose({ refresh: reload })

onMounted(reload)
watch(() => [props.category, props.city], reload)

// Polling 30с — новые объявления появляются + статусы могут протухать
// (sold/archived). Без fading=true чтобы не моргало каждый тик —
// прозрачно подмена данных в .grid.
usePolling(run, 30000)

const items = computed(() => data.value || [])

function adapt(b) {
  return {
    id: b.id,
    price: formatPriceKzt(b.price),
    name: b.title,
    seller: b.seller?.display_name || 'Продавец',
    avatar: b.seller?.avatar_url || '',
    photo: (b.photos && b.photos[0]) || '',
    isOwn: !!(me.value && b.seller_id === me.value.user_id),
    myOffer: b.my_offer
      ? { id: b.my_offer.id, price: formatPriceKzt(b.my_offer.price) }
      : null,
    raw: b,
  }
}
</script>

<template>
  <PullToRefreshScroll ref="ptrRef" @refresh="run" @scroll="(y) => emit('scroll', y)">
    <div class="feed-inner">
      <div v-if="loading && items.length === 0" class="status">
        <span class="spinner"></span>
      </div>

      <div v-else-if="error && items.length === 0" class="status">
        <p>Не удалось загрузить ленту</p>
        <button class="retry" type="button" @click="reload">Попробовать снова</button>
        <p class="err-msg">{{ error.message }}</p>
      </div>

      <div v-else class="grid" :class="{ fading }">
        <BouquetCard
          v-for="b in items"
          :key="b.id"
          :bouquet="adapt(b)"
          @offer="emit('offer', adapt(b))"
          @open="emit('open', b)"
        />
        <EmptyState v-if="items.length === 0" text="В этом городе пока нет букетов" />
      </div>
    </div>
  </PullToRefreshScroll>
</template>

<style scoped>
.feed-inner {
  padding: 0 16px calc(120px + env(safe-area-inset-bottom, 0px));
}

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  transition: opacity 0.25s ease-out;
}
.grid.fading {
  opacity: 0;
}

.status {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 80px 24px;
  text-align: center;
  color: var(--text-secondary);
}
.spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--border);
  border-top-color: var(--text);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.retry {
  background: var(--primary);
  color: var(--primary-text);
  border: 0;
  border-radius: var(--radius-button);
  padding: 10px 18px;
  font-weight: 600;
  font-size: 13px;
}
.err-msg {
  font-size: 12px;
  color: var(--text-muted);
  max-width: 260px;
  margin: 0;
  word-break: break-word;
}
</style>
