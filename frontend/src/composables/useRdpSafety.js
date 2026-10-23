import { ref } from 'vue'
import { api } from '../api/index.js'

// Shared by the global recovery banner and the RDP page. A failed refresh must
// never hide the last pending transaction or enable a stale confirmation.
const status = ref(null)
const error = ref('')
const loading = ref(false)
let request
export function useRdpSafety() {
  function refresh() {
    if (request) return request
    loading.value = true
    request = api
      .getRdpSafetyStatus()
      .then((value) => {
        status.value = value
        error.value = ''
        return value
      })
      .catch((e) => {
        error.value = e?.message || String(e)
        if (status.value) status.value = { ...status.value, canConfirm: false }
      })
      .finally(() => {
        request = null
        loading.value = false
      })
    return request
  }
  return { status, error, loading, refresh }
}
