<script setup>
import { computed, ref } from 'vue'
import { updateMe } from '../api/users'
import { haptic, hapticNotify } from '../telegram'
import { setMe } from '../state/auth'
import { cities } from '../data/cities'

// Третий шаг онбординга (имя → аватар → ГОРОД → телефон). Юзер выбирает
// родной город из whitelist'а (тот же что в utils/cities.js + backend
// allowedCities). Каталог потом по умолчанию открывается в этом городе.

const selected = ref('')
const submitting = ref(false)
const errorText = ref('')

const canSubmit = computed(() => !!selected.value && !submitting.value)

function pick(c) {
  haptic('light')
  selected.value = c
}

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  errorText.value = ''
  try {
    const updated = await updateMe({ city: selected.value })
    setMe(updated)
    hapticNotify('success')
  } catch (e) {
    errorText.value = e.message || 'Не удалось сохранить'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="step">
    <div class="card">
      <h1 class="title">Ваш город?</h1>
      <p class="lead">
        В каталоге сразу будем показывать букеты вашего города. В любой момент
        сможете переключить.
      </p>

      <div class="grid">
        <button
          v-for="c in cities"
          :key="c"
          type="button"
          class="chip"
          :class="{ active: selected === c }"
          @click="pick(c)"
        >
          {{ c }}
        </button>
      </div>

      <p v-if="errorText" class="err">{{ errorText }}</p>

      <button class="primary-btn" type="button" :disabled="!canSubmit" @click="submit">
        {{ submitting ? 'Сохраняю…' : 'Далее' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.step {
  position: absolute;
  inset: 0;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 24px calc(120px + env(safe-area-inset-bottom, 0px));
  z-index: 100;
}
.card {
  width: 100%;
  max-width: 380px;
  text-align: center;
}
.title {
  font-size: 26px;
  font-weight: 700;
  margin: 0 0 12px;
  letter-spacing: -0.02em;
}
.lead {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 24px;
  line-height: 1.4;
}

.grid {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-bottom: 20px;
}
.chip {
  padding: 9px 14px;
  border-radius: var(--radius-pill);
  border: 1px solid var(--secondary-border);
  background: transparent;
  color: var(--secondary-text);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s, transform 0.1s;
}
.chip:active:not(.active) {
  transform: scale(0.97);
  background: var(--surface-2);
}
.chip.active {
  background: var(--primary);
  color: var(--primary-text);
  border-color: var(--primary);
  font-weight: 600;
}

.err {
  color: #d6553f;
  font-size: 13px;
  margin: 0 0 12px;
}

.primary-btn {
  width: 100%;
  height: 50px;
  background: var(--primary);
  color: var(--primary-text);
  border: 0;
  border-radius: var(--radius-button);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.005em;
  transition: opacity 0.15s, transform 0.1s ease-out;
}
.primary-btn:disabled {
  opacity: 0.4;
}
.primary-btn:active:not(:disabled) {
  transform: scale(0.98);
}
</style>
