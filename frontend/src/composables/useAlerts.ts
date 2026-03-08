import { ref } from 'vue'
import type { Alert } from '../types'

const alerts = ref<Alert[]>([])
let nextId = 0

export function useAlerts() {
  function showAlert(message: string, type: 'success' | 'error') {
    const id = ++nextId
    alerts.value.push({ id, message, type })
    setTimeout(() => {
      const idx = alerts.value.findIndex(a => a.id === id)
      if (idx !== -1) alerts.value.splice(idx, 1)
    }, 5000)
  }

  return { alerts, showAlert }
}
