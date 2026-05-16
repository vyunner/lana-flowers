<script setup>
defineProps({
  city: { type: String, required: true },
})
defineEmits(['open-city', 'open-sell'])
</script>

<template>
  <div class="topbar">
    <button class="city" type="button" @click="$emit('open-city')">
      <svg class="pin" viewBox="0 0 24 24" fill="none">
        <path
          d="M12 21s-7-6.5-7-12a7 7 0 1114 0c0 5.5-7 12-7 12z"
          stroke="currentColor"
          stroke-width="1.6"
        />
        <circle cx="12" cy="9.5" r="2.6" stroke="currentColor" stroke-width="1.6" />
      </svg>
      <span class="name">{{ city }}</span>
      <svg class="caret" viewBox="0 0 16 16" fill="none">
        <path
          d="M4 6l4 4 4-4"
          stroke="currentColor"
          stroke-width="1.6"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    </button>

    <button class="sell-btn" type="button" @click="$emit('open-sell')">
      <svg class="plus" viewBox="0 0 16 16" fill="none">
        <path d="M8 3v10M3 8h10" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
      </svg>
      Продать
    </button>
  </div>
</template>

<style scoped>
.topbar {
  /* --tg-safe-top из telegram.js (отступ под шапкой Telegram), env() —
     фолбэк для devices с notch. Снизу почти не паддим — следом идут чипсы
     со своим верхним отступом, иначе воздух дублируется. */
  padding-top: calc(max(var(--tg-safe-top, 0px), env(safe-area-inset-top, 0px)) + 22px);
  padding-right: 16px;
  padding-bottom: 4px;
  padding-left: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
}

.city {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 0;
  padding: 8px 4px;
}
.city .pin {
  width: 18px;
  height: 18px;
}
.city .name {
  font-weight: 700;
  font-size: 17px;
  letter-spacing: -0.01em;
}
.city .caret {
  width: 14px;
  height: 14px;
  opacity: 0.55;
}

.sell-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--primary);
  color: var(--primary-text);
  border: 0;
  border-radius: var(--radius-pill);
  padding: 9px 14px 9px 12px;
  font-weight: 600;
  font-size: 13px;
  letter-spacing: -0.005em;
  transition: transform 0.1s ease-out;
}
.sell-btn:active {
  transform: scale(0.95);
}
.sell-btn .plus {
  width: 14px;
  height: 14px;
}
</style>
