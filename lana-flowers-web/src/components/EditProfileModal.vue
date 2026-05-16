<script setup>
import { computed, ref, watch } from 'vue'
import { updateMe } from '../api/users'
import { uploadPhoto } from '../api/upload'
import { haptic, hapticNotify } from '../telegram'
import { me, setMe } from '../state/auth'
import BaseSheet from './base/BaseSheet.vue'

const props = defineProps({
  open: { type: Boolean, required: true },
})
const emit = defineEmits(['close'])

const name = ref('')
const avatarUrl = ref('')
const uploading = ref(false)
const submitting = ref(false)
const errorText = ref('')
const fileInput = ref(null)

watch(
  () => props.open,
  (o) => {
    if (o) {
      name.value = me.value?.display_name || ''
      avatarUrl.value = me.value?.avatar_url || ''
      errorText.value = ''
    }
  },
)

const canSave = computed(
  () =>
    !submitting.value &&
    !uploading.value &&
    name.value.trim().length > 0 &&
    (name.value.trim() !== (me.value?.display_name || '') ||
      avatarUrl.value !== (me.value?.avatar_url || '')),
)

function pickPhoto() {
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
  } catch (err) {
    errorText.value = err.message || 'Не удалось загрузить'
  } finally {
    uploading.value = false
  }
}

function removePhoto() {
  haptic('medium')
  avatarUrl.value = ''
}

async function save() {
  if (!canSave.value) return
  submitting.value = true
  errorText.value = ''
  try {
    const updated = await updateMe({
      display_name: name.value.trim(),
      avatar_url: avatarUrl.value,
    })
    setMe(updated)
    hapticNotify('success')
    emit('close')
  } catch (e) {
    errorText.value = e.message || 'Не удалось сохранить'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseSheet :open="open" full-height @close="$emit('close')">
    <h2 class="title">Редактировать профиль</h2>

    <div class="avatar-section">
      <button class="picker" type="button" @click="pickPhoto" :disabled="uploading || submitting">
        <img v-if="avatarUrl" :src="avatarUrl" class="preview" alt="" />
        <div v-else-if="uploading" class="placeholder">
          <span class="spinner"></span>
        </div>
        <div v-else class="placeholder">
          <svg viewBox="0 0 32 32" fill="none">
            <circle cx="16" cy="13" r="5" stroke="currentColor" stroke-width="1.8" />
            <path d="M6 26c1.8-4.4 5.7-7 10-7s8.2 2.6 10 7" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
          </svg>
        </div>
      </button>
      <div class="avatar-actions">
        <button class="link" type="button" @click="pickPhoto" :disabled="uploading">
          {{ avatarUrl ? 'Заменить фото' : 'Загрузить фото' }}
        </button>
        <button v-if="avatarUrl" class="link danger" type="button" @click="removePhoto">
          Удалить
        </button>
      </div>
    </div>

    <input
      ref="fileInput"
      type="file"
      accept="image/jpeg,image/png,image/webp"
      style="display:none"
      @change="onFile"
    />

    <label class="field">
      <span class="field-label">Имя</span>
      <input v-model="name" class="field-input" type="text" maxlength="64" placeholder="Имя" />
    </label>

    <template #footer>
      <p v-if="errorText" class="err">{{ errorText }}</p>
      <button class="save-btn" type="button" :disabled="!canSave" @click="save">
        {{ submitting ? 'Сохраняю…' : 'Сохранить' }}
      </button>
    </template>
  </BaseSheet>
</template>

<style scoped>
.title {
  margin: 0 0 16px;
  font-size: 17px;
  font-weight: 700;
  text-align: center;
}

.avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 8px 0 4px;
}
.picker {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  border: 0;
  background: var(--surface-2);
  overflow: hidden;
  cursor: pointer;
  padding: 0;
  transition: transform 0.1s ease-out;
}
.picker:active:not(:disabled) { transform: scale(0.97); }
.preview { width: 100%; height: 100%; object-fit: cover; display: block; }
.placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
}
.placeholder svg { width: 36px; height: 36px; }
.spinner {
  width: 22px;
  height: 22px;
  border: 2.5px solid var(--border);
  border-top-color: var(--text);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.avatar-actions {
  display: flex;
  gap: 14px;
  align-items: center;
}
.link {
  background: transparent;
  border: 0;
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
  text-decoration: underline;
}
.link.danger { color: #d6553f; }
.link:disabled { opacity: 0.5; }

.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.field-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
}
.field-input {
  width: 100%;
  height: 48px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-button);
  padding: 0 16px;
  font-size: 16px;
  color: var(--text);
  outline: none;
  transition: border-color 0.15s;
}
.field-input:focus { border-color: var(--text); }

/* Поля идут друг под другом с воздухом */
.field { margin-top: 20px; }

.err {
  margin: 0 0 10px;
  font-size: 13px;
  color: #d6553f;
  text-align: center;
}
.save-btn {
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
.save-btn:active:not(:disabled) { transform: scale(0.98); }
.save-btn:disabled { opacity: 0.4; }
</style>
