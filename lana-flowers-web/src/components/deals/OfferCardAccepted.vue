<script setup>
import { formatPrice } from '../../utils/format'
import { cpName } from '../../utils/offers'
import { thumbUrl } from '../../utils/image'

// Карточка accepted-сделки (секция «Свяжитесь»).
// Primary действие зелёное «Связаться с …» (positive resolved-state),
// secondary danger «Отменить сделку» (редкий аварийный путь).
// Без таймера — сделка уже состоялась, спешить некуда.

const props = defineProps({
  offer: { type: Object, required: true },
  busy: { type: Boolean, default: false },
})
defineEmits(['contact', 'cancel'])
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
          Договорились с {{ cpName(offer) }} —
          <b>{{ formatPrice(offer.price) }} ₸</b>
        </div>
      </div>
    </div>
    <div class="actions">
      <button class="btn success" type="button" @click="$emit('contact', offer)">
        Связаться с {{ offer.role === 'buyer' ? 'продавцом' : 'покупателем' }}
      </button>
      <button class="btn secondary danger" type="button" :disabled="busy" @click="$emit('cancel', offer)">
        Отменить сделку
      </button>
    </div>
  </article>
</template>

<style scoped src="./_shared.css"></style>
