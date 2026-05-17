<script setup>
import { formatPrice } from '../../utils/format'
import { cpName } from '../../utils/offers'
import { thumbUrl } from '../../utils/image'

// Карточка pending-оффера для ПОКУПАТЕЛЯ (секция «Ждёте ответа»).
// Мяч на той стороне — единственное действие «Отозвать предложение».
// Hint «X ещё думает» вместо delta — у buyer'а нет «своей цены» как
// ориентира для %.

const props = defineProps({
  offer: { type: Object, required: true },
  busy: { type: Boolean, default: false },
  expiresLabel: { type: Function, required: true },
  expiresUrgency: { type: Function, required: true },
})
defineEmits(['withdraw'])
</script>

<template>
  <article class="card">
    <div class="card-row">
      <div
        class="photo"
        :style="{ backgroundImage: `url(${thumbUrl(offer.bouquet.photo || '')})` }"
      ></div>
      <div class="card-info">
        <div class="card-title">{{ offer.bouquet.title }}</div>
        <div class="card-meta">
          Вы предложили <b>{{ formatPrice(offer.price) }} ₸</b>
          <template v-if="expiresLabel(offer)">
            <span class="sep">·</span>
            <span :class="['inline-time', expiresUrgency(offer)]">{{ expiresLabel(offer) }}</span>
          </template>
        </div>
        <div class="hint">{{ cpName(offer) }} ещё думает</div>
      </div>
    </div>
    <div class="actions">
      <button class="btn secondary" type="button" :disabled="busy" @click="$emit('withdraw', offer)">
        Отозвать предложение
      </button>
    </div>
  </article>
</template>

<style scoped src="./_shared.css"></style>
