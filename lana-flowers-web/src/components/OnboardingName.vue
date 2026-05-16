<script setup>
import { computed, onMounted, ref } from 'vue'
import { updateMe } from '../api/users'
import { haptic, hapticNotify } from '../telegram'
import { me, setMe } from '../state/auth'

const name = ref(me.value?.first_name || '')
const submitting = ref(false)
const errorText = ref('')
const inputRef = ref(null)

onMounted(() => {
  setTimeout(() => inputRef.value?.focus(), 100)
})

const canSubmit = computed(() => name.value.trim().length > 0 && !submitting.value)

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  errorText.value = ''
  try {
    haptic('light')
    const updated = await updateMe({ display_name: name.value.trim() })
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
      <h1 class="title">Как вас зовут?</h1>
      <p class="lead">Это имя увидят продавцы и покупатели в каталоге и сообщениях.</p>

      <input
        ref="inputRef"
        v-model="name"
        class="input"
        type="text"
        placeholder="Например, Айгерим"
        maxlength="64"
        @keyup.enter="submit"
      />

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
  max-width: 340px;
  text-align: center;
}
.title {
  font-size: 26px;
  font-weight: 700;
  margin: 0 0 12px;
  letter-spacing: -0.02em;
  line-height: 1.2;
}
.lead {
  margin: 0 0 28px;
  font-size: 15px;
  color: var(--text-secondary);
  line-height: 1.45;
}
.input {
  width: 100%;
  height: 52px;
  border-radius: var(--radius-button);
  border: 1px solid var(--border);
  background: var(--surface-2);
  color: var(--text);
  padding: 0 16px;
  font-size: 16px;
  outline: none;
  margin-bottom: 14px;
  text-align: center;
  transition: border-color 0.15s;
}
.input:focus {
  border-color: var(--text);
}
.err {
  margin: 0 0 12px;
  font-size: 13px;
  color: #d6553f;
}
.primary-btn {
  width: 100%;
  height: 52px;
  border: 0;
  border-radius: var(--radius-button);
  background: var(--primary);
  color: var(--primary-text);
  font-size: 16px;
  font-weight: 700;
  transition: transform 0.1s, opacity 0.15s;
}
.primary-btn:active:not(:disabled) {
  transform: scale(0.98);
}
.primary-btn:disabled {
  opacity: 0.4;
}
</style>
