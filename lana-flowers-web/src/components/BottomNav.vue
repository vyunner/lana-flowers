<script setup>
const props = defineProps({
  active: { type: String, required: true },
  collapsed: { type: Boolean, default: false },
  // dealsBadge — сколько pending-офферов ждут ответа от меня. 0 = нет точки.
  dealsBadge: { type: Number, default: 0 },
})
defineEmits(['select'])

const tabs = [
  { key: 'catalog', label: 'Каталог' },
  { key: 'deals', label: 'Сделки' },
  { key: 'profile', label: 'Профиль' },
]

const activeIndex = () => tabs.findIndex((t) => t.key === props.active)
</script>

<template>
  <nav class="navbar" :class="{ collapsed }">
    <span
      class="nav-indicator"
      :style="{ transform: `translateX(${activeIndex() * 100}%)` }"
    ></span>

    <button
      v-for="tab in tabs"
      :key="tab.key"
      type="button"
      class="nav-tab"
      :class="{ active: active === tab.key }"
      @click="$emit('select', tab.key)"
    >
      <!-- icon-wrap — position:relative anchor для бейджа, чтобы он сидел
           точно в правом-верхнем углу ИКОНКИ (а не таба), и не зависел
           от ширины таба на разных экранах. -->
      <span class="icon-wrap">
        <!-- catalog: 4-petal bloom -->
        <svg v-if="tab.key === 'catalog'" viewBox="0 0 28 28" fill="none">
          <g fill="currentColor">
            <ellipse cx="14" cy="7.8" rx="3.4" ry="4.6" />
            <ellipse cx="14" cy="20.2" rx="3.4" ry="4.6" />
            <ellipse cx="7.8" cy="14" rx="4.6" ry="3.4" />
            <ellipse cx="20.2" cy="14" rx="4.6" ry="3.4" />
          </g>
          <circle cx="14" cy="14" r="2.4" :fill="active === 'catalog' ? 'var(--nav-active-bg)' : 'var(--nav-bg)'" />
        </svg>
        <!-- deals — чат/сообщения (пузырь) -->
        <svg v-else-if="tab.key === 'deals'" viewBox="0 0 28 28" fill="none">
          <path
            d="M5 9.5a3.5 3.5 0 013.5-3.5h11A3.5 3.5 0 0123 9.5v7a3.5 3.5 0 01-3.5 3.5H12l-4.5 3.5V20H8.5A3.5 3.5 0 015 16.5v-7z"
            stroke="currentColor"
            stroke-width="1.7"
            stroke-linejoin="round"
          />
        </svg>
        <!-- profile — ВАЖНО: должен быть НЕПОСРЕДСТВЕННО после v-else-if='deals',
             иначе Vue ломает цепочку v-if/v-else-if/v-else и иконка профиля
             рендерится во всех табах одновременно (был баг, ловили вживую). -->
        <svg v-else viewBox="0 0 28 28" fill="none">
          <circle cx="14" cy="10.5" r="4.2" stroke="currentColor" stroke-width="1.7" />
          <path
            d="M5.5 23c1.6-3.8 4.8-5.8 8.5-5.8s6.9 2 8.5 5.8"
            stroke="currentColor"
            stroke-width="1.7"
            stroke-linecap="round"
          />
        </svg>

        <span v-if="tab.key === 'deals' && dealsBadge > 0" class="badge">
          {{ dealsBadge > 9 ? '9+' : dealsBadge }}
        </span>
      </span>
      <span class="label">{{ tab.label }}</span>
    </button>
  </nav>
</template>

<style scoped>
.navbar {
  position: absolute;
  left: 16px;
  right: 16px;
  /* iPhone: ~safe-area + чуть-чуть. Android: фиксированный минимум. */
  bottom: max(12px, calc(env(safe-area-inset-bottom, 0px) + 4px));
  height: 64px;
  background: var(--nav-bg);
  border-radius: var(--radius-pill);
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  align-items: stretch;
  padding: 6px;
  box-shadow: var(--shadow-nav);
  z-index: 6;
  transform-origin: 50% 100%;
  transition:
    transform 0.2s ease-out,
    opacity 0.2s ease-out;
}
.navbar.collapsed {
  transform: scale(0.92);
  opacity: 0.95;
}

.nav-tab {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  border-radius: var(--radius-pill);
  background: transparent;
  border: 0;
  color: var(--nav-inactive-text);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.005em;
  transition:
    color 0.2s ease-out,
    font-weight 0.2s ease-out;
}
.icon-wrap {
  position: relative;
  width: 24px;
  height: 24px;
  display: block;
  flex-shrink: 0;
}
.icon-wrap svg {
  width: 24px;
  height: 24px;
  display: block;
}
.label {
  display: block;
  line-height: 1;
}
.nav-tab.active {
  color: var(--nav-active-text);
  font-weight: 700;
}

.nav-indicator {
  position: absolute;
  top: 6px;
  bottom: 6px;
  left: 6px;
  width: calc((100% - 12px) / 3);
  background: var(--nav-active-bg);
  border-radius: var(--radius-pill);
  z-index: 0;
  transition: transform 0.3s cubic-bezier(0.65, 0, 0.35, 1);
  pointer-events: none;
}

/* Бейдж — красная капля в правом-верхнем углу ИКОНКИ (не таба).
   Сидит в icon-wrap, position:relative anchor, поэтому стабильно
   при любой ширине таба. Чуть «выглядывает» за края иконки —
   отрицательные top/right даёт привычный нативный вид. */
.badge {
  position: absolute;
  top: -6px;
  right: -10px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--accent);
  color: var(--accent-text);
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
  font-variant-numeric: tabular-nums;
  /* белая подложка вокруг чтобы бейдж не сливался с иконкой */
  box-shadow: 0 0 0 2px var(--nav-bg);
  z-index: 2;
}
</style>
