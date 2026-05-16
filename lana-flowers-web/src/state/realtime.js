import { ref } from 'vue'

// sseConnected — глобальный флаг состояния SSE-стрима. App.vue
// устанавливает true/false по событиям useEventStream. Polling-composables
// проверяют его и пропускают тики пока true (real-time апдейты приходят
// без задержки через SSE → polling избыточен).
//
// Если SSE упадёт → флаг false → polling возобновится как safety net.
export const sseConnected = ref(false)
