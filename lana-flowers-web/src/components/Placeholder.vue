<script setup>
defineProps({
  icon: { type: String, default: 'profile' },
  title: { type: String, default: 'Скоро' },
  subtitle: { type: String, default: '' },
})
</script>

<template>
  <div class="placeholder">
    <svg v-if="icon === 'messages'" class="ph-icon" viewBox="0 0 28 28" fill="none">
      <path
        d="M5 9.5a3.5 3.5 0 013.5-3.5h11A3.5 3.5 0 0123 9.5v7a3.5 3.5 0 01-3.5 3.5H12l-4.5 3.5V20H8.5A3.5 3.5 0 015 16.5v-7z"
        stroke="currentColor"
        stroke-width="1.6"
        stroke-linejoin="round"
      />
    </svg>
    <svg v-else class="ph-icon" viewBox="0 0 28 28" fill="none">
      <circle cx="14" cy="10.5" r="4.2" stroke="currentColor" stroke-width="1.6" />
      <path
        d="M5.5 23c1.6-3.8 4.8-5.8 8.5-5.8s6.9 2 8.5 5.8"
        stroke="currentColor"
        stroke-width="1.6"
        stroke-linecap="round"
      />
    </svg>
    <div class="ph-text">{{ title }}</div>
    <div class="ph-sub">{{ subtitle }}</div>
  </div>
</template>

<style scoped>
.placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding-bottom: 80px;
  color: var(--text-muted);
}
.ph-icon {
  width: 48px;
  height: 48px;
  opacity: 0.55;
}
.ph-text {
  font-size: 17px;
  font-weight: 600;
  color: var(--secondary-text);
  letter-spacing: -0.005em;
}
.ph-sub {
  font-size: 13px;
  color: var(--text-muted);
  text-align: center;
  padding: 0 32px;
}
</style>
