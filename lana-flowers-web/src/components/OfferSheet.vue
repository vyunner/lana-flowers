<script setup>
import { computed, ref, watch } from 'vue'
import { createOffer } from '../api/offers'
import { formatPrice, parsePrice } from '../utils/format'
import BaseModal from './base/BaseModal.vue'
import StatusState from './base/StatusState.vue'

const props = defineProps({
  open: { type: Boolean, required: true },
  bouquet: { type: Object, default: null },
})
const emit = defineEmits(['close', 'submitted'])

const sellerPrice = computed(() => (props.bouquet ? parsePrice(props.bouquet.price) : 0))
const value = ref(0)
const submitting = ref(false)
const errorText = ref('')
const selfError = ref(false)
const duplicateError = ref(false)
const unavailableError = ref(false)
const sentState = ref(false)
const sentPrice = ref(0)

watch(
  () => props.open,
  (o) => {
    if (o) {
      value.value = sellerPrice.value
      errorText.value = ''
      selfError.value = false
      duplicateError.value = false
      unavailableError.value = false
      sentState.value = false
      sentPrice.value = 0
    }
  },
)

function adjust(delta) {
  value.value = Math.max(0, value.value + delta)
}

const canSubmit = computed(() => value.value > 0 && !submitting.value)
const diff = computed(() => value.value - sellerPrice.value)

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  errorText.value = ''
  try {
    const bouquetId = props.bouquet?.raw?.id ?? props.bouquet?.id
    const result = await createOffer({ bouquetId, price: value.value })
    sentPrice.value = value.value
    sentState.value = true
    emit('submitted', {
      bouquet: props.bouquet,
      price: value.value,
      ...result,
      keepOpen: true,
    })
  } catch (e) {
    if (e.code === 'SELF_OFFER') {
      selfError.value = true
    } else if (e.code === 'DUPLICATE_OFFER') {
      duplicateError.value = true
    } else if (e.code === 'BOUQUET_UNAVAILABLE') {
      unavailableError.value = true
      emit('submitted', { bouquet: props.bouquet, unavailable: true })
    } else {
      errorText.value = e.message || 'Ошибка при отправке'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" @close="$emit('close')">
    <StatusState
      v-if="sentState"
      title="Предложение отправлено"
      @cta="$emit('close')"
    >
      Вы предложили <strong>{{ formatPrice(sentPrice) }} ₸</strong>.
      Продавец получит уведомление в Telegram.
    </StatusState>

    <StatusState
      v-else-if="duplicateError"
      title="Предложение уже отправлено"
      @cta="$emit('close')"
    >
      Дождитесь ответа продавца. Если он отклонит или предложит встречную цену —
      сможете снова сделать предложение.
    </StatusState>

    <StatusState
      v-else-if="selfError"
      icon="🛍"
      title="Это ваше объявление"
      @cta="$emit('close')"
    >
      Нельзя сделать предложение на свой собственный букет.
    </StatusState>

    <StatusState
      v-else-if="unavailableError"
      title="Объявление больше не активно"
      @cta="$emit('close')"
    >
      Продавец снял букет с продажи или он уже продан. Каталог обновится.
    </StatusState>

    <template v-else>
      <div class="ctx">
        Продавец просит <strong>{{ formatPrice(sellerPrice) }} ₸</strong>
      </div>

      <div class="price-row">
        <span class="price-display">
          <span class="num">{{ formatPrice(value) }}</span>
          <span class="cur">₸</span>
        </span>
      </div>

      <div class="diff" :class="{ minus: diff < 0, plus: diff > 0 }">
        <template v-if="diff === 0">совпадает с ценой продавца</template>
        <template v-else-if="diff < 0">на {{ formatPrice(Math.abs(diff)) }} ₸ ниже</template>
        <template v-else>на {{ formatPrice(diff) }} ₸ выше</template>
      </div>

      <div class="quick-btns">
        <button type="button" @click="adjust(-1000)">−1 000</button>
        <button type="button" @click="adjust(-500)">−500</button>
        <button type="button" @click="adjust(500)">+500</button>
        <button type="button" @click="adjust(1000)">+1 000</button>
      </div>

      <p v-if="errorText" class="err">{{ errorText }}</p>
      <button class="submit-btn" type="button" :disabled="!canSubmit" @click="submit">
        <span v-if="submitting">Отправляю…</span>
        <span v-else>Предложить {{ formatPrice(value) }} ₸</span>
      </button>
    </template>
  </BaseModal>
</template>

<style scoped>
.ctx {
  text-align: center;
  font-size: 14px;
  color: var(--text-secondary);
  margin: 4px 0 22px;
}
.ctx strong {
  color: var(--text);
  font-weight: 600;
}

/* Цена — только просмотр. Меняется ±-кнопками ниже, ручной ввод выключен. */
.price-row {
  display: flex;
  justify-content: center;
}
.price-display {
  display: inline-flex;
  align-items: baseline;
  gap: 8px;
  font-size: 48px;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
.price-display .cur {
  font-weight: 400;
  color: var(--text-muted);
}

.diff {
  text-align: center;
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 6px;
  margin-bottom: 28px;
  min-height: 18px;
  font-weight: 500;
}
.diff.minus { color: #d6553f; }
.diff.plus { color: #2c8a52; }

.quick-btns {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
  padding-bottom: 20px;
}
.quick-btns button {
  padding: 12px 0;
  border-radius: var(--radius-button);
  background: var(--surface-2);
  color: var(--text);
  border: 0;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.005em;
  transition: transform 0.1s ease-out, background 0.15s ease-out;
}
.quick-btns button:active {
  transform: scale(0.96);
  background: var(--border);
}

.err {
  margin: 0 0 10px;
  color: #d6553f;
  font-size: 13px;
  text-align: center;
}
.submit-btn {
  width: 100%;
  height: 52px;
  border-radius: var(--radius-button);
  background: var(--accent);
  color: var(--accent-text);
  border: 0;
  font-size: 16px;
  font-weight: 700;
  transition: background 0.15s, opacity 0.15s, transform 0.1s;
}
.submit-btn:active:not(:disabled) {
  transform: scale(0.98);
  background: var(--accent-hover);
}
.submit-btn:disabled {
  opacity: 0.5;
}
</style>
