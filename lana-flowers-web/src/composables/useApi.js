import { ref, shallowRef } from 'vue'

/**
 * useApi — обёртка над async API-вызовом с loading/error/data состояниями.
 *
 * const { data, loading, error, run, refresh } = useApi(() => listBouquets({ city: 'Алматы' }))
 * onMounted(run)
 *
 * Dedup: если run() позвали пока предыдущий ещё в flight — возвращаем
 * тот же promise вместо нового запроса. Иначе при polling+SSE+manual
 * refresh могло уходить 3 запроса параллельно за теми же данными.
 */
export function useApi(fn) {
  const data = shallowRef(null)
  const loading = ref(false)
  const error = ref(null)

  let inflight = null

  async function run(...args) {
    if (inflight) return inflight
    loading.value = true
    error.value = null
    inflight = (async () => {
      try {
        data.value = await fn(...args)
        return data.value
      } catch (e) {
        error.value = e
        throw e
      } finally {
        loading.value = false
        inflight = null
      }
    })()
    return inflight
  }

  return { data, loading, error, run, refresh: run }
}
