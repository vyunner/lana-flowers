<script setup>
import { ref } from 'vue'
import { updateMe } from '../api/users'
import { uploadPhoto } from '../api/upload'
import { haptic, hapticNotify } from '../telegram'
import { me, setMe } from '../state/auth'

const avatarUrl = ref('')
const uploading = ref(false)
const submitting = ref(false)
const errorText = ref('')
const fileInput = ref(null)

function pick() {
  haptic('light')
  fileInput.value?.click()
}

async function onFile(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  uploading.value = true
  errorText.value = ''
  try {
    avatarUrl.value = await uploadPhoto(file)
    hapticNotify('success')
  } catch (err) {
    errorText.value = err.message || 'Не удалось загрузить'
  } finally {
    uploading.value = false
  }
}

async function finish(skip = false) {
  submitting.value = true
  errorText.value = ''
  try {
    const patch = { onboarding_completed: true }
    if (!skip && avatarUrl.value) patch.avatar_url = avatarUrl.value
    const updated = await updateMe(patch)
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
      <h1 class="title">Аватарка</h1>
      <p class="lead">Можно загрузить сейчас или пропустить — поменяете в профиле.</p>

      <button class="picker" type="button" @click="pick" :disabled="uploading || submitting">
        <img v-if="avatarUrl" :src="avatarUrl" class="preview" alt="" />
        <div v-else-if="uploading" class="placeholder">
          <span class="spinner"></span>
        </div>
        <div v-else class="placeholder">
          <svg viewBox="0 0 32 32" fill="none">
            <circle cx="16" cy="13" r="5" stroke="currentColor" stroke-width="1.8" />
            <path
              d="M6 26c1.8-4.4 5.7-7 10-7s8.2 2.6 10 7"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
            />
          </svg>
          <span class="add">Загрузить фото</span>
        </div>
      </button>

      <input
        ref="fileInput"
        type="file"
        accept="image/jpeg,image/png,image/webp"
        style="display:none"
        @change="onFile"
      />

      <p v-if="errorText" class="err">{{ errorText }}</p>

      <button
        class="primary-btn"
        type="button"
        :disabled="!avatarUrl || submitting || uploading"
        @click="finish(false)"
      >
        {{ submitting ? 'Сохраняю…' : 'Готово' }}
      </button>
      <button
        class="skip-btn"
        type="button"
        :disabled="submitting"
        @click="finish(true)"
      >
        Пропустить
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
  padding: 24px 24px calc(22vh + env(safe-area-inset-bottom, 0px));
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
}
.lead {
  margin: 0 0 24px;
  font-size: 15px;
  color: var(--text-secondary);
  line-height: 1.45;
}

.picker {
  width: 140px;
  height: 140px;
  border-radius: 50%;
  border: 0;
  background: var(--surface-2);
  margin: 0 auto 24px;
  display: block;
  overflow: hidden;
  cursor: pointer;
  padding: 0;
  position: relative;
  transition: transform 0.1s ease-out;
}
.picker:active:not(:disabled) {
  transform: scale(0.97);
}
.picker:disabled {
  opacity: 0.7;
}
.preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-secondary);
}
.placeholder svg {
  width: 36px;
  height: 36px;
}
.placeholder .add {
  font-size: 12px;
  font-weight: 600;
}
.spinner {
  width: 24px;
  height: 24px;
  border: 2.5px solid var(--border);
  border-top-color: var(--text);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

.err {
  margin: 0 0 10px;
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
  margin-bottom: 8px;
}
.primary-btn:active:not(:disabled) {
  transform: scale(0.98);
}
.primary-btn:disabled {
  opacity: 0.4;
}
.skip-btn {
  width: 100%;
  height: 44px;
  border: 0;
  background: transparent;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
}
.skip-btn:disabled {
  opacity: 0.4;
}
</style>
