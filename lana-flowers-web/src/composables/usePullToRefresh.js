import { onMounted, onUnmounted, ref } from 'vue'

/**
 * Pull-to-refresh для скроллящегося контейнера.
 *
 * Использование:
 *   const scrollEl = ref(null)
 *   const { pullDistance, refreshing } = usePullToRefresh(scrollEl, async () => {
 *     await loadData()
 *   })
 *
 *   <div ref="scrollEl" class="scroll-area">
 *     <div class="ptr" :style="{ transform: `translateY(${pullDistance}px)` }">
 *       <span class="spinner" :class="{ spinning: refreshing || pullDistance >= 60 }"></span>
 *     </div>
 *     ... контент ...
 *   </div>
 */
export function usePullToRefresh(elRef, onRefresh, { threshold = 60, damping = 0.5 } = {}) {
  const pullDistance = ref(0)
  const refreshing = ref(false)
  const dragging = ref(false)

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
  })

  onUnmounted(() => {
    const el = elRef.value
    if (!el) return
    el.removeEventListener('touchstart', onTouchStart)
    el.removeEventListener('touchmove', onTouchMove)
    el.removeEventListener('touchend', onTouchEnd)
    el.removeEventListener('touchcancel', onTouchEnd)
  })

  return { pullDistance, refreshing, dragging }
}
