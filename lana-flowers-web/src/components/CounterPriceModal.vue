<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  open: { type: Boolean, required: true },
  offer: { type: Object, default: null }, // { id, price, bouquet_title }
})
const emit = defineEmits(['close', 'confirm'])

function parsePrice(str) {
  const n = parseInt(String(str).replace(/\D/g, ''), 10)
  return Number.isFinite(n) ? n : 0
}
function formatPrice(n) {
  return new Intl.NumberFormat('ru-RU').format(n)
}

const buyerPrice = computed(() => Number(props.offer?.price || 0))
const value = ref(0)
const submitting = ref(false)
const errorText = ref('')

watch(
  () => props.open,
  (o) => {
    if (o) {
      value.value = buyerPrice.value
      errorText.value = ''
    }
  },
)

const display = computed({
  get: () => formatPrice(value.value),
  set: (str) => (value.value = parsePrice(str)),
})

function adjust(delta) {
  value.value = Math.max(0, value.value + delta)
}

const canSubmit = computed(() => value.value > 0 && !submitting.value)

async function confirm() {
  if (!canSubmit.value) return
  submitting.value = true
  errorText.value = ''
  try {
    await emit('confirm', { offerId: props.offer.id, price: value.value })
    submitting.value = false
  } catch (e) {
    errorText.value = e.message || 'Ошибка'
    submitting.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <div class="overlay" :class="{ open }" @click="$emit('close')"></div>
    <div class="modal" :class="{ open }">
      <div class="title">Встречная цена</div>
      <div class="hint">
        Покупатель предложил <strong>{{ formatPrice(buyerPrice) }} ₸</strong>
      </div>

      <div class="price-row">
        <input
          v-model="display"
          class="price-input"
          type="text"
          inputmode="numeric"
          :style="{ width: Math.max(3, display.length) + 'ch' }"
        />
        <span class="cur">₸</span>
      </div>

      <div class="quick">
        <button type="button" @click="adjust(-1000)">−1 000</button>
        <button type="button" @click="adjust(-500)">−500</button>
        <button type="button" @click="adjust(500)">+500</button>
        <button type="button" @click="adjust(1000)">+1 000</button>
      </div>

      <p v-if="errorText" class="err">{{ errorText }}</p>

      <button class="submit" type="button" :disabled="!canSubmit" @click="confirm">
        {{ submitting ? 'Отправляю…' : 'Отправить встречную цену' }}
      </button>
    </div>
  </Teleport>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease-out;
  z-index: 30;
}
.overlay.open {
  opacity: 1;
  pointer-events: auto;
}
.modal {
  position: fixed;
  top: 50%;
  left: 50%;
  width: calc(100% - 32px);
  max-width: 360px;
  background: var(--surface);
  color: var(--text);
  border-radius: 18px;
  padding: 22px 22px 20px;
  box-shadow: var(--shadow-modal);
  z-index: 31;
  transform: translate(-50%, -50%) scale(0.92);
  opacity: 0;
  pointer-events: none;
  transition:
    transform 0.22s cubic-bezier(0.34, 1.4, 0.64, 1),
    opacity 0.18s ease-out;
}
.modal.open {
  transform: translate(-50%, -50%) scale(1);
  opacity: 1;
  pointer-events: auto;
}

.title {
  text-align: center;
  font-size: 17px;
  font-weight: 700;
  margin-bottom: 6px;
}
.hint {
  text-align: center;
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 18px;
}
.price-row {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 8px;
  margin-bottom: 14px;
}
.price-input {
  font-size: 38px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--text);
  background: transparent;
  border: 0;
  border-bottom: 1px solid transparent;
  text-align: center;
  outline: none;
  padding: 0;
  font-family: inherit;
  transition: border-color 0.15s ease-out;
  min-width: 3ch;
}
.price-input:focus {
  border-bottom-color: var(--text);
}
.cur {
  font-size: 22px;
  font-weight: 600;
  color: var(--text-secondary);
}
.quick {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
  margin-bottom: 16px;
}
.quick button {
  padding: 11px 0;
  border-radius: var(--radius-button);
  background: var(--surface-2);
  color: var(--text);
  border: 0;
  font-size: 13px;
  font-weight: 600;
  transition: transform 0.1s ease-out, background 0.15s ease-out;
}
.quick button:active {
  transform: scale(0.96);
  background: var(--border);
}
.err {
  text-align: center;
  color: #d6553f;
  font-size: 13px;
  margin: 0 0 10px;
}
.submit {
  width: 100%;
  height: 50px;
  border-radius: var(--radius-button);
  background: var(--accent);
  color: var(--accent-text);
  border: 0;
  font-size: 15px;
  font-weight: 700;
  transition: background 0.15s, opacity 0.15s, transform 0.1s;
}
.submit:active:not(:disabled) {
  transform: scale(0.98);
  background: var(--accent-hover);
}
.submit:disabled {
  opacity: 0.5;
}
</style>
