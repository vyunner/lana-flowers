<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  open: { type: Boolean, required: true },
  // offer = { id, price, counterparty: { name }, ... } или null
  offer: { type: Object, default: null },
})
const emit = defineEmits(['close', 'confirm'])

function formatPrice(n) {
  return new Intl.NumberFormat('ru-RU').format(n)
}

const lastPrice = computed(() => Number(props.offer?.price || 0))
const value = ref(0)
const submitting = ref(false)
const errorText = ref('')

watch(
  () => props.open,
  (o) => {
    if (o) {
      value.value = lastPrice.value
      errorText.value = ''
      submitting.value = false
    }
  },
)

function adjust(delta) {
  value.value = Math.max(0, value.value + delta)
}

const diff = computed(() => value.value - lastPrice.value)

const canSubmit = computed(() => value.value > 0 && !submitting.value)

async function confirm() {
  if (!canSubmit.value) return
  submitting.value = true
  errorText.value = ''
  try {
    await emit('confirm', { offerId: props.offer.id, price: value.value })
    // closing — родитель сам управляет open после success
  } catch (e) {
    errorText.value = e.message || 'Ошибка'
    submitting.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <div class="overlay" :class="{ open }" @click="$emit('close')"></div>
    <div class="modal" :class="{ open }" role="dialog" aria-modal="true">
      <div class="ctx">
        Текущая цена: <strong>{{ formatPrice(lastPrice) }} ₸</strong>
      </div>

      <div class="price-row">
        <span class="price-display">
          <span class="num">{{ formatPrice(value) }}</span>
          <span class="cur">₸</span>
        </span>
      </div>

      <div class="diff" :class="{ minus: diff < 0, plus: diff > 0 }">
        <template v-if="diff === 0">совпадает с текущей</template>
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
      <button class="submit" type="button" :disabled="!canSubmit" @click="confirm">
        <span v-if="submitting">Отправляю…</span>
        <span v-else>Предложить {{ formatPrice(value) }} ₸</span>
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

.ctx {
  text-align: center;
  font-size: 14px;
  color: var(--text-secondary);
  margin: 4px 0 22px;
}
.ctx strong { color: var(--text); font-weight: 600; }

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
.submit {
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
.submit:active:not(:disabled) {
  transform: scale(0.98);
  background: var(--accent-hover);
}
.submit:disabled { opacity: 0.5; }
</style>
