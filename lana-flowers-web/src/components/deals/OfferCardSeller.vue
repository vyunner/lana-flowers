<script setup>
import { formatPrice } from '../../utils/format'
import { cpName, priceDeltaPct } from '../../utils/offers'
import { thumbUrl } from '../../utils/image'

// Карточка pending-оффера для ПРОДАВЦА (секция «Ответьте»).
// Три действия: согласиться (primary бордо) / встречка (secondary) /
// отклонить (secondary danger). Показывает дельту цены если она >=1%.

const props = defineProps({
  offer: { type: Object, required: true },
  busy: { type: Boolean, default: false },
  // expiresLabel/expiresUrgency — результат useOfferExpiry из родителя.
  // Прокидываем функции а не значения чтобы карточка реактивно перерисовывалась
  // при tick'е таймера в композабле.
  expiresLabel: { type: Function, required: true },
  expiresUrgency: { type: Function, required: true },
})
defineEmits(['accept', 'counter', 'reject'])
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
          От {{ cpName(offer) }}
          <template v-if="expiresLabel(offer)">
            <span class="sep">·</span>
            <span :class="['inline-time', expiresUrgency(offer)]">{{ expiresLabel(offer) }}</span>
          </template>
        </div>
        <div
          v-if="priceDeltaPct(offer) != null"
          :class="['delta-line', priceDeltaPct(offer) > 0 ? 'down' : 'up']"
        >
          {{ priceDeltaPct(offer) > 0 ? '−' : '+' }}{{ Math.abs(priceDeltaPct(offer)) }}% от вашей цены ({{ formatPrice(offer.bouquet.price) }} ₸)
        </div>
      </div>
    </div>
    <div class="actions">
      <button class="btn primary" :disabled="busy" @click="$emit('accept', offer)">
        Согласиться на {{ formatPrice(offer.price) }} ₸
      </button>
      <div class="btn-row">
        <button class="btn secondary" type="button" @click="$emit('counter', offer)">
          Предложить свою
        </button>
        <button
          class="btn secondary danger"
          type="button"
          :disabled="busy"
          @click="$emit('reject', offer)"
        >
          Отклонить
        </button>
      </div>
    </div>
  </article>
</template>

<style scoped src="./_shared.css"></style>
