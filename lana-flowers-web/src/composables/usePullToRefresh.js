import { onMounted, onUnmounted, ref } from 'vue'

/**
 * Pull-to-refresh для скроллящегося контейнера.
 *
 * Поддерживает оба источника жеста:
 *  - touch (палец на телефоне) — touchstart/touchmove/touchend
 *  - wheel (трекпад/колесо мыши на десктопе) — wheel-события с deltaY<0
 *    при scrollTop=0 интерпретируются как «pull down». Без этого на маке
 *    в Telegram-десктоп клиенте PTR вообще не работал (нет touch-событий).
 *
 * Использование:
 *   const scrollEl = ref(null)
 *   const { pullDistance, refreshing } = usePullToRefresh(scrollEl, async () => {
 *     await loadData()
 *   })
 */
export function usePullToRefresh(elRef, onRefresh, { threshold = 60, damping = 0.5 } = {}) {
  const pullDistance = ref(0)
  const refreshing = ref(false)
  const dragging = ref(false)

  // ---- Touch (mobile) ----
  let startY = 0
  let pulling = false

  function onTouchStart(e) {
    if (refreshing.value) return
    const el = elRef.value
    if (!el || el.scrollTop > 0) return
    startY = e.touches[0].clientY
    pulling = false
  }

  function onTouchMove(e) {
    if (refreshing.value) return
    const el = elRef.value
    if (!el || el.scrollTop > 0) return
    const dy = e.touches[0].clientY - startY
    if (dy > 0) {
      pulling = true
      dragging.value = true
      pullDistance.value = Math.min(dy * damping, threshold * 1.6)
    }
  }

  async function onTouchEnd() {
    dragging.value = false
    if (!pulling) {
      pullDistance.value = 0
      return
    }
    pulling = false
    await commit()
  }

  // ---- Wheel (desktop: trackpad / mouse wheel) ----
  //
  // Wheel не имеет события «жест закончен», поэтому накапливаем deltaY
  // и через 200мс простоя коммитим (либо refresh, либо reset). 200мс —
  // компромисс между «не дёргать рано если юзер ещё крутит» и «не ждать
  // вечность после того как он отпустил». Трекпадная инерция отрабатывает
  // за ~100-150мс импульсов.
  let wheelAccum = 0
  let wheelTimer = null

  function clearWheelTimer() {
    if (wheelTimer) {
      clearTimeout(wheelTimer)
      wheelTimer = null
    }
  }

  function resetWheel() {
    wheelAccum = 0
    pullDistance.value = 0
    dragging.value = false
    clearWheelTimer()
  }

  function onWheel(e) {
    if (refreshing.value) return
    const el = elRef.value
    if (!el) return

    // Уже проскроллен вниз — не PTR, сбрасываем если накопили.
    if (el.scrollTop > 0) {
      if (wheelAccum > 0) resetWheel()
      return
    }

    // На самом верху. deltaY > 0 — скролл вниз (обычный), не PTR.
    if (e.deltaY >= 0) {
      if (wheelAccum > 0) resetWheel()
      return
    }

    // deltaY < 0 — pull-down жест (трекпад двумя пальцами вниз = scroll up).
    // Накапливаем, рисуем спиннер, ставим таймер на коммит.
    wheelAccum = Math.min(wheelAccum + Math.abs(e.deltaY) * damping, threshold * 1.6)
    pullDistance.value = wheelAccum
    dragging.value = true

    clearWheelTimer()
    wheelTimer = setTimeout(async () => {
      wheelTimer = null
      dragging.value = false
      await commit()
      wheelAccum = 0
    }, 200)
  }

  // ---- Общий коммит после жеста (и тач, и wheel) ----
  async function commit() {
    if (pullDistance.value >= threshold) {
      refreshing.value = true
      pullDistance.value = threshold
      try {
        await onRefresh()
      } catch {}
      refreshing.value = false
      pullDistance.value = 0
    } else {
      pullDistance.value = 0
    }
  }

  onMounted(() => {
    const el = elRef.value
    if (!el) return
    el.addEventListener('touchstart', onTouchStart, { passive: true })
    el.addEventListener('touchmove', onTouchMove, { passive: true })
    el.addEventListener('touchend', onTouchEnd)
    el.addEventListener('touchcancel', onTouchEnd)
    el.addEventListener('wheel', onWheel, { passive: true })
  })

  onUnmounted(() => {
    const el = elRef.value
    if (!el) return
    el.removeEventListener('touchstart', onTouchStart)
    el.removeEventListener('touchmove', onTouchMove)
    el.removeEventListener('touchend', onTouchEnd)
    el.removeEventListener('touchcancel', onTouchEnd)
    el.removeEventListener('wheel', onWheel)
    clearWheelTimer()
  })

  return { pullDistance, refreshing, dragging }
}
