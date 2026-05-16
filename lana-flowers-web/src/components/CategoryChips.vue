<script setup>
defineProps({
  categories: { type: Array, required: true },
  active: { type: String, required: true },
})
defineEmits(['select'])
</script>

<template>
  <div class="chips">
    <button
      v-for="cat in categories"
      :key="cat.key"
      class="chip-item"
      :class="{ active: active === cat.key }"
      type="button"
      @click="$emit('select', cat.key)"
    >
      {{ cat.label }}
    </button>
  </div>
</template>

<style scoped>
.chips {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  /* Сверху 2px чтобы не дублировать воздух с топбаром, снизу — нормальный
     отступ перед сеткой карточек. */
  padding: 2px 16px 12px;
  flex-shrink: 0;
}

.chip-item {
  flex-shrink: 0;
  font-size: 12.5px;
  padding: 7px 12px;
  border-radius: var(--radius-pill);
  border: 1px solid var(--secondary-border);
  color: var(--secondary-text);
  font-weight: 500;
  white-space: nowrap;
  background: transparent;
  transition:
    background-color 0.2s ease-in-out,
    color 0.2s ease-in-out,
    border-color 0.2s ease-in-out;
}
.chip-item.active {
  background: var(--primary);
  color: var(--primary-text);
  border-color: var(--primary);
}
</style>
