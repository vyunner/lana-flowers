import { ref } from 'vue'

/**
 * Реализует свайп-вниз для закрытия bottom-sheet'ов.
 * Возвращает обработчики pointer-событий и стили для применения к шторке.
 *
 * Использование:
 *   const sheet = useSwipeDismiss(() => closeSheet())
 *   <div :style="sheet.dragStyle" :class="{ dragging: sheet.dragging.value }">
 *     <div @pointerdown="sheet.onPointerDown" @pointermove="sheet.onPointerMove"
 *          @pointerup="sheet.onPointerEnd" @pointercancel="sheet.onPointerEnd">handle</div>
 *   </div>
 */
export function useSwipeDismiss(onDismiss, threshold = 0.25) {
  const dragging = ref(false)
  const offset = ref(0)
  let startY = null
  let sheetHeight = 0

  function onPointerDown(e) {
    startY = e.clientY
    sheetHeight = e.currentTarget.closest('[data-sheet]')?.getBoundingClientRect().height || 0
    dragging.value = true
    e.currentTarget.setPointerCapture?.(e.pointerId)
  }
  function onPointerMove(e) {
    if (startY == null) return
    offset.value = Math.max(0, e.clientY - startY)
  }
  function onPointerEnd(e) {
    if (startY == null) return
    const dy = Math.max(0, e.clientY - startY)
    dragging.value = false
    if (sheetHeight && dy > sheetHeight * threshold) {
      onDismiss()
    }
    offset.value = 0
    startY = null
  }

  const dragStyle = () =>
    offset.value > 0 ? { transform: `translateY(${offset.value}px)` } : {}

  return { dragging, offset, dragStyle, onPointerDown, onPointerMove, onPointerEnd }
}
