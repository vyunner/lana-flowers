<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { createBouquet } from '../api/bouquets'
import { uploadPhoto } from '../api/upload'
import { resizeImage } from '../utils/image'
import BaseSheet from './base/BaseSheet.vue'

// resetDelay должен совпадать с длительностью BaseSheet'овской slide-down
// анимации (0.25s), чтобы поля не «гасли» прямо у юзера на глазах.
const RESET_DELAY_MS = 300

const props = defineProps({
  open: { type: Boolean, required: true },
  city: { type: String, required: true },
})
const emit = defineEmits(['close', 'open-city', 'published'])

const title = ref('')
const price = ref('')
const desc = ref('')
const cityFocused = ref(false)
const submitting = ref(false)
const errorText = ref('')

const photos = ref([]) // [{ url, uploading: bool, error: string }]
const fileInput = ref(null)

const CATEGORIES = [
  { key: 'roses', label: 'Розы' },
  { key: 'peonies', label: 'Пионы' },
  { key: 'wild', label: 'Полевые' },
  { key: 'composition', label: 'Композиции' },
  { key: 'dried', label: 'Сухоцветы' },
  { key: 'all', label: 'Другое' },
]
const category = ref('roses')

const canPublish = computed(
  () =>
    title.value.trim() &&
    String(price.value).trim() &&
    !submitting.value &&
    !photos.value.some((p) => p.uploading),
)

watch(
  () => props.open,
  (o) => {
    if (!o) {
      cityFocused.value = false
      errorText.value = ''
      setTimeout(() => {
        title.value = ''
        price.value = ''
        desc.value = ''
        photos.value = []
        category.value = 'roses'
      }, RESET_DELAY_MS)
    }
  },
)

function pickFiles() {
  fileInput.value?.click()
}

async function onFiles(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = '' // позволяем выбрать тот же файл повторно
  for (const file of files) {
    if (photos.value.length >= 5) break
    // reactive() — иначе мутации item.uploading=false не триггерят rerender
    // последнего пуша (Vue track'ает только через прокси).
    const item = reactive({
      url: '',
      uploading: true,
      error: '',
      preview: URL.createObjectURL(file),
    })
    photos.value.push(item)
    try {
      // Сжимаем перед upload'ом — айфон даёт 10MB-фото, бэк лимит 5MB.
      // resizeImage возвращает оригинал если он уже маленький.
      const resized = await resizeImage(file)
      const url = await uploadPhoto(resized)
      item.url = url
      item.uploading = false
    } catch (err) {
      item.uploading = false
      item.error = err.message || 'upload failed'
    }
  }
}

function removePhoto(idx) {
  photos.value.splice(idx, 1)
}

async function publish() {
  if (!canPublish.value) return
  // ВАЖНО: submitting=true СНАЧАЛА, до любых async/await — иначе
  // двойной тап на «Опубликовать» проскакивает оба раза и создаёт
  // два дубля. canPublish.value завязан на submitting → второй тап
  // мгновенно дисэйблится.
  submitting.value = true
  errorText.value = ''
  try {
    const result = await createBouquet({
      title: title.value.trim(),
      description: desc.value.trim(),
      price: Number(price.value),
      city: props.city,
      photos: photos.value.filter((p) => p.url).map((p) => p.url),
      category: category.value,
    })
    emit('published', result)
  } catch (e) {
    errorText.value = e.message || 'Не удалось опубликовать'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseSheet :open="open" full-height @close="$emit('close')">
    <h2 class="title">Новое объявление</h2>

    <div class="photo-strip">
      <button
        v-if="photos.length < 5"
        class="photo-add"
        type="button"
        @click="pickFiles"
      >
        <svg viewBox="0 0 24 24" fill="none">
          <path d="M12 5v14M5 12h14" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
        </svg>
        <span class="label">Фото</span>
      </button>

      <div
        v-for="(p, i) in photos"
        :key="i"
        class="photo-item"
        :style="{ backgroundImage: `url(${p.url || p.preview})` }"
      >
        <div v-if="p.uploading" class="upload-overlay">
          <span class="mini-spinner"></span>
        </div>
        <div v-else-if="p.error" class="upload-overlay err">!</div>
        <button class="photo-remove" type="button" @click="removePhoto(i)" aria-label="Удалить">×</button>
        <span v-if="i === 0" class="main-tag">главное</span>
      </div>
    </div>
    <input
      ref="fileInput"
      type="file"
      accept="image/jpeg,image/png,image/webp"
      multiple
      style="display: none"
      @change="onFiles"
    />
    <p class="photo-caption">До 5 фото. Первое — главное.</p>

    <div class="form-fields">
      <div class="field">
        <span class="field-label">Город</span>
        <button
          class="field-select"
          type="button"
          :class="{ focused: cityFocused }"
          @click="cityFocused = true; $emit('open-city')"
        >
          <span>{{ city }}</span>
          <svg viewBox="0 0 16 16" fill="none">
            <path d="M4 6l4 4 4-4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </button>
      </div>

      <label class="field">
        <span class="field-label">Название</span>
        <input
          v-model="title"
          class="field-input"
          type="text"
          placeholder="Букет роз 25 шт"
          @keyup.enter="$event.target.blur()"
        />
      </label>

      <label class="field">
        <span class="field-label">Описание</span>
        <textarea
          v-model="desc"
          class="field-textarea"
          rows="4"
          placeholder="Расскажите о букете: состав, сколько дней прошло с покупки, особенности"
        />
      </label>

      <div class="field">
        <span class="field-label">Категория</span>
        <div class="cat-chips">
          <button
            v-for="c in CATEGORIES"
            :key="c.key"
            type="button"
            class="cat-chip"
            :class="{ active: category === c.key }"
            @click="category = c.key"
          >
            {{ c.label }}
          </button>
        </div>
      </div>

      <label class="field">
        <span class="field-label">Цена, ₸</span>
        <input
          v-model="price"
          class="field-input"
          type="number"
          inputmode="numeric"
          placeholder="0"
          @keyup.enter="$event.target.blur()"
        />
      </label>
    </div>

    <template #footer>
      <p v-if="errorText" class="err">{{ errorText }}</p>
      <button class="publish-btn" type="button" :disabled="!canPublish" @click="publish">
        {{ submitting ? 'Публикую…' : 'Опубликовать' }}
      </button>
    </template>
  </BaseSheet>
</template>

<style scoped>
.title {
  margin: 0 0 12px;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.005em;
  text-align: center;
}

.photo-strip {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 2px 0 8px;
}
.photo-add {
  flex-shrink: 0;
  width: 80px;
  height: 80px;
  border: 1.5px dashed var(--text-muted);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  background: transparent;
  color: var(--text-secondary);
  transition: transform 0.1s ease-out;
}
.photo-add:active { transform: scale(0.95); }
.photo-add svg { width: 24px; height: 24px; }
.photo-add .label { font-size: 11px; font-weight: 500; }

.photo-item {
  position: relative;
  flex-shrink: 0;
  width: 80px;
  height: 80px;
  border-radius: 10px;
  background-size: cover;
  background-position: center;
  background-color: var(--surface-2);
  overflow: hidden;
}
.upload-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 700;
}
.upload-overlay.err { background: rgba(214, 85, 63, 0.85); }
.mini-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.4);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.photo-remove {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.65);
  color: white;
  border: 0;
  font-size: 16px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}
.main-tag {
  position: absolute;
  bottom: 4px;
  left: 4px;
  font-size: 9px;
  font-weight: 600;
  text-transform: uppercase;
  background: rgba(0, 0, 0, 0.65);
  color: white;
  padding: 2px 6px;
  border-radius: 4px;
}

.photo-caption {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 10px 0 24px;
  line-height: 1.4;
}

.form-fields {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.field-label {
  color: var(--text);
  font-size: 14px;
  font-weight: 500;
  letter-spacing: -0.005em;
}
.field-input,
.field-textarea,
.field-select {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-button);
  padding: 12px 16px;
  color: var(--text);
  font-size: 15px;
  outline: none;
  transition: border-color 0.15s ease-out;
  width: 100%;
  text-align: left;
}
.field-input::placeholder,
.field-textarea::placeholder { color: var(--text-muted); }
.field-input:focus,
.field-textarea:focus,
.field-select.focused { border-color: var(--text); }
.field-textarea {
  min-height: 96px;
  resize: none;
  font-family: inherit;
  line-height: 1.4;
}
.field-select {
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  font-weight: 500;
  user-select: none;
}
.field-select svg {
  width: 16px;
  height: 16px;
  opacity: 0.55;
  flex-shrink: 0;
}

.cat-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.cat-chip {
  background: transparent;
  border: 1px solid var(--secondary-border);
  color: var(--secondary-text);
  border-radius: var(--radius-pill);
  padding: 7px 12px;
  font-size: 12.5px;
  font-weight: 500;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}
.cat-chip.active {
  background: var(--primary);
  color: var(--primary-text);
  border-color: var(--primary);
}

.err {
  margin: 0 0 10px;
  color: #d6553f;
  font-size: 13px;
  text-align: center;
}
.publish-btn {
  width: 100%;
  height: 56px;
  border-radius: var(--radius-button);
  background: var(--primary);
  color: var(--primary-text);
  border: 0;
  font-size: 16px;
  font-weight: 700;
  transition: opacity 0.15s ease-out, transform 0.1s ease-out;
}
.publish-btn:active:not(:disabled) { transform: scale(0.98); }
.publish-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
