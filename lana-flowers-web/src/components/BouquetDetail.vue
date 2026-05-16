<script setup>
import { computed, ref, watch, onUnmounted } from 'vue'
import { tg, haptic } from '../telegram'
import { me } from '../state/auth'
import { formatPriceKzt } from '../utils/format'

const props = defineProps({
  open: { type: Boolean, required: true },
  // Сырой объект букета из /bouquets (id, title, description, price, city,
  // photos[], category, status, seller{display_name, avatar_url}, my_offer)
  bouquet: { type: Object, default: null },
})
const emit = defineEmits(['close', 'offer'])

// ---- Геттеры ----
const photos = computed(() => props.bouquet?.photos || [])
const title = computed(() => props.bouquet?.title || '')
const description = computed(() => props.bouquet?.description || '')
const city = computed(() => props.bouquet?.city || '')
const sellerName = computed(
  () => props.bouquet?.seller?.display_name || 'Продавец',
)
const sellerAvatar = computed(() => props.bouquet?.seller?.avatar_url || '')
const isOwn = computed(
  () => !!(me.value && props.bouquet?.seller_id === me.value.user_id),
)
const myOffer = computed(() => props.bouquet?.my_offer || null)

const priceDisplay = computed(() => formatPriceKzt(props.bouquet?.price))

const myOfferDisplay = computed(() => {
  if (!myOffer.value) return ''
  return formatPriceKzt(myOffer.value.price)
})

// Шаринг через Telegram. tg.shareLink даёт нативный Telegram «forward
// в чат». Минимальный вирусный канал — юзер пересылает букет другу.
// Использует start_param чтобы получатель открыл мини-апп СРАЗУ на
// нужном букете (через будущий ?bouquet=N deeplink, пока — простой URL).
function shareBouquet() {
  if (!props.bouquet) return
  haptic('light')
  const url = `https://lana-flowers.vercel.app/?bouquet=${props.bouquet.id}`
  const text = `${title.value} — ${priceDisplay.value}`
  if (tg && typeof tg.shareLink === 'function') {
    tg.shareLink(url, text)
  } else if (tg && typeof tg.openTelegramLink === 'function') {
    const tgUrl = `https://t.me/share/url?url=${encodeURIComponent(url)}&text=${encodeURIComponent(text)}`
    tg.openTelegramLink(tgUrl)
  } else {
    window.open(`https://t.me/share/url?url=${encodeURIComponent(url)}&text=${encodeURIComponent(text)}`, '_blank')
  }
}

// ---- Карусель ----
const scrollerRef = ref(null)
const currentPhoto = ref(0)

function onScroll() {
  const el = scrollerRef.value
  if (!el) return
  const i = Math.round(el.scrollLeft / el.clientWidth)
  if (i !== currentPhoto.value) currentPhoto.value = i
}

// При открытии — сбрасываем на первое фото
watch(
  () => props.open,
  (o) => {
    if (o) {
      currentPhoto.value = 0
      // подождать пока DOM смонтируется и скроллер появится
      requestAnimationFrame(() => {
        if (scrollerRef.value) scrollerRef.value.scrollLeft = 0
      })
    }
  },
)

// ---- Telegram BackButton ----
// Когда детальный экран открыт — Telegram'у говорим показать стрелку Назад
// в его собственной шапке. Тап по ней = close (как у нативного NavigationController).
function close() {
  haptic('light')
  emit('close')
}

watch(
  () => props.open,
  (o) => {
    if (!tg?.BackButton) return
    if (o) {
      tg.BackButton.onClick(close)
      tg.BackButton.show()
    } else {
      tg.BackButton.hide()
      tg.BackButton.offClick(close)
    }
  },
)

onUnmounted(() => {
  if (tg?.BackButton) {
    tg.BackButton.hide()
    tg.BackButton.offClick(close)
  }
})

function makeOffer() {
  emit('offer', props.bouquet)
}
</script>

<template>
  <Teleport to="body">
    <div class="detail" :class="{ open }">
      <!-- Единый вертикальный скролл: фотки + контент уезжают вверх вместе.
           CTA снизу зафиксирован снаружи скролла. -->
      <div class="scroll">
        <!-- Карусель фото (горизонтальная). touch-action: pan-x — чтобы
             вертикальные свайпы НЕ цеплялись за этот элемент, а уходили
             в родительский вертикальный скролл. -->
        <div class="gallery">
          <div ref="scrollerRef" class="hslider" @scroll.passive="onScroll">
            <div
              v-for="(url, i) in photos"
              :key="i"
              class="slide"
              :style="{ backgroundImage: `url(${url})` }"
            ></div>
            <div v-if="photos.length === 0" class="slide slide-empty"></div>
          </div>

          <div v-if="photos.length > 1" class="dots">
            <span
              v-for="(_, i) in photos"
              :key="i"
              class="dot"
              :class="{ active: i === currentPhoto }"
            ></span>
          </div>
        </div>

        <div class="content">
          <div class="price-row">
            <div class="price">{{ priceDisplay }}</div>
            <button class="share" type="button" @click="shareBouquet" aria-label="Поделиться">
              <svg viewBox="0 0 24 24" fill="none">
                <path d="M4 12v7a2 2 0 002 2h12a2 2 0 002-2v-7M16 6l-4-4-4 4M12 2v14"
                  stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
          </div>
          <h1 class="title">{{ title }}</h1>

          <div v-if="city" class="meta">
            <svg class="pin" viewBox="0 0 24 24" fill="none">
              <path d="M12 21s-7-6.5-7-12a7 7 0 1114 0c0 5.5-7 12-7 12z" stroke="currentColor" stroke-width="1.5"/>
              <circle cx="12" cy="9.5" r="2.4" stroke="currentColor" stroke-width="1.5"/>
            </svg>
            {{ city }}
          </div>

          <div v-if="description" class="desc-block">
            <div class="sec-title">Описание</div>
            <p class="desc">{{ description }}</p>
          </div>

          <div class="seller-block">
            <div class="sec-title">Продавец</div>
            <div class="seller">
              <div class="ava" :class="{ 'is-placeholder': !sellerAvatar }">
                <img v-if="sellerAvatar" :src="sellerAvatar" :alt="sellerName" />
                <svg v-else viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" />
                </svg>
              </div>
              <div class="sname">{{ sellerName }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Sticky CTA снаружи скролла -->
      <div class="cta-wrap">
        <div v-if="isOwn" class="cta own">Ваше объявление</div>
        <div v-else-if="myOffer" class="cta pending">
          Вы предложили {{ myOfferDisplay }}
        </div>
        <button v-else class="cta primary" type="button" @click="makeOffer">
          Предложить цену
        </button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.detail {
  position: fixed;
  inset: 0;
  background: var(--bg);
  color: var(--text);
  display: flex;
  flex-direction: column;
  z-index: 40;
  transform: translateX(100%);
  transition: transform 0.28s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.detail.open {
  transform: translateX(0);
}

/* Один большой вертикальный скролл — фотки и контент уезжают вверх вместе.
   overscroll-behavior:contain — внутренний bounce не закрывает Telegram. */
.scroll {
  flex: 1;
  overflow-y: auto;
  overscroll-behavior-y: contain;
  -webkit-overflow-scrolling: touch;
}

/* ===== Галерея ===== */
.gallery {
  position: relative;
  background: var(--photo-bg);
}
.hslider {
  display: flex;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  aspect-ratio: 1 / 1;
  /* Ловим ТОЛЬКО горизонтальные свайпы и pinch-zoom. Вертикальные —
     пропускаем в родительский .scroll, иначе скролл вниз заедает на фотках. */
  touch-action: pan-x pinch-zoom;
}
.hslider::-webkit-scrollbar { display: none; }
.slide {
  flex-shrink: 0;
  width: 100%;
  height: 100%;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  scroll-snap-align: start;
}
.slide-empty {
  background: var(--surface-2);
}

.dots {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 10px;
  display: flex;
  justify-content: center;
  gap: 6px;
  pointer-events: none;
}
.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.55);
  box-shadow: 0 0 4px rgba(0, 0, 0, 0.25);
  transition: width 0.2s ease-out, background 0.2s ease-out;
}
.dot.active {
  background: #fff;
  width: 18px;
  border-radius: 3px;
}

/* ===== Контент ===== */
.content {
  padding: 18px 18px 24px;
}
.price-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 4px;
}
.price {
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.01em;
  font-variant-numeric: tabular-nums;
}
.share {
  width: 38px;
  height: 38px;
  border: 0;
  border-radius: 10px;
  background: var(--surface-2);
  color: var(--text);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: background 0.15s, transform 0.1s;
}
.share:active { transform: scale(0.95); background: var(--border); }
.share svg { width: 18px; height: 18px; }
.title {
  font-size: 19px;
  font-weight: 600;
  margin: 0 0 10px;
  letter-spacing: -0.01em;
  line-height: 1.25;
}
.meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 18px;
}
.meta .pin { width: 14px; height: 14px; opacity: 0.7; }

.sec-title {
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-muted);
  margin-bottom: 8px;
}
.desc-block { margin-bottom: 22px; }
.desc {
  margin: 0;
  font-size: 14.5px;
  line-height: 1.55;
  color: var(--text);
  white-space: pre-wrap;
  word-wrap: break-word;
}

.seller-block { margin-bottom: 16px; }
.seller {
  display: flex;
  align-items: center;
  gap: 12px;
}
.ava {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--surface-2);
  flex-shrink: 0;
}
.ava img { width: 100%; height: 100%; object-fit: cover; display: block; }
.ava.is-placeholder {
  display: flex; align-items: center; justify-content: center; color: var(--text-muted);
}
.ava.is-placeholder svg { width: 60%; height: 60%; }
.sname { font-size: 15px; font-weight: 600; }

/* ===== Sticky CTA ===== */
.cta-wrap {
  flex-shrink: 0;
  padding: 12px 16px calc(12px + env(safe-area-inset-bottom, 0px));
  border-top: 1px solid var(--border);
  background: var(--surface);
}
.cta {
  width: 100%;
  height: 52px;
  border-radius: var(--radius-button);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.005em;
  border: 0;
  transition: background 0.15s, transform 0.1s, color 0.15s;
}
.cta.primary {
  background: var(--accent);
  color: var(--accent-text);
}
.cta.primary:active {
  background: var(--accent-hover);
  transform: scale(0.99);
}
.cta.pending {
  background: var(--accent-tint);
  color: var(--accent);
  font-size: 15px;
}
.cta.own {
  background: var(--surface-2);
  color: var(--text-muted);
  font-size: 14px;
  font-weight: 600;
}
</style>
