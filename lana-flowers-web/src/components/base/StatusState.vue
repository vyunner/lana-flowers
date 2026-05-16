<script setup>
// StatusState — заглушка-сабэкран внутри BaseModal: иконка (опционально) +
// заголовок + текст + одна кнопка-CTA. Используется для success-экранов и
// «бизнесовых» ошибок (DUPLICATE_OFFER / SELF_OFFER / BOUQUET_UNAVAILABLE).
defineProps({
  title: { type: String, required: true },
  // icon — emoji или короткий символ. Опционально, чтобы не лепить смайл
  // где он лишний (юзер уже просил «убрать смайлики»).
  icon: { type: String, default: '' },
  cta: { type: String, default: 'Понятно' },
})
defineEmits(['cta'])
</script>

<template>
  <div class="state">
    <div v-if="icon" class="icon">{{ icon }}</div>
    <h3 class="title">{{ title }}</h3>
    <p class="text">
      <slot></slot>
    </p>
    <button class="btn" type="button" @click="$emit('cta')">{{ cta }}</button>
  </div>
</template>

<style scoped>
.state {
  text-align: center;
  padding: 8px 0 4px;
}
.icon {
  font-size: 48px;
  line-height: 1;
  margin-bottom: 14px;
}
.title {
  font-size: 19px;
  font-weight: 700;
  margin: 0 0 8px;
  color: var(--text);
}
.text {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 22px;
  line-height: 1.4;
}
.btn {
  width: 100%;
  height: 52px;
  border-radius: var(--radius-button);
  background: var(--accent);
  color: var(--accent-text);
  border: 0;
  font-size: 16px;
  font-weight: 700;
  transition: background 0.15s, transform 0.1s;
}
.btn:active {
  background: var(--accent-hover);
  transform: scale(0.98);
}
</style>
