<script setup>
import { onMounted, ref } from 'vue'
import { categories } from './data/categories'
import { haptic, hapticNotify } from './telegram'
import { currentStep, setMe } from './state/auth'
import { getMe } from './api/users'
import TopBar from './components/TopBar.vue'
import CategoryChips from './components/CategoryChips.vue'
import BouquetGrid from './components/BouquetGrid.vue'
import BottomNav from './components/BottomNav.vue'
import Deals from './components/Deals.vue'
import Profile from './components/Profile.vue'
import CitySheet from './components/CitySheet.vue'
import SellSheet from './components/SellSheet.vue'
import OfferSheet from './components/OfferSheet.vue'
import BouquetDetail from './components/BouquetDetail.vue'
import OnboardingPhone from './components/OnboardingPhone.vue'
import OnboardingName from './components/OnboardingName.vue'
import OnboardingAvatar from './components/OnboardingAvatar.vue'
import OnboardingCity from './components/OnboardingCity.vue'
import Toast from './components/base/Toast.vue'
import { useEventStream } from './composables/useEventStream'
import { useToastDispatch } from './composables/useToastDispatch'
const authReady = ref(false)

onMounted(async () => {
  try {
    const u = await getMe()
    setMe(u)
    // Если у юзера задан город — каталог сразу открывается в нём
    // (а не в дефолтной Алматы). selectedCity создан раньше с дефолтом
    // на случай если getMe ещё не успел.
    if (u?.city) selectedCity.value = u.city
  } catch {
    // /users/me не должен фейлиться при валидном initData
  } finally {
    authReady.value = true
  }

  // Deeplink из Telegram-DM: WebApp inline-кнопки в боте открывают мини-апп
  // c URL'ом вида https://.../?screen=deals|profile|catalog. Если screen
  // валидный — стартуем сразу на нужном табе.
  try {
    const screen = new URLSearchParams(window.location.search).get('screen')
    if (screen === 'deals' || screen === 'profile' || screen === 'catalog') {
      activeTab.value = screen
    }
  } catch {}
})

// ---- Tabs ----
const activeTab = ref('catalog')

// Бейдж: количество офферов ждущих ответа от меня (sets by Deals on load)
const dealsActionable = ref(0)
function onDealsUpdated(n) {
  dealsActionable.value = n
}

// ---- Category ----
const activeCategory = ref('all')

// ---- City: один общий стейт для шапки и формы продажи ----
const selectedCity = ref('Алматы')

const cityOpen = ref(false)
function openCitySheet() {
  cityOpen.value = true
}
function selectCity(c) {
  selectedCity.value = c
  cityOpen.value = false
}

// ---- Refs to children ----
const gridRef = ref(null)
const profileRef = ref(null)
const dealsRef = ref(null)

// ---- Sell sheet ----
const sellOpen = ref(false)
function onPublished() {
  hapticNotify('success')
  sellOpen.value = false
  gridRef.value?.refresh?.()
  // Букеты теперь живут во вкладке Сделки (секция «На продаже»), не в Профиле.
  dealsRef.value?.refresh?.()
}

// ---- Bouquet detail ----
const detailBouquet = ref(null)
function openDetail(b) {
  haptic('light')
  detailBouquet.value = b
}
function closeDetail() {
  detailBouquet.value = null
}

// ---- Offer sheet ----
const offerOpen = ref(false)
const offerBouquet = ref(null)
function openOffer(bouquet) {
  haptic('light')
  offerBouquet.value = bouquet
  offerOpen.value = true
}
// Из деталь-экрана прилетает СЫРОЙ bouquet (не adapted), OfferSheet ждёт
// объект с .price (строкой типа "10 000 ₸") и .raw — собираем на лету.
function openOfferFromDetail(rawBouquet) {
  haptic('light')
  const ruFmt = new Intl.NumberFormat('ru-RU')
  offerBouquet.value = {
    id: rawBouquet.id,
    price: ruFmt.format(rawBouquet.price) + ' ₸',
    name: rawBouquet.title,
    raw: rawBouquet,
  }
  offerOpen.value = true
}
function onOfferSubmitted() {
  hapticNotify('success')
  // Модалку не закрываем — там покажется success-экран который юзер закроет сам.
  // Освежаем ленту чтобы карточка получила бейдж «Предложено».
  gridRef.value?.refresh?.()
  profileRef.value?.refresh?.()
  // Сделки тоже — у юзера появилась новая отправленная.
  dealsRef.value?.refresh?.()
}

// ---- Navbar collapse on scroll ----
const navCollapsed = ref(false)
let lastScrollY = 0
function onFeedScroll(y) {
  const dy = y - lastScrollY
  if (Math.abs(dy) >= 4) {
    if (dy > 0 && y > 24) navCollapsed.value = true
    else if (dy < 0) navCollapsed.value = false
    lastScrollY = y
  }
}

function selectTab(t) {
  haptic('light')
  activeTab.value = t
}

// ---- SSE: in-app push для событий когда юзер сидит в мини-аппе ----
// Telegram не показывает push-уведомления когда юзер «в чате с ботом»
// (а мини-апп открытый — это и есть «в чате»). Поэтому при event'е
// рисуем toast (через useToastDispatch) и рефрешим активную вкладку
// чтобы UI был консистентен с тем что бэк только что сообщил.
// Polling 60с остаётся safety-net'ом на случай если SSE лопнуло.
const refreshOpenTabs = () => {
  gridRef.value?.refresh?.()
  dealsRef.value?.refresh?.()
  profileRef.value?.refresh?.()
}
const dispatchToast = useToastDispatch({
  onToastTap: () => { activeTab.value = 'deals' },
})
function handleEvent(e) {
  dispatchToast(e)
  refreshOpenTabs()
}

// onConnect: после установки SSE рефрешим всё что открыто — лечит
// init-race (между fetch'ем при mount и подпиской могло проскочить
// событие, которое навсегда потеряно).
useEventStream(handleEvent, refreshOpenTabs)
</script>

<template>
  <!-- 1) Сплеш пока ждём /users/me -->
  <div v-if="!authReady" class="boot-splash">
    <span class="spinner"></span>
  </div>

  <!-- 2) Этапы онбординга -->
  <OnboardingPhone v-else-if="currentStep === 'phone'" />
  <OnboardingName v-else-if="currentStep === 'name'" />
  <OnboardingAvatar v-else-if="currentStep === 'avatar'" />
  <OnboardingCity v-else-if="currentStep === 'city'" />

  <!-- 3) Главный экран -->
  <div v-else class="screen">
    <TopBar :city="selectedCity" @open-city="openCitySheet" @open-sell="sellOpen = true" />

    <div class="panes">
      <div v-show="activeTab === 'catalog'" class="pane">
        <CategoryChips
          :categories="categories"
          :active="activeCategory"
          @select="activeCategory = $event"
        />
        <BouquetGrid
          ref="gridRef"
          :category="activeCategory"
          :city="selectedCity"
          @scroll="onFeedScroll"
          @offer="openOffer"
          @open="openDetail"
        />
      </div>

      <Deals
        v-show="activeTab === 'deals'"
        ref="dealsRef"
        @deals-updated="onDealsUpdated"
        @switch-tab="selectTab"
      />

      <Profile v-show="activeTab === 'profile'" ref="profileRef" />
    </div>

    <BottomNav
      :active="activeTab"
      :collapsed="navCollapsed"
      :deals-badge="dealsActionable"
      @select="selectTab"
    />

    <CitySheet
      :open="cityOpen"
      :current="selectedCity"
      @select="selectCity"
      @close="cityOpen = false"
    />

    <SellSheet
      :open="sellOpen"
      :city="selectedCity"
      @close="sellOpen = false"
      @open-city="openCitySheet"
      @published="onPublished"
    />

    <OfferSheet
      :open="offerOpen"
      :bouquet="offerBouquet"
      @close="offerOpen = false"
      @submitted="onOfferSubmitted"
    />

    <BouquetDetail
      :open="!!detailBouquet"
      :bouquet="detailBouquet"
      @close="closeDetail"
      @offer="openOfferFromDetail"
    />

    <!-- Глобальный стек тоастов поверх всего: SSE-события рендерятся здесь -->
    <Toast />
  </div>
</template>

<style scoped>
.screen {
  position: relative;
  width: 100%;
  height: 100%;
  background: var(--bg);
  color: var(--text);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panes {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.pane {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.boot-splash {
  position: absolute;
  inset: 0;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}
.boot-splash .spinner {
  width: 28px;
  height: 28px;
  border: 2.5px solid var(--border);
  border-top-color: var(--text);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
