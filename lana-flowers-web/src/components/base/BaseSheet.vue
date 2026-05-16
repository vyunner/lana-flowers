<script setup>
import { onUnmounted, ref, useSlots, watch } from 'vue'
import { tg } from '../../telegram'

// BaseSheet — bottom sheet (выезжает снизу). Использовать для контента где
// важна высота: списки (CitySheet), длинные формы (SellSheet), детали
// контакта (ContactSheet).
//
// Слоты:
//   default — основное содержимое (скроллится если #footer задан)
//   footer — опциональный pinned-блок снизу (CTA-кнопка, сообщение об ошибке).
//            Если задан, .body становится скролл-областью, footer прижат.
//
// Особенности:
// - Teleport в body (sheet всегда поверх всего)
// - Telegram BackButton + close-on-overlay-tap
// - swipe-down-to-dismiss (если swipeable=true) — больше четверти sheet
//   утащил вниз → close, иначе snap back
const props = defineProps({
  open: { type: Boolean, required: true },
  closeOnOverlay: { type: Boolean, default: true },
  swipeable: { type: Boolean, default: true },
  // fullHeight=true → sheet занимает ~95vh, для длинных форм типа SellSheet.
  // По дефолту — auto height, контент диктует размер до max-height: 90vh.
  fullHeight: { type: Boolean, default: false },
  // level — z-index слой. 1 = базовый sheet (overlay=30, sheet=31).
  // 2 = sheet поверх другого sheet (CitySheet открывается из SellSheet),
  // и т.д. без этого пропа sheet с одинаковым z-index перекрывались
  // в порядке монтирования в DOM, что было непредсказуемо.
  level: { type: Number, default: 1 },
})
const emit = defineEmits(['close'])

const slots = useSlots()
const hasFooter = !!slots.footer

function close() {
  emit('close')
}

// ---- swipe-down ----
const dragY = ref(0)
const dragging = ref(false)
let startY = 0
let sheetH = 0
let sheetEl = null

function onTouchStart(e) {
  if (!props.swipeable) return
  // Свайп ловим только если палец стартовал на handle/верхнем 80px sheet'а —
  // иначе скролл внутри body перехватывал бы каждое движение пальцем.
  // Простой способ: вешаем хендлеры на handle-area отдельно (см. template).
  startY = e.touches[0].clientY
  dragging.value = true
  sheetEl = e.currentTarget.closest('.sheet')
  sheetH = sheetEl?.offsetHeight || 0
}
function onTouchMove(e) {
  if (!props.swipeable || !dragging.value) return
  const dy = e.touches[0].clientY - startY
  if (dy > 0) dragY.value = dy
}
function onTouchEnd() {
  if (!props.swipeable) return
  dragging.value = false
  if (sheetH > 0 && dragY.value > sheetH * 0.25) {
    close()
  }
  requestAnimationFrame(() => {
    dragY.value = 0
  })
}

// Сброс drag-стейта при close снаружи
watch(
  () => props.open,
  (o) => {
    if (!o) {
      dragY.value = 0
      dragging.value = false
    }
  },
)

// ---- Telegram BackButton ----
watch(
  () => props.open,
  (o) => {
    if (!tg?.BackButton) return
    if (o) {
      tg.BackButton.onClick(close)
      tg.BackButton.show()
    } else {
      tg.BackButton.hide()
      tg.BackButton.offClick(close)
    }
  },
)

onUnmounted(() => {
  if (tg?.BackButton) {
    tg.BackButton.hide()
    tg.BackButton.offClick(close)
  }
})
</script>

<template>
  <Teleport to="body">
    <div
      class="overlay"
      :class="{ open }"
      :style="{ zIndex: 28 + level * 2 }"
      @click="closeOnOverlay && close()"
    ></div>
    <div
      class="sheet"
      :class="{ open, 'full-height': fullHeight, 'has-footer': hasFooter }"
      :style="{
        transform: open ? `translateY(${dragY}px)` : 'translateY(100%)',
        transition: dragging ? 'none' : 'transform .25s cubic-bezier(.2,.8,.2,1)',
        zIndex: 29 + level * 2,
      }"
      role="dialog"
      aria-modal="true"
    >
      <!-- Swipe-зона — только handle. Не вешаем touch на весь sheet, чтобы
           вертикальный скролл body работал без конфликтов. -->
      <div
        class="handle-area"
        v-if="swipeable"
        @touchstart.passive="onTouchStart"
        @touchmove.passive="onTouchMove"
        @touchend="onTouchEnd"
        @touchcancel="onTouchEnd"
      >
        <div class="handle"></div>
      </div>

      <div class="body">
        <slot></slot>
      </div>

      <div v-if="hasFooter" class="footer">
        <slot name="footer"></slot>
      </div>
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

.sheet {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  max-height: 90vh;
  background: var(--surface);
  color: var(--text);
  border-radius: 18px 18px 0 0;
  box-shadow: var(--shadow-sheet);
  z-index: 31;
  display: flex;
  flex-direction: column;
}
.sheet.full-height {
  height: 95vh;
  max-height: 95vh;
}

.handle-area {
  padding: 8px 0 6px;
  display: flex;
  justify-content: center;
  flex-shrink: 0;
  cursor: grab;
  touch-action: none;
  user-select: none;
}
.handle {
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--border);
}

.body {
  flex: 1;
  overflow-y: auto;
  overscroll-behavior-y: contain;
  padding: 0 20px calc(20px + env(safe-area-inset-bottom, 0px));
}
/* Если есть footer — body НЕ паддит снизу (footer прижат, имеет свой паддинг) */
.sheet.has-footer .body {
  padding-bottom: 12px;
}

.footer {
  flex-shrink: 0;
  padding: 14px 20px calc(20px + env(safe-area-inset-bottom, 0px));
  border-top: 1px solid var(--border);
  background: var(--surface);
}
</style>
