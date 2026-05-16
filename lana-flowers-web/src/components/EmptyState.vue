<script setup>
defineProps({
  text: { type: String, default: 'В этой категории пока нет букетов' },
})
</script>

<template>
  <div class="empty-state">
    <svg viewBox="0 0 48 48" fill="none">
      <g fill="currentColor">
        <ellipse cx="24" cy="13" rx="6" ry="8" />
        <ellipse cx="24" cy="35" rx="6" ry="8" />
        <ellipse cx="13" cy="24" rx="8" ry="6" />
        <ellipse cx="35" cy="24" rx="8" ry="6" />
      </g>
    </svg>
    <p>{{ text }}</p>
  </div>
</template>

<style scoped>
.empty-state {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 80px 24px;
  text-align: center;
}
.empty-state svg {
  width: 48px;
  height: 48px;
  color: #c9c5bb;
}
.empty-state p {
  margin: 0;
  font-size: 14px;
  color: var(--text-muted);
  max-width: 220px;
  line-height: 1.35;
  white-space: pre-line;
}
</style>
