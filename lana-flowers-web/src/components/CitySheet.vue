<script setup>
import { useSwipeDismiss } from '../composables/useSwipeDismiss'
import { cities } from '../data/cities'

const props = defineProps({
  open: { type: Boolean, required: true },
  current: { type: String, required: true },
})
const emit = defineEmits(['select', 'close'])

const swipe = useSwipeDismiss(() => emit('close'))
</script>

<template>
  <div class="city-overlay" :class="{ open }" @click="$emit('close')"></div>
  <div
    class="city-sheet"
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
    <div class="sheet-title">Выберите город</div>
    <ul class="city-list">
      <li
        v-for="c in cities"
        :key="c"
        :class="{ selected: c === current }"
        @click="$emit('select', c)"
      >
        <span>{{ c }}</span>
        <svg class="check" viewBox="0 0 20 20" fill="none">
          <path
            d="M4 10.5l3.8 3.8L16 6"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.city-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.25s ease-out;
  z-index: 20;
}
.city-overlay.open {
  opacity: 1;
  pointer-events: auto;
}

.city-sheet {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 70%;
  background: var(--surface);
  border-radius: var(--radius-sheet) var(--radius-sheet) 0 0;
  transform: translateY(100%);
  transition: transform 0.35s ease-out;
  z-index: 21;
  display: flex;
  flex-direction: column;
  color: var(--text);
  box-shadow: var(--shadow-sheet);
  will-change: transform;
}
.city-sheet.open {
  transform: translateY(0);
}
.city-sheet.dragging {
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
.sheet-handle-area:active {
  cursor: grabbing;
}
.sheet-handle {
  width: 40px;
  height: 4px;
  background: #d4d1c8;
  border-radius: var(--radius-pill);
}

.sheet-title {
  text-align: center;
  font-size: 17px;
  font-weight: 700;
  padding: 6px 20px 14px;
  letter-spacing: -0.005em;
}

.city-list {
  list-style: none;
  margin: 0;
  padding: 0 8px 24px;
  flex: 1;
  overflow-y: auto;
}
.city-list li {
  padding: 14px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-radius: 12px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 500;
  transition: background 0.15s ease-out;
}
.city-list li:active {
  background: rgba(0, 0, 0, 0.06);
}
.city-list li .check {
  width: 18px;
  height: 18px;
  opacity: 0;
  transition: opacity 0.15s ease-out;
}
.city-list li.selected .check {
  opacity: 1;
}
</style>
