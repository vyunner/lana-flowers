<script setup>
import { toasts, dismissToast } from '../../state/toasts'
import { haptic } from '../../telegram'

// Стек тоастов в правом-верхнем углу. Появляются slide-from-top,
// автодизмисс через store. Тап → закрывается + haptic feedback.
// Telegram-шапка имеет 0 высоты внутри нашего viewport'а, поэтому
// top: 8px + safe-area-inset-top рисует под ней.

function onTap(t) {
  haptic('light')
  if (typeof t.action === 'function') {
    try { t.action() } catch {}
  }
  dismissToast(t.id)
}
</script>

<template>
  <Teleport to="body">
    <div class="stack">
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          class="toast"
          :class="[t.kind, { actionable: !!t.action }]"
          @click="onTap(t)"
          role="status"
          aria-live="polite"
        >
          <span class="msg">{{ t.text }}</span>
          <span v-if="t.action" class="chev">›</span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.stack {
  position: fixed;
  top: calc(8px + env(safe-area-inset-top, 0px));
  left: 12px;
  right: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 100; /* поверх всех модалок, сheets, BackButton'а Telegram */
  pointer-events: none;
}
.toast {
  pointer-events: auto;
  background: var(--text);
  color: var(--bg);
  padding: 12px 14px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 500;
  line-height: 1.35;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
  cursor: pointer;
  user-select: none;
}
.toast.success {
  background: #2c8a52;
  color: #fff;
}
.toast.warn {
  background: var(--accent);
  color: var(--accent-text);
}
.toast.err {
  background: #d6553f;
  color: #fff;
}

.toast.actionable {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.toast .msg {
  /* flex:1 + min-width:0 — критично: без min-width:0 flex-item не
     уменьшается ниже intrinsic ширины текста, и длинный текст пушит
     чевронку за край тоста. С min-width:0 текст переносится корректно. */
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}
.toast .chev {
  font-size: 20px;
  font-weight: 400;
  opacity: 0.7;
  flex-shrink: 0;
}

/* Анимации */
.toast-enter-from {
  opacity: 0;
  transform: translateY(-20px);
}
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.25s ease-out, transform 0.25s ease-out;
}
.toast-leave-to {
  opacity: 0;
  transform: translateY(-12px);
}
.toast-move {
  transition: transform 0.25s ease-out;
}
</style>
