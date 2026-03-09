import { ref } from 'vue'
import type { DSN } from '../types'
import * as api from '../api'

const dsns = ref<DSN[]>([])
const isLoading = ref(false)

export function useConnections() {
  async function loadAll() {
    isLoading.value = true
    try {
      const dsnData = await api.listDSNs()
      dsns.value = dsnData.dsns ?? []
    } finally {
      isLoading.value = false
    }
  }

  return {
    dsns,
    isLoading,
    loadAll,
    createDSN: api.createDSN,
    updateDSN: api.updateDSN,
    deleteDSN: api.deleteDSN,
    testDSN: api.testDSN,
  }
}
