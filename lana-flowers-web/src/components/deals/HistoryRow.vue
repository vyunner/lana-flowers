<script setup>
import { thumbUrl } from '../../utils/image'

// Generic display-only строка для секции «История» — рендерит
// уже подготовленные родителем title/sub/photo. Без действий и без
// состояний, чтобы один компонент закрывал и завершённые офферы,
// и проданные/снятые букеты — родитель решает что подать.

const props = defineProps({
  title: { type: String, required: true },
  sub: { type: String, default: '' },
  photo: { type: String, default: '' },
})
</script>

<template>
  <li class="compact-row history-row">
    <div
      class="compact-thumb"
      :style="{ backgroundImage: `url(${thumbUrl(photo)})` }"
    ></div>
    <div class="compact-body">
      <div class="compact-title">{{ title }}</div>
      <div v-if="sub" class="compact-sub">{{ sub }}</div>
    </div>
  </li>
</template>

<style scoped src="./_shared.css"></style>
