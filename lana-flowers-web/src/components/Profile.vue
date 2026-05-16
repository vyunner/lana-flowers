<script setup>
import { computed, onMounted, ref } from 'vue'
import { haptic, tg } from '../telegram'
import { getMyBouquets, deleteBouquet } from '../api/bouquets'
import { useApi } from '../composables/useApi'
import { usePolling } from '../composables/usePolling'
import { me } from '../state/auth'
import { formatPrice } from '../utils/format'
import { confirm, alert } from '../utils/dialog'
import EditProfileModal from './EditProfileModal.vue'
import PullToRefreshScroll from './base/PullToRefreshScroll.vue'

const WHATSAPP_URL = 'https://wa.me/77026207447'

function openWhatsApp() {
  haptic('light')
  if (tg && typeof tg.openLink === 'function') {
    tg.openLink(WHATSAPP_URL)
  } else {
    window.open(WHATSAPP_URL, '_blank')
  }
}

const userName = computed(
  () => me.value?.display_name || me.value?.first_name || 'Без имени',
)
const userPhoto = computed(() => me.value?.avatar_url || null)

const editOpen = ref(false)

// ---- API ----
const myBouquets = useApi(getMyBouquets)

async function loadAll() {
  await myBouquets.run()
}
onMounted(loadAll)
defineExpose({ refresh: loadAll })

// Polling — safety net (60с). Основные апдейты приходят через SSE.
usePolling(loadAll, 60000)

async function removeBouquet(b) {
  if (!(await confirm(`Снять с продажи «${b.title}»?`))) return
  try {
    await deleteBouquet(b.id)
    haptic('medium')
    await loadAll()
  } catch (e) {
    await alert('Ошибка: ' + (e.message || e))
  }
}

function statusLabel(s) {
  return (
    {
      active: 'Активно',
      sold: 'Продано',
      archived: 'Снято',
    }[s] || s
  )
}
function statusColor(s) {
  if (s === 'active') return 'ok'
  if (s === 'archived') return 'err'
  return 'mute'
}
</script>

<template>
  <PullToRefreshScroll :loader="loadAll">
    <div class="profile-inner">
      <header class="profile-head">
        <div class="avatar" :class="{ 'is-placeholder': !userPhoto }">
          <img v-if="userPhoto" :src="userPhoto" :alt="userName" />
          <svg v-else viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" />
          </svg>
        </div>
        <div class="name">{{ userName }}</div>
        <button class="edit-btn" type="button" @click="editOpen = true">
          Редактировать профиль
        </button>
      </header>

      <section class="section">
        <h3 class="sec-title">
          Мои объявления
          <span v-if="myBouquets.data.value?.length" class="count">
            {{ myBouquets.data.value.length }}
          </span>
        </h3>
        <div v-if="myBouquets.loading.value && !myBouquets.data.value" class="empty">
          Загружаю…
        </div>
        <div v-else-if="!myBouquets.data.value?.length" class="empty">
          Пока ничего не опубликовано
        </div>
        <ul v-else class="rows">
          <li v-for="b in myBouquets.data.value" :key="b.id" class="row">
            <div
              class="thumb"
              :style="{ backgroundImage: `url(${b.photos?.[0] || ''})` }"
            ></div>
            <div class="row-body">
              <div class="row-title">{{ b.title }}</div>
              <div class="row-sub">
                {{ formatPrice(b.price) }} ₸ ·
                <span :class="['st', statusColor(b.status)]">{{ statusLabel(b.status) }}</span>
              </div>
            </div>
            <button
              v-if="b.status === 'active'"
              class="remove-btn"
              type="button"
              aria-label="Снять с продажи"
              @click="removeBouquet(b)"
            >
              <svg viewBox="0 0 24 24" fill="none">
                <path d="M5 7h14M9 7V5a1 1 0 011-1h4a1 1 0 011 1v2M7 7l1 12a1 1 0 001 1h6a1 1 0 001-1l1-12" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
          </li>
        </ul>
      </section>

      <button class="whatsapp-btn" type="button" @click="openWhatsApp">
        <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.297-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.71.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893A11.821 11.821 0 0020.464 3.488" />
        </svg>
        Поддержка в WhatsApp
      </button>

      <p class="ver">Lana Flowers · v0.2.0</p>
    </div>

    <EditProfileModal :open="editOpen" @close="editOpen = false" />
  </PullToRefreshScroll>
</template>

<style scoped>
.profile-inner {
  padding: 8px 16px calc(120px + env(safe-area-inset-bottom, 0px));
}

.profile-head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px 0 20px;
}
.avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--surface-2);
}
.avatar img {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
}
.avatar.is-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}
.avatar.is-placeholder svg {
  width: 56%;
  height: 56%;
  display: block;
}
.name {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.01em;
}
.edit-btn {
  margin-top: 6px;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 500;
  transition: background 0.15s, border-color 0.15s;
}
.edit-btn:active {
  background: var(--surface-2);
  border-color: var(--text);
}

.section {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  margin-bottom: 12px;
  padding: 14px 16px 16px;
}
.sec-title {
  font-size: 17px;
  font-weight: 700;
  margin: 4px 0 14px;
  display: flex;
  align-items: baseline;
  gap: 8px;
  color: var(--text);
  letter-spacing: -0.01em;
}
.count {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
}

.empty {
  color: var(--text-muted);
  font-size: 14px;
  padding: 10px 0;
}

.rows {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.thumb {
  width: 52px;
  height: 52px;
  border-radius: 10px;
  background: var(--surface-2);
  background-size: cover;
  background-position: center;
  flex-shrink: 0;
}
.row-body {
  flex: 1;
  min-width: 0;
}
.row-title {
  font-size: 15px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row-sub {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 3px;
}
.remove-btn {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: background 0.15s, color 0.15s;
}
.remove-btn:active {
  background: var(--surface-2);
  color: #d6553f;
}
.remove-btn svg {
  width: 18px;
  height: 18px;
}
.st.ok { color: #2c8a52; font-weight: 600; }
.st.err { color: #d6553f; font-weight: 600; }
.st.mute { color: var(--text-muted); }

.whatsapp-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  height: 46px;
  background: transparent;
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: var(--radius-button);
  font-size: 14px;
  font-weight: 600;
  margin-top: 4px;
  transition: background 0.15s, transform 0.1s ease-out;
}
.whatsapp-btn:active {
  transform: scale(0.98);
  background: var(--surface-2);
}
.whatsapp-btn svg {
  width: 20px;
  height: 20px;
  color: #25d366;
}

.ver {
  text-align: center;
  font-size: 12px;
  color: var(--text-muted);
  margin: 18px 0 0;
}
</style>
