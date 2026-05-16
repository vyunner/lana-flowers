<script setup>
import { cities } from '../data/cities'
import BaseSheet from './base/BaseSheet.vue'

defineProps({
  open: { type: Boolean, required: true },
  current: { type: String, required: true },
})
defineEmits(['select', 'close'])
</script>

<template>
  <!-- level=2 — CitySheet может открываться поверх SellSheet (юзер тапает
       «Город» внутри формы создания объявления). Без level CitySheet
       сидел бы на том же z-index что и SellSheet и тонул под ним. -->
  <BaseSheet :open="open" :level="2" @close="$emit('close')">
    <h2 class="title">Выберите город</h2>
    <ul class="list">
      <li
        v-for="c in cities"
        :key="c"
        :class="{ selected: c === current }"
        @click="$emit('select', c)"
      >
        <span>{{ c }}</span>
        <svg class="check" viewBox="0 0 20 20" fill="none" aria-hidden="true">
          <path d="M4 10.5l3.8 3.8L16 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </li>
    </ul>
  </BaseSheet>
</template>

<style scoped>
.title {
  text-align: center;
  font-size: 17px;
  font-weight: 700;
  margin: 0 0 12px;
  letter-spacing: -0.005em;
}

.list {
  list-style: none;
  margin: 0;
  padding: 0 0 8px;
}
.list li {
  padding: 14px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-radius: 12px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 500;
  transition: background 0.15s ease-out;
}
.list li:active {
  background: var(--surface-2);
}
.list li .check {
  width: 18px;
  height: 18px;
  opacity: 0;
  transition: opacity 0.15s ease-out;
}
.list li.selected .check {
  opacity: 1;
}
</style>
