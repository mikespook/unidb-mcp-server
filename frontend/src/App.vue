<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useAuth } from './composables/useAuth'
import { useConnections } from './composables/useConnections'
import { useAlerts } from './composables/useAlerts'
import type { DSN, Bridge } from './types'

import InitWizard from './components/InitWizard.vue'
import LoginScreen from './components/LoginScreen.vue'
import AppHeader from './components/AppHeader.vue'
import ConnectionsTable from './components/ConnectionsTable.vue'
import DsnModal from './components/DsnModal.vue'
import DsnInfoModal from './components/DsnInfoModal.vue'
import BridgeEditModal from './components/BridgeEditModal.vue'
import BridgeTipsModal from './components/BridgeTipsModal.vue'
import ChangePasswordModal from './components/ChangePasswordModal.vue'
import AccessManagementModal from './components/AccessManagementModal.vue'

const { isAuthenticated, needsInit, checkAuth, logout } = useAuth()

async function onInitComplete() {
  await checkAuth()
}
const { dsns, bridges, isLoading, loadAll, createDSN, updateDSN, deleteDSN, testDSN, registerBridge, updateBridge, deleteBridge } = useConnections()
const { alerts, showAlert } = useAlerts()

// Modal state
const showDsnModal = ref(false)
const editingDsn = ref<DSN | null>(null)

const showInfoModal = ref(false)
const infoDriver = ref('')
const infoDsn = ref('')

const showBridgeEditModal = ref(false)
const editingBridge = ref<Bridge | null>(null)

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
    if (payload.driver === 'sqlite-bridge') {
      await registerBridge(payload.name, payload.secret!, 'sqlite')
      showDsnModal.value = false
      await loadAll()
      openBridgeTips(payload.name, payload.secret!)
    } else if (payload.id) {
      await updateDSN(payload.id, payload.name, payload.driver, payload.dsn)
      showAlert('DSN updated successfully', 'success')
      showDsnModal.value = false
      loadAll()
    } else {
      await createDSN(payload.name, payload.driver, payload.dsn)
      showAlert('DSN added successfully', 'success')
      showDsnModal.value = false
      loadAll()
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

// Info modal
function openInfoModal(driver: string, dsn: string) {
  infoDriver.value = driver
  infoDsn.value = dsn
  showInfoModal.value = true
}

// Bridge actions
function openBridgeTips(name: string, secret: string) {
  tipsName.value = name
  tipsSecret.value = secret
  showBridgeTipsModal.value = true
}

function openBridgeEdit(bridge: Bridge) {
  editingBridge.value = bridge
  showBridgeEditModal.value = true
}

async function handleSaveBridge(payload: { oldName: string; name: string; secret: string }) {
  if (!payload.name || !payload.secret) {
    showAlert('Name and secret are required', 'error')
    return
  }
  try {
    await updateBridge(payload.oldName, payload.name, payload.secret)
    showBridgeEditModal.value = false
    await loadAll()
    openBridgeTips(payload.name, payload.secret)
  } catch (e: unknown) {
    const err = e as { error?: string }
    showAlert(err?.error || 'Update failed', 'error')
  }
}

async function handleDeleteBridge(name: string) {
  if (!confirm(`Are you sure you want to delete bridge "${name}"?`)) return
  try {
    await deleteBridge(name)
    showAlert('Bridge deleted successfully', 'success')
    loadAll()
  } catch (e: unknown) {
    const err = e as { error?: string }
    showAlert(err?.error || 'Delete failed', 'error')
  }
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
        :bridges="bridges"
        :is-loading="isLoading"
        @add-dsn="openAddDsn"
        @edit-dsn="openEditDsn"
        @info-dsn="openInfoModal"
        @delete-dsn="handleDeleteDsn"
        @test-dsn="handleTestDsn"
        @edit-bridge="openBridgeEdit"
        @tips-bridge="b => openBridgeTips(b.name, b.secret ?? '')"
        @delete-bridge="handleDeleteBridge"
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
      :dsn="infoDsn"
      @close="showInfoModal = false"
    />

    <BridgeEditModal
      v-if="showBridgeEditModal"
      :bridge="editingBridge"
      @save="handleSaveBridge"
      @close="showBridgeEditModal = false"
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
      @close="showAccessManagementModal = false"
    />
  </template>
</template>
