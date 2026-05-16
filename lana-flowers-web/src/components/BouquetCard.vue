<script setup>
import { ref } from 'vue'
import { thumbUrl } from '../utils/image'

const props = defineProps({
  bouquet: { type: Object, required: true },
})
defineEmits(['offer', 'open'])

// useThumb=false → перешли на оригинал после 404 на thumb (старые фотки
// до бэк-генерации не имеют _thumb.jpg). Один раз на карточку.
const useThumb = ref(true)
function onPhotoError() { useThumb.value = false }
</script>

<template>
  <!-- Вся карточка кликабельная → открывает деталь. Кнопка offer ниже
       тоже принимает тапы, но через @click.stop — иначе оба хендлера
       сработают и юзер уйдёт в деталь вместо отправки оффера. -->
  <article class="card" @click="$emit('open', bouquet)">
    <img
      v-if="bouquet.photo"
      class="photo"
      :src="useThumb ? thumbUrl(bouquet.photo) : bouquet.photo"
      :alt="bouquet.name"
      loading="lazy"
      decoding="async"
      @error="onPhotoError"
    />
    <div v-else class="photo placeholder"></div>
    <div class="body">
      <div class="price">{{ bouquet.price }}</div>
      <div class="name">{{ bouquet.name }}</div>
      <div class="seller">
        <span
          v-if="bouquet.avatar"
          class="ava"
          :style="{ backgroundImage: `url('${bouquet.avatar}')` }"
        ></span>
        <span v-else class="ava is-placeholder">
          <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" />
          </svg>
        </span>
        <span class="seller-name">{{ bouquet.seller }}</span>
      </div>
      <button
        v-if="!bouquet.isOwn && !bouquet.myOffer"
        class="offer-btn"
        type="button"
        @click.stop="$emit('offer', bouquet)"
      >
        Предложить цену
      </button>
      <div v-else-if="bouquet.myOffer" class="own-badge pending">
        Предложено: {{ bouquet.myOffer.price }}
      </div>
      <div v-else class="own-badge">Ваше объявление</div>
    </div>
  </article>
</template>

<style scoped>
.card {
  background: var(--surface);
  border-radius: var(--radius-card);
  overflow: hidden;
  border: 1px solid var(--border);
  display: flex;
  flex-direction: column;
}

.photo {
  width: 100%;
  aspect-ratio: 4 / 5;
  object-fit: cover;
  display: block;
  background-color: var(--photo-bg);
}
.photo.placeholder {
  /* «дырка» если photo пустой — серый прямоугольник вместо broken-image */
  background-color: var(--photo-bg);
}

.body {
  padding: 10px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.price {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: -0.01em;
  color: var(--text);
}

.name {
  font-size: 13px;
  font-weight: 400;
  color: var(--text-secondary);
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.seller {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
  color: var(--text);
  line-height: 1;
  margin-top: 2px;
}
.seller .ava {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background-size: cover;
  background-position: center;
  background-color: var(--surface-2);
  flex-shrink: 0;
}
.seller .ava.is-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}
.seller .ava.is-placeholder svg {
  width: 70%;
  height: 70%;
}
.seller .seller-name {
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

/* "Предложить цену" — tonal accent: тихий, но узнаваемый brand color.
   Без solid fill чтобы лента не превращалась в столб burgundy. */
.offer-btn {
  margin-top: 4px;
  width: 100%;
  padding: 9px 0;
  background: var(--accent-tint);
  color: var(--accent);
  border: 0;
  border-radius: var(--radius-button);
  font-size: 12.5px;
  font-weight: 600;
  letter-spacing: -0.005em;
  transition:
    background 0.15s ease-out,
    transform 0.1s ease-out;
}
.offer-btn:active {
  transform: scale(0.97);
  background: var(--accent-tint-hover);
}

.own-badge {
  margin-top: 4px;
  width: 100%;
  padding: 9px 0;
  text-align: center;
  background: var(--surface-2);
  color: var(--text-secondary);
  border-radius: var(--radius-button);
  font-size: 12.5px;
  font-weight: 600;
  letter-spacing: -0.005em;
}
.own-badge.pending {
  background: var(--accent-tint);
  color: var(--accent);
}
</style>
