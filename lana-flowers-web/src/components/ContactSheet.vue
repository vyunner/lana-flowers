<script setup>
import { computed } from 'vue'
import { haptic, tg } from '../telegram'

const props = defineProps({
  open: { type: Boolean, required: true },
  // counterparty = { name, avatar_url, phone } (phone обязателен — sheet
  // открывается ТОЛЬКО для accepted-сделок, где бэк отдал телефон)
  counterparty: { type: Object, default: null },
})
defineEmits(['close'])

// Нормализуем телефон в международный формат без +/пробелов для wa.me/tel:
const phoneClean = computed(() => {
  const raw = props.counterparty?.phone || ''
  return raw.replace(/\D/g, '')
})

const phoneDisplay = computed(() => {
  // +7 705 123 45 67 — читаемый формат для отображения
  const p = phoneClean.value
  if (p.length === 11 && (p[0] === '7' || p[0] === '8')) {
    return `+7 ${p.slice(1, 4)} ${p.slice(4, 7)} ${p.slice(7, 9)} ${p.slice(9, 11)}`
  }
  return props.counterparty?.phone || ''
})

function openLink(url, hap = 'light') {
  haptic(hap)
  if (tg && typeof tg.openLink === 'function') {
    tg.openLink(url)
  } else {
    window.open(url, '_blank')
  }
}

function openTel() {
  // tel: — Telegram WebApp openLink не любит non-https, fallback на window.
  haptic('medium')
  window.location.href = 'tel:+' + phoneClean.value
}

function openWhatsApp() {
  openLink('https://wa.me/' + phoneClean.value)
}

function openTelegram() {
  // tg://resolve?phone=... — стандартная схема, работает в Telegram.
  haptic('light')
  if (tg && typeof tg.openTelegramLink === 'function') {
    tg.openTelegramLink('https://t.me/+' + phoneClean.value)
  } else {
    window.open('https://t.me/+' + phoneClean.value, '_blank')
  }
}
</script>

<template>
  <Teleport to="body">
    <div class="overlay" :class="{ open }" @click="$emit('close')"></div>
    <div class="sheet" :class="{ open }" role="dialog" aria-modal="true">
      <div class="head">
        <div class="avatar" :class="{ 'is-placeholder': !counterparty?.avatar_url }">
          <img v-if="counterparty?.avatar_url" :src="counterparty.avatar_url" :alt="counterparty.name" />
          <svg v-else viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" />
          </svg>
        </div>
        <div class="name">{{ counterparty?.name || 'Контакт' }}</div>
        <div class="phone">{{ phoneDisplay }}</div>
      </div>

      <div class="actions">
        <button class="act primary" type="button" @click="openTel">
          <svg viewBox="0 0 24 24" fill="none">
            <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07 19.5 19.5 0 01-6-6 19.79 19.79 0 01-3.07-8.67A2 2 0 014.11 2h3a2 2 0 012 1.72c.13.96.37 1.9.72 2.8a2 2 0 01-.45 2.11L8.09 9.91a16 16 0 006 6l1.27-1.27a2 2 0 012.11-.45c.9.35 1.84.59 2.8.72A2 2 0 0122 16.92z"
              stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/>
          </svg>
          Позвонить
        </button>
        <button class="act" type="button" @click="openWhatsApp">
          <svg viewBox="0 0 24 24" fill="currentColor" style="color:#25d366">
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.297-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.71.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893A11.821 11.821 0 0020.464 3.488"/>
          </svg>
          WhatsApp
        </button>
        <button class="act" type="button" @click="openTelegram">
          <svg viewBox="0 0 24 24" fill="currentColor" style="color:#229ED9">
            <path d="M9.78 18.65l.28-4.23 7.68-6.92c.34-.31-.07-.46-.52-.19L7.74 13.24 3.64 11.95c-.88-.25-.89-.86.2-1.3l15.97-6.16c.73-.33 1.43.18 1.15 1.3l-2.72 12.81c-.19.91-.74 1.13-1.5.7L12.6 16.3l-1.99 1.93c-.23.23-.42.42-.83.42z"/>
          </svg>
          Telegram
        </button>
      </div>

      <button class="close" type="button" @click="$emit('close')">Закрыть</button>
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
.overlay.open { opacity: 1; pointer-events: auto; }

.sheet {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--surface);
  color: var(--text);
  border-radius: 18px 18px 0 0;
  padding: 24px 20px calc(20px + env(safe-area-inset-bottom, 0px));
  box-shadow: var(--shadow-sheet);
  z-index: 31;
  transform: translateY(100%);
  transition: transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.sheet.open { transform: translateY(0); }

.head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  margin-bottom: 22px;
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
  grid-template-columns: 1fr 1fr 1fr;
  gap: 8px;
  margin-bottom: 12px;
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
  transition: transform 0.1s, background 0.15s;
}
.act:active { transform: scale(0.96); background: var(--border); }
.act svg { width: 22px; height: 22px; }
.act.primary {
  background: var(--text);
  color: var(--bg);
}
.act.primary svg { color: currentColor; }

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
