<script setup>
import { computed, ref, watch } from 'vue'
import { createOffer } from '../api/offers'

const props = defineProps({
  open: { type: Boolean, required: true },
  bouquet: { type: Object, default: null },
})
const emit = defineEmits(['close', 'submitted'])

function parsePrice(str) {
  const n = parseInt(String(str).replace(/\D/g, ''), 10)
  return Number.isFinite(n) ? n : 0
}
function formatPrice(n) {
  return new Intl.NumberFormat('ru-RU').format(n)
}

const sellerPrice = computed(() => (props.bouquet ? parsePrice(props.bouquet.price) : 0))
const value = ref(0)
const submitting = ref(false)
const errorText = ref('')
const selfError = ref(false)
const duplicateError = ref(false)
const unavailableError = ref(false) // букет уже снят / продан / удалён
const sentState = ref(false) // success-экран после отправки
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

// Только read-only форматирование — ввод цены идёт исключительно через ±-кнопки.
const display = computed(() => formatPrice(value.value))

function adjust(delta) {
  value.value = Math.max(0, value.value + delta)
}

const canSubmit = computed(() => value.value > 0 && !submitting.value)

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  errorText.value = ''
  try {
    const bouquetId = props.bouquet?.raw?.id ?? props.bouquet?.id
    const result = await createOffer({ bouquetId, price: value.value })
    sentPrice.value = value.value
    sentState.value = true
    // Сигнал родителю — обновить ленту (там появится бейдж «Предложено»).
    // Модалку родитель НЕ закрывает, пользователь сам закрывает после success.
    emit('submitted', { bouquet: props.bouquet, price: value.value, ...result, keepOpen: true })
  } catch (e) {
    if (e.code === 'SELF_OFFER') {
      selfError.value = true
    } else if (e.code === 'DUPLICATE_OFFER') {
      duplicateError.value = true
    } else if (e.code === 'BOUQUET_UNAVAILABLE') {
      unavailableError.value = true
      // Сигнал родителю что список протух — пусть рефрешит каталог.
      // closeOnly=true → не считать это успешным оффером, бейдж не нужен.
      emit('submitted', { bouquet: props.bouquet, unavailable: true })
    } else {
      errorText.value = e.message || 'Ошибка при отправке'
    }
  } finally {
    submitting.value = false
  }
}

const diff = computed(() => value.value - sellerPrice.value)
</script>

<template>
  <Teleport to="body">
    <div class="offer-overlay" :class="{ open }" @click="$emit('close')"></div>
    <div class="offer-modal" :class="{ open }" role="dialog" aria-modal="true">
      <!-- Успех: оффер отправлен -->
      <div v-if="sentState" class="self-state">
        <h3 class="self-title">Предложение отправлено</h3>
        <p class="self-text">
          Вы предложили <strong>{{ formatPrice(sentPrice) }} ₸</strong>. Продавец получит уведомление в Telegram.
        </p>
        <button class="submit-btn" type="button" @click="$emit('close')">
          Понятно
        </button>
      </div>

      <!-- Уже есть активное предложение от этого юзера -->
      <div v-else-if="duplicateError" class="self-state">
        <h3 class="self-title">Предложение уже отправлено</h3>
        <p class="self-text">
          Дождитесь ответа продавца. Если он отклонит или предложит встречную цену — сможете снова сделать предложение.
        </p>
        <button class="submit-btn" type="button" @click="$emit('close')">
          Понятно
        </button>
      </div>

      <!-- Попытка купить свой же букет -->
      <div v-else-if="selfError" class="self-state">
        <div class="self-icon">🛍</div>
        <h3 class="self-title">Это ваше объявление</h3>
        <p class="self-text">
          Нельзя сделать предложение на свой собственный букет.
        </p>
        <button class="submit-btn" type="button" @click="$emit('close')">
          Понятно
        </button>
      </div>

      <!-- Букет уже снят / продан / удалён, у юзера протухшая карточка -->
      <div v-else-if="unavailableError" class="self-state">
        <h3 class="self-title">Объявление больше не активно</h3>
        <p class="self-text">
          Продавец снял букет с продажи или он уже продан. Каталог обновится.
        </p>
        <button class="submit-btn" type="button" @click="$emit('close')">
          Понятно
        </button>
      </div>

      <template v-else>
      <div class="ctx">
        Продавец просит <strong>{{ bouquet ? formatPrice(sellerPrice) : 0 }} ₸</strong>
      </div>

      <div class="price-row">
        <span class="price-display">
          <span class="num">{{ display }}</span>
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
    </div>
  </Teleport>
</template>

<style scoped>
.offer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease-out;
  z-index: 30;
}
.offer-overlay.open {
  opacity: 1;
  pointer-events: auto;
}

.offer-modal {
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
.offer-modal.open {
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
.ctx strong {
  color: var(--text);
  font-weight: 600;
}

/* Цена — только просмотр. Меняется ±-кнопками ниже, ручной ввод выключен,
   чтобы юзер не залипал на клавиатуру в коротком сценарии торга. */
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
  /* Прижато к цене сверху, большой воздух снизу — diff = подпись к цене,
     а не часть блока с кнопками. */
  margin-top: 6px;
  margin-bottom: 28px;
  min-height: 18px;
  font-weight: 500;
}
.diff.minus {
  color: #d6553f;
}
.diff.plus {
  color: #2c8a52;
}

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
  transition:
    transform 0.1s ease-out,
    background 0.15s ease-out;
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
  transition:
    background 0.15s ease-out,
    opacity 0.15s ease-out,
    transform 0.1s ease-out;
}
.submit-btn:active:not(:disabled) {
  transform: scale(0.98);
  background: var(--accent-hover);
}
.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Self-offer состояние */
.self-state {
  text-align: center;
  padding: 8px 0 4px;
}
.self-icon {
  font-size: 48px;
  line-height: 1;
  margin-bottom: 14px;
}
.self-title {
  font-size: 19px;
  font-weight: 700;
  margin: 0 0 8px;
  color: var(--text);
}
.self-text {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 22px;
  line-height: 1.4;
}
</style>
