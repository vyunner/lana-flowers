<script setup>
import { onUnmounted, watch } from 'vue'
import { tg } from '../../telegram'

// BaseModal — центрированный диалог. Использовать для коротких форм/состояний
// (OfferSheet, CounterPriceModal, EditProfileModal). Для bottom-sheet
// контента (выбор из списка) — BaseSheet.
//
// Особенности:
// - Teleport в body, чтобы перекрывать ВСЁ (топбар, BottomNav)
// - Telegram BackButton показывается пока open=true, тап на ней = close
// - Esc + клик по overlay тоже закрывают (если closeOnOverlay=true)
// - Анимация scale 0.92→1 + opacity (snappy spring-easing)
const props = defineProps({
  open: { type: Boolean, required: true },
  closeOnOverlay: { type: Boolean, default: true },
})
const emit = defineEmits(['close'])

function close() {
  emit('close')
}

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
      @click="closeOnOverlay && close()"
    ></div>
    <div class="modal" :class="{ open }" role="dialog" aria-modal="true">
      <slot></slot>
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

.modal {
  position: fixed;
  top: 50%;
  left: 50%;
  width: calc(100% - 32px);
  max-width: 360px;
  background: var(--surface);
  color: var(--text);
  border-radius: 18px;
  padding: 22px 22px 20px;
  box-shadow: var(--shadow-modal);
  z-index: 31;
  transform: translate(-50%, -50%) scale(0.92);
  opacity: 0;
  pointer-events: none;
  transition:
    transform 0.22s cubic-bezier(0.34, 1.4, 0.64, 1),
    opacity 0.18s ease-out;
}
.modal.open {
  transform: translate(-50%, -50%) scale(1);
  opacity: 1;
  pointer-events: auto;
}
</style>
