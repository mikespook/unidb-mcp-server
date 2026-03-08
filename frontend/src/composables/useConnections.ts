import { ref } from 'vue'
import type { DSN, Bridge } from '../types'
import * as api from '../api'

const dsns = ref<DSN[]>([])
const bridges = ref<Bridge[]>([])
const isLoading = ref(false)

export function useConnections() {
  async function loadAll() {
    isLoading.value = true
    try {
      const [dsnData, bridgeData] = await Promise.all([api.listDSNs(), api.listBridges()])
      dsns.value = dsnData.dsns ?? []
      bridges.value = bridgeData.bridges ?? []
    } finally {
      isLoading.value = false
    }
  }

  return {
    dsns,
    bridges,
    isLoading,
    loadAll,
    createDSN: api.createDSN,
    updateDSN: api.updateDSN,
    deleteDSN: api.deleteDSN,
    testDSN: api.testDSN,
    registerBridge: api.registerBridge,
    updateBridge: api.updateBridge,
    deleteBridge: api.deleteBridge,
  }
}
