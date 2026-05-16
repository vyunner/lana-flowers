<script setup>
import { ref } from 'vue'
import { tg, haptic, hapticNotify } from '../telegram'
import { getMe } from '../api/users'
import { setMe } from '../state/auth'

const status = ref('idle') // idle | requesting | checking | error
const errorText = ref('')

async function waitUntilRegistered(maxTries = 10, delayMs = 500) {
  for (let i = 0; i < maxTries; i++) {
    try {
      const u = await getMe()
      if (u.is_registered) {
        setMe(u)
        return true
      }
    } catch {}
    await new Promise((r) => setTimeout(r, delayMs))
  }
  return false
}

async function share() {
  errorText.value = ''
  haptic('light')

  if (tg && typeof tg.requestContact === 'function') {
    status.value = 'requesting'
    tg.requestContact(async (ok) => {
      if (!ok) {
        status.value = 'idle'
        return
      }
      status.value = 'checking'
      const registered = await waitUntilRegistered()
      if (registered) {
        hapticNotify('success')
      } else {
        status.value = 'error'
        errorText.value = 'Не получилось получить номер. Попробуйте ещё раз.'
      }
    })
    return
  }

  // Фоллбэк — открываем бот
  const url = 'https://t.me/lana_flowers_bot'
  if (tg && typeof tg.openTelegramLink === 'function') {
    tg.openTelegramLink(url)
  } else {
    window.open(url, '_blank')
  }
}

async function recheck() {
  status.value = 'checking'
  errorText.value = ''
  const ok = await waitUntilRegistered(1, 0)
  if (!ok) {
    status.value = 'idle'
    errorText.value = 'Пока не получили номер. Попробуйте ещё раз.'
  }
}
</script>

<template>
  <div class="step">
    <div class="card">
      <h1 class="title">Последний шаг</h1>
      <p class="lead">
        Чтобы завершить регистрацию, отправьте свой номер телефона.
      </p>

      <button
        class="primary-btn"
        type="button"
        :disabled="status === 'requesting' || status === 'checking'"
        @click="share"
      >
        <span v-if="status === 'requesting'">Жду подтверждения…</span>
        <span v-else-if="status === 'checking'">Проверяю…</span>
        <span v-else>Поделиться номером</span>
      </button>

      <button
        v-if="status === 'error' || errorText"
        class="link-btn"
        type="button"
        @click="recheck"
      >
        Проверить заново
      </button>

      <p v-if="errorText" class="err">{{ errorText }}</p>
    </div>
  </div>
</template>

<style scoped>
.step {
  position: absolute;
  inset: 0;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  /* Шапка Telegram съедает верх + визуальный центр выше геометрического —
     компенсируем большим padding-bottom чтобы контент сел выше. */
  padding: 24px 24px calc(22vh + env(safe-area-inset-bottom, 0px));
  z-index: 100;
}
.card {
  width: 100%;
  max-width: 340px;
  text-align: center;
}
.title {
  font-size: 26px;
  font-weight: 700;
  margin: 0 0 12px;
  letter-spacing: -0.02em;
  color: var(--text);
  line-height: 1.2;
}
.lead {
  margin: 0 0 28px;
  font-size: 15px;
  color: var(--text-secondary);
  line-height: 1.45;
}
.primary-btn {
  width: 100%;
  height: 52px;
  border: 0;
  border-radius: var(--radius-button);
  background: var(--primary);
  color: var(--primary-text);
  font-size: 16px;
  font-weight: 700;
  transition: transform 0.1s ease-out, opacity 0.15s;
}
.primary-btn:active:not(:disabled) {
  transform: scale(0.98);
}
.primary-btn:disabled {
  opacity: 0.6;
}
.link-btn {
  margin-top: 14px;
  background: transparent;
  border: 0;
  color: var(--accent);
  font-size: 14px;
  font-weight: 600;
  text-decoration: underline;
}
.err {
  margin-top: 12px;
  font-size: 13px;
  color: #d6553f;
}
</style>
