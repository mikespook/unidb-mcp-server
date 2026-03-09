<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useAuth } from './composables/useAuth'
import { useConnections } from './composables/useConnections'
import { useAlerts } from './composables/useAlerts'
import type { DSN } from './types'

import InitWizard from './components/InitWizard.vue'
import LoginScreen from './components/LoginScreen.vue'
import AppHeader from './components/AppHeader.vue'
import ConnectionsTable from './components/ConnectionsTable.vue'
import DsnModal from './components/DsnModal.vue'
import DsnInfoModal from './components/DsnInfoModal.vue'
import BridgeTipsModal from './components/BridgeTipsModal.vue'
import ChangePasswordModal from './components/ChangePasswordModal.vue'
import AccessManagementModal from './components/AccessManagementModal.vue'

const { isAuthenticated, needsInit, initAdminId, checkAuth, logout } = useAuth()

async function onInitComplete() {
  await checkAuth()
}
const { dsns, isLoading, loadAll, createDSN, updateDSN, deleteDSN, testDSN } = useConnections()
const { alerts, showAlert } = useAlerts()

// Modal state
const showDsnModal = ref(false)
const editingDsn = ref<DSN | null>(null)

const showInfoModal = ref(false)
const infoDriver = ref('')
const infoDsnStr = ref('')

const showBridgeTipsModal = ref(false)
const tipsName = ref('')
const tipsSecret = ref('')

const showChangePasswordModal = ref(false)
const showAccessManagementModal = ref(false)

onMounted(async () => {
  await checkAuth()
  if (isAuthenticated.value) loadAll()
})

watch(isAuthenticated, (val) => {
  if (val) loadAll()
})

// Auth
async function handleLogout() {
  await logout()
}

// DSN actions
function openAddDsn() {
  editingDsn.value = null
  showDsnModal.value = true
}

function openEditDsn(dsn: DSN) {
  editingDsn.value = dsn
  showDsnModal.value = true
}

async function handleSaveDsn(payload: { id?: string; name: string; driver: string; dsn: string; secret?: string }) {
  try {
    const dsnValue = payload.driver === 'sqlite-bridge' ? payload.secret! : payload.dsn
    if (payload.id) {
      await updateDSN(payload.id, payload.name, payload.driver, dsnValue)
      showDsnModal.value = false
      await loadAll()
      if (payload.driver === 'sqlite-bridge') {
        openBridgeTips(payload.name, payload.secret!)
      } else {
        showAlert('DSN updated successfully', 'success')
      }
    } else {
      await createDSN(payload.name, payload.driver, dsnValue)
      showDsnModal.value = false
      await loadAll()
      if (payload.driver === 'sqlite-bridge') {
        openBridgeTips(payload.name, payload.secret!)
      } else {
        showAlert('DSN added successfully', 'success')
      }
    }
  } catch (e: unknown) {
    const err = e as { error?: string }
    showAlert(err?.error || 'Operation failed', 'error')
  }
}

async function handleDeleteDsn(id: string) {
  if (!confirm('Are you sure you want to delete this DSN?')) return
  try {
    await deleteDSN(id)
    showAlert('DSN deleted successfully', 'success')
    loadAll()
  } catch (e: unknown) {
    const err = e as { error?: string }
    showAlert(err?.error || 'Delete failed', 'error')
  }
}

async function handleTestDsn(id: string, name: string, btn: HTMLButtonElement) {
  const orig = btn.innerHTML
  btn.innerHTML = '⏳'
  btn.disabled = true
  try {
    const result = await testDSN(id)
    if (result.success) {
      showAlert(`<strong>${name}</strong>: Connection successful (${result.duration})`, 'success')
    } else {
      showAlert(`<strong>${name}</strong>: Connection failed: ${result.error}`, 'error')
    }
  } catch {
    showAlert('Test failed', 'error')
  } finally {
    btn.innerHTML = orig
    btn.disabled = false
  }
}

// Info modal — routes to BridgeTipsModal for sqlite-bridge, DsnInfoModal for others
function handleInfoDsn(dsn: DSN) {
  if (dsn.driver === 'sqlite-bridge') {
    openBridgeTips(dsn.name, dsn.dsn)
  } else {
    infoDriver.value = dsn.driver
    infoDsnStr.value = dsn.dsn
    showInfoModal.value = true
  }
}

function openBridgeTips(name: string, secret: string) {
  tipsName.value = name
  tipsSecret.value = secret
  showBridgeTipsModal.value = true
}
</script>

<template>
  <!-- Init wizard: shown on first run before any login -->
  <InitWizard v-if="needsInit" @complete="onInitComplete" />

  <!-- Login screen overlays everything when not authenticated -->
  <LoginScreen v-else-if="!isAuthenticated" />

  <template v-else-if="isAuthenticated">
    <AppHeader
      @change-password="showChangePasswordModal = true"
      @access-management="showAccessManagementModal = true"
      @logout="handleLogout"
    />

    <div class="container">
      <!-- Alerts -->
      <div
        v-for="alert in alerts"
        :key="alert.id"
        :class="`alert alert-${alert.type}`"
        v-html="alert.message"
      />

      <ConnectionsTable
        :dsns="dsns"
        :is-loading="isLoading"
        @add-dsn="openAddDsn"
        @edit-dsn="openEditDsn"
        @info-dsn="handleInfoDsn"
        @delete-dsn="handleDeleteDsn"
        @test-dsn="handleTestDsn"
      />
    </div>

    <!-- Modals -->
    <DsnModal
      v-if="showDsnModal"
      :edit-dsn="editingDsn"
      @save="handleSaveDsn"
      @close="showDsnModal = false"
    />

    <DsnInfoModal
      v-if="showInfoModal"
      :driver="infoDriver"
      :dsn="infoDsnStr"
      @close="showInfoModal = false"
    />

    <BridgeTipsModal
      v-if="showBridgeTipsModal"
      :bridge-name="tipsName"
      :bridge-secret="tipsSecret"
      @close="showBridgeTipsModal = false"
    />

    <ChangePasswordModal
      v-if="showChangePasswordModal"
      @close="showChangePasswordModal = false"
    />

    <AccessManagementModal
      v-if="showAccessManagementModal"
      :init-admin-id="initAdminId"
      @close="showAccessManagementModal = false"
    />
  </template>
</template>
