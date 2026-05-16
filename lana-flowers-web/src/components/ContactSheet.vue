<script setup>
import { computed } from 'vue'
import { haptic, tg } from '../telegram'
import { formatPhoneKz } from '../utils/format'
import BaseSheet from './base/BaseSheet.vue'

const props = defineProps({
  open: { type: Boolean, required: true },
  // counterparty = { name, avatar_url, phone, username } (phone обязателен —
  // sheet открывается ТОЛЬКО для accepted-сделок, где бэк отдал контакты;
  // username — необязателен, многие юзеры в KZ без @username)
  counterparty: { type: Object, default: null },
})
defineEmits(['close'])

const phoneClean = computed(() => (props.counterparty?.phone || '').replace(/\D/g, ''))
const phoneDisplay = computed(() => formatPhoneKz(props.counterparty?.phone || ''))
const waHref = computed(() => 'https://wa.me/' + phoneClean.value)
const username = computed(() => (props.counterparty?.username || '').trim())
const hasTelegram = computed(() => !!username.value)

function onWhatsAppTap(e) {
  // wa.me — это https, надёжнее открыть через Telegram openLink (не схлопывает
  // мини-апп), а не дефолтом по href. preventDefault'им навигацию.
  e.preventDefault()
  haptic('light')
  if (tg && typeof tg.openLink === 'function') {
    tg.openLink(waHref.value)
  } else {
    window.open(waHref.value, '_blank')
  }
}

function openTelegram() {
  if (!hasTelegram.value) return
  haptic('light')
  const url = 'https://t.me/' + username.value.replace(/^@/, '')
  // openTelegramLink — нативный путь открыть чат по @username. Открытие
  // чата по НОМЕРУ через ссылки Telegram не поддерживается, поэтому если
  // username пустой — кнопка вообще скрыта.
  if (tg && typeof tg.openTelegramLink === 'function') {
    tg.openTelegramLink(url)
  } else {
    window.open(url, '_blank')
  }
}
</script>

<template>
  <BaseSheet :open="open" @close="$emit('close')">
    <div class="head">
      <div class="avatar" :class="{ 'is-placeholder': !counterparty?.avatar_url }">
        <img v-if="counterparty?.avatar_url" :src="counterparty.avatar_url" :alt="counterparty.name" />
        <svg v-else viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" />
        </svg>
      </div>
      <div class="name">{{ counterparty?.name || 'Контакт' }}</div>
      <div class="phone">{{ phoneDisplay }}</div>
    </div>

    <div class="actions" :class="{ 'two-cols': hasTelegram }">
      <a class="act" :href="waHref" @click="onWhatsAppTap">
        <svg viewBox="0 0 24 24" fill="currentColor" style="color:#25d366">
          <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.297-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.71.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893A11.821 11.821 0 0020.464 3.488"/>
        </svg>
        WhatsApp
      </a>
      <button v-if="hasTelegram" class="act" type="button" @click="openTelegram">
        <svg viewBox="0 0 24 24" fill="currentColor" style="color:#229ED9">
          <path d="M9.78 18.65l.28-4.23 7.68-6.92c.34-.31-.07-.46-.52-.19L7.74 13.24 3.64 11.95c-.88-.25-.89-.86.2-1.3l15.97-6.16c.73-.33 1.43.18 1.15 1.3l-2.72 12.81c-.19.91-.74 1.13-1.5.7L12.6 16.3l-1.99 1.93c-.23.23-.42.42-.83.42z"/>
        </svg>
        Telegram
      </button>
    </div>

    <button class="close" type="button" @click="$emit('close')">Закрыть</button>
  </BaseSheet>
</template>

<style scoped>
.head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  margin: 6px 0 22px;
}
.avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--surface-2);
  margin-bottom: 6px;
}
.avatar img { width: 100%; height: 100%; object-fit: cover; display: block; }
.avatar.is-placeholder {
  display: flex; align-items: center; justify-content: center; color: var(--text-muted);
}
.avatar.is-placeholder svg { width: 56%; height: 56%; }
.name { font-size: 17px; font-weight: 700; letter-spacing: -0.01em; }
.phone { font-size: 14px; color: var(--text-secondary); font-variant-numeric: tabular-nums; }

.actions {
  display: grid;
  /* По умолчанию одна кнопка (WhatsApp) — растягиваем во всю ширину. */
  grid-template-columns: 1fr;
  gap: 8px;
  margin-bottom: 12px;
}
.actions.two-cols {
  /* Если есть Telegram — две колонки поровну. */
  grid-template-columns: 1fr 1fr;
}
.act {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 14px 8px;
  border-radius: 12px;
  background: var(--surface-2);
  color: var(--text);
  border: 0;
  font-size: 12px;
  font-weight: 600;
  text-decoration: none; /* <a> тоже .act — снимаем подчёркивание */
  transition: transform 0.1s, background 0.15s;
}
.act:active { transform: scale(0.96); background: var(--border); }
.act svg { width: 22px; height: 22px; }

.close {
  width: 100%;
  height: 46px;
  background: transparent;
  color: var(--text-secondary);
  border: 0;
  font-size: 14px;
  font-weight: 500;
}
</style>
