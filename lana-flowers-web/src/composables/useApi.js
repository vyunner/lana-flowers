import { ref, shallowRef } from 'vue'

/**
 * useApi — обёртка над async API-вызовом с loading/error/data состояниями.
 *
 * const { data, loading, error, run, refresh } = useApi(() => listBouquets({ city: 'Алматы' }))
 * onMounted(run)
 */
export function useApi(fn) {
  const data = shallowRef(null)
  const loading = ref(false)
  const error = ref(null)

  async function run(...args) {
    loading.value = true
    error.value = null
    try {
      data.value = await fn(...args)
      return data.value
    } catch (e) {
      error.value = e
      throw e
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, run, refresh: run }
}
