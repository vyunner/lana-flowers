<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useSwipeDismiss } from '../composables/useSwipeDismiss'
import { createBouquet } from '../api/bouquets'
import { uploadPhoto } from '../api/upload'

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

const swipe = useSwipeDismiss(() => {
  if (!submitting.value) emit('close')
})

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
      }, 400)
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
      const url = await uploadPhoto(file)
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
  <div class="sell-overlay" :class="{ open }" @click="$emit('close')"></div>
  <div
    class="sell-sheet"
    :class="{ open, dragging: swipe.dragging.value }"
    :style="swipe.dragStyle()"
    data-sheet
  >
    <div
      class="sheet-handle-area"
      @pointerdown="swipe.onPointerDown"
      @pointermove="swipe.onPointerMove"
      @pointerup="swipe.onPointerEnd"
      @pointercancel="swipe.onPointerEnd"
    >
      <span class="sheet-handle"></span>
    </div>

    <div class="sell-header">
      <button class="back" type="button" aria-label="Закрыть" @click="$emit('close')">
        <svg viewBox="0 0 24 24" fill="none">
          <path
            d="M15 6l-6 6 6 6"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
      <h2>Новое объявление</h2>
    </div>

    <div class="sell-scroll">
      <div class="photo-strip">
        <button
          v-if="photos.length < 5"
          class="photo-add"
          type="button"
          @click="pickFiles"
        >
          <svg viewBox="0 0 24 24" fill="none">
            <path
              d="M12 5v14M5 12h14"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
            />
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
          <button class="photo-remove" type="button" @click="removePhoto(i)">×</button>
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
            @click="
              () => {
                cityFocused = true
                $emit('open-city')
              }
            "
          >
            <span>{{ city }}</span>
            <svg viewBox="0 0 16 16" fill="none">
              <path
                d="M4 6l4 4 4-4"
                stroke="currentColor"
                stroke-width="1.6"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
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
    </div>

    <div class="sell-publish">
      <p v-if="errorText" class="err">{{ errorText }}</p>
      <button class="publish-btn" type="button" :disabled="!canPublish" @click="publish">
        {{ submitting ? 'Публикую…' : 'Опубликовать' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.sell-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.25s ease-out;
  z-index: 12;
}
.sell-overlay.open {
  opacity: 1;
  pointer-events: auto;
}

.sell-sheet {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 95%;
  background: var(--surface);
  color: var(--text);
  border-radius: var(--radius-sheet) var(--radius-sheet) 0 0;
  transform: translateY(100%);
  transition: transform 0.4s ease-out;
  z-index: 13;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-sheet);
  will-change: transform;
  overflow: hidden;
}
.sell-sheet.open {
  transform: translateY(0);
}
.sell-sheet.dragging {
  transition: none;
}

.sheet-handle-area {
  padding: 10px 0 4px;
  display: flex;
  justify-content: center;
  cursor: grab;
  touch-action: none;
  user-select: none;
}
.sheet-handle {
  width: 40px;
  height: 4px;
  background: #d4d1c8;
  border-radius: var(--radius-pill);
}

.sell-header {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2px 16px 14px;
  position: relative;
  flex-shrink: 0;
}
.sell-header h2 {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.005em;
  color: var(--text);
}
.sell-header .back {
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 0;
  border-radius: 50%;
  color: var(--text);
  transition: background 0.15s;
}
.sell-header .back:active {
  background: rgba(0, 0, 0, 0.06);
}
.sell-header .back svg {
  width: 22px;
  height: 22px;
}

.sell-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 4px 20px 24px;
}

/* Photos */
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
.photo-add:active {
  transform: scale(0.95);
}
.photo-add svg {
  width: 24px;
  height: 24px;
}
.photo-add .label {
  font-size: 11px;
  font-weight: 500;
}

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
.upload-overlay.err {
  background: rgba(214, 85, 63, 0.85);
}
.mini-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.4);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
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
  margin-top: 10px;
  margin-bottom: 24px;
  line-height: 1.4;
}

/* Form */
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
.field-textarea::placeholder {
  color: var(--text-muted);
}
.field-input:focus,
.field-textarea:focus,
.field-select.focused {
  border-color: var(--text);
}
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

.sell-publish {
  padding: 14px 20px 22px;
  background: var(--surface);
  border-top: 1px solid var(--border);
  flex-shrink: 0;
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
.publish-btn:active:not(:disabled) {
  transform: scale(0.98);
}
.publish-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
