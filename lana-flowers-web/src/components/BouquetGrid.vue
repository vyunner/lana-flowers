<script setup>
import { ref, watch, onMounted, computed } from 'vue'
import { listBouquets } from '../api/bouquets'
import { useApi } from '../composables/useApi'
import { usePullToRefresh } from '../composables/usePullToRefresh'
import { me } from '../state/auth'
import BouquetCard from './BouquetCard.vue'
import EmptyState from './EmptyState.vue'

const props = defineProps({
  category: { type: String, required: true },
  city: { type: String, required: true },
})
const emit = defineEmits(['scroll', 'offer'])

const fading = ref(false)

const { data, loading, error, run } = useApi(() =>
  listBouquets({ city: props.city, category: props.category }),
)

async function reload() {
  fading.value = true
  try {
    await run()
  } catch (e) {
    // Ошибка уже в state
  } finally {
    requestAnimationFrame(() => (fading.value = false))
  }
}

defineExpose({ refresh: reload })

const feedRef = ref(null)
const { pullDistance, refreshing, dragging } = usePullToRefresh(feedRef, run)

onMounted(reload)
watch(() => [props.category, props.city], reload)

const items = computed(() => data.value || [])

const ruFmt = new Intl.NumberFormat('ru-RU')

function adapt(b) {
  return {
    id: b.id,
    price: ruFmt.format(b.price) + ' ₸',
    name: b.title,
    seller: b.seller?.display_name || 'Продавец',
    avatar: b.seller?.avatar_url || '',
    photo: (b.photos && b.photos[0]) || '',
    isOwn: !!(me.value && b.seller_id === me.value.user_id),
    // pending-оффер текущего юзера на этот букет (если есть)
    myOffer: b.my_offer ? { id: b.my_offer.id, price: ruFmt.format(b.my_offer.price) + ' ₸' } : null,
    raw: b,
  }
}

let ticking = false
function onScroll(e) {
  if (ticking) return
  ticking = true
  const y = e.target.scrollTop
  requestAnimationFrame(() => {
    emit('scroll', y)
    ticking = false
  })
}
</script>

<template>
  <div ref="feedRef" class="feed" @scroll.passive="onScroll">
    <div
      class="feed-inner"
      :style="{
        transform: `translateY(${pullDistance}px)`,
        transition: dragging ? 'none' : 'transform .25s cubic-bezier(.2,.8,.2,1)',
      }"
    >
      <div class="ptr" :class="{ spinning: refreshing || pullDistance >= 60 }">
        <span class="ptr-spinner"></span>
      </div>

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
        />
        <EmptyState v-if="items.length === 0" text="В этом городе пока нет букетов" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.feed {
  position: relative;
  flex: 1;
  overflow-y: auto;
  /* iOS: убираем системный bounce — будет наш pull-to-refresh */
  overscroll-behavior-y: contain;
}
.feed-inner {
  position: relative; /* якорь для абсолютного .ptr */
  padding: 0 16px calc(120px + env(safe-area-inset-bottom, 0px));
  will-change: transform;
}

/* Спиннер ABSOLUTE'ом над верхним краем inner'а — он не в потоке, поэтому
   карточки начинаются ровно с y=0 и НЕ обрезаются. На pull inner ползёт вниз
   вместе со спиннером (transform применяется к родителю → к children тоже). */
.ptr {
  position: absolute;
  top: -52px;
  left: 0;
  right: 0;
  height: 52px;
  display: flex;
  justify-content: center;
  align-items: flex-end;
  pointer-events: none;
  padding-bottom: 14px;
}
.ptr-spinner {
  width: 22px;
  height: 22px;
  border: 2px solid var(--border);
  border-top-color: var(--text);
  border-radius: 50%;
}
.ptr.spinning .ptr-spinner {
  animation: spin 0.8s linear infinite;
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
  flex: 1;
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
@keyframes spin {
  to { transform: rotate(360deg); }
}
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
