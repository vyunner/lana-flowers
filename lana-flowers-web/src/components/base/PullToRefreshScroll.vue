<script setup>
import { ref } from 'vue'
import { usePullToRefresh } from '../../composables/usePullToRefresh'

// PullToRefreshScroll — обёртка над вертикальным скролл-контейнером с
// pull-to-refresh. Использовать вместо дублирующего PTR-кода в каждом
// списке (раньше: BouquetGrid, Profile, Deals — три копии .ptr + .inner +
// translateY-привязки).
//
// API:
//   <PullToRefreshScroll :loader="loadData" @scroll="onScroll" ref="ptrRef">
//     ...твоя сетка/список...
//   </PullToRefreshScroll>
//
//   ptrRef.value.scrollEl  ← реальный scroll-DOM-узел
//
// Почему loader-проп, а не @refresh-event: emit возвращает undefined и
// не ждёт parent'а — спиннер исчезал бы мгновенно, юзер думал «не работает».
// С функцией мы реально ждём промис и держим спиннер видимым.
const props = defineProps({
  loader: { type: Function, required: true },
})
const emit = defineEmits(['scroll'])

const MIN_VISIBLE_MS = 400 // минимум показать спиннер чтобы юзер успел увидеть

const scrollEl = ref(null)
const { pullDistance, refreshing, dragging } = usePullToRefresh(scrollEl, async () => {
  const startedAt = Date.now()
  try {
    await props.loader()
  } catch (e) {
    if (typeof console !== 'undefined') console.warn('ptr loader failed:', e)
  }
  const elapsed = Date.now() - startedAt
  if (elapsed < MIN_VISIBLE_MS) {
    await new Promise((r) => setTimeout(r, MIN_VISIBLE_MS - elapsed))
  }
})

defineExpose({ scrollEl })

let ticking = false
function onScroll(e) {
  if (ticking) return
  ticking = true
  const y = e.target.scrollTop
  requestAnimationFrame(() => {
    emit('scroll', y)
    ticking = false
  })
}
</script>

<template>
  <div ref="scrollEl" class="scroll" @scroll.passive="onScroll">
    <div
      class="inner"
      :style="{
        transform: `translateY(${pullDistance}px)`,
        transition: dragging ? 'none' : 'transform .25s cubic-bezier(.2,.8,.2,1)',
      }"
    >
      <!-- Спиннер ABSOLUTE'ом над верхним краем inner'а — не в потоке,
           поэтому верх контента не клипится. На pull inner ползёт вниз
           вместе со спиннером (transform применяется к родителю). -->
      <div class="ptr" :class="{ spinning: refreshing || pullDistance >= 60 }">
        <span class="ptr-spinner"></span>
      </div>

      <slot></slot>
    </div>
  </div>
</template>

<style scoped>
.scroll {
  position: relative;
  flex: 1;
  overflow-y: auto;
  overscroll-behavior-y: contain;
}
.inner {
  position: relative; /* якорь для absolute .ptr */
  will-change: transform;
}

.ptr {
  position: absolute;
  top: -52px;
  left: 0;
  right: 0;
  height: 52px;
  display: flex;
  justify-content: center;
  align-items: flex-end;
  pointer-events: none;
  padding-bottom: 14px;
}
.ptr-spinner {
  width: 22px;
  height: 22px;
  border: 2px solid var(--border);
  border-top-color: var(--text);
  border-radius: 50%;
}
.ptr.spinning .ptr-spinner {
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
