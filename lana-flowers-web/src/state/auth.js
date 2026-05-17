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
// Порядок: name → avatar → city → phone → app.
// Город идёт ПЕРЕД phone: все «in-app»-шаги онбординга (имя/аватар/город)
// проходим в мини-аппе, потом единственный шаг с переключением в TG-бот
// для шары номера. Город до телефона нужен чтобы каталог сразу открылся
// в правильном городе после онбординга, а не на дефолтной Алматы.
export const currentStep = computed(() => {
  if (!me.value) {
    return needsRegistration.value ? 'phone' : 'loading'
  }
  if (!me.value.display_name) return 'name'
  if (!me.value.onboarding_completed) return 'avatar'
  if (!me.value.city) return 'city'
  if (!me.value.is_registered) return 'phone'
  return 'app'
})
