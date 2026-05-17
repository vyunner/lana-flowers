<script setup>
import { formatPrice } from '../../utils/format'
import { thumbUrl } from '../../utils/image'

// Компактная строка моего активного букета без офферов (секция «На продаже»).
// Действие — только «Снять» как text-link справа. CTA нет: смотреть особо не на что,
// букет просто висит и ждёт покупателей.

const props = defineProps({
  bouquet: { type: Object, required: true },
  busy: { type: Boolean, default: false },
})
defineEmits(['remove'])
</script>

<template>
  <li class="compact-row">
    <div
      class="compact-thumb"
      :style="{ backgroundImage: `url(${thumbUrl(bouquet.photos?.[0] || '')})` }"
    ></div>
    <div class="compact-body">
      <div class="compact-title">{{ bouquet.title }}</div>
      <div class="compact-sub">{{ formatPrice(bouquet.price) }} ₸</div>
    </div>
    <button
      class="link danger compact-action"
      type="button"
      :disabled="busy"
      @click="$emit('remove', bouquet)"
    >
      Снять
    </button>
  </li>
</template>

<style scoped src="./_shared.css"></style>
