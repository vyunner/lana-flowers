import { computed, ref } from 'vue'

// Полные данные текущего юзера из /users/me. Null до первой загрузки.
export const me = ref(null)

// Гейт регистрации форс-ит из api/client.js когда любой запрос вернул NOT_REGISTERED.
// Используется как fallback, если кеш `me` не успел обновиться.
export const needsRegistration = ref(false)

export function setMe(u) {
  me.value = u
  needsRegistration.value = !u?.is_registered
}

// Текущий шаг онбординга / основного экрана.
// Порядок: name → avatar → phone → app. Телефон — финальный шаг.
export const currentStep = computed(() => {
  if (!me.value) {
    return needsRegistration.value ? 'phone' : 'loading'
  }
  if (!me.value.display_name) return 'name'
  if (!me.value.onboarding_completed) return 'avatar'
  if (!me.value.is_registered) return 'phone'
  return 'app'
})
