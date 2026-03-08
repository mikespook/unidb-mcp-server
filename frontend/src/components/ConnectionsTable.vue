<script setup lang="ts">
import type { DSN, Bridge } from '../types'

defineProps<{
  dsns: DSN[]
  bridges: Bridge[]
  isLoading: boolean
}>()

const emit = defineEmits<{
  testDsn: [id: string, name: string, btn: HTMLButtonElement]
  editDsn: [dsn: DSN]
  infoDsn: [driver: string, dsn: string]
  deleteDsn: [id: string]
  editBridge: [bridge: Bridge]
  tipsBridge: [bridge: Bridge]
  deleteBridge: [name: string]
  addDsn: []
}>()

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

function hasInfo(driver: string) {
  return ['mysql', 'postgres', 'sqlite'].includes(driver)
}

async function handleTest(id: string, name: string, event: MouseEvent) {
  const btn = event.currentTarget as HTMLButtonElement
  emit('testDsn', id, name, btn)
}
</script>

<template>
  <div class="card">
    <div class="card-header">
      <h2>Database Connections</h2>
      <button class="btn btn-primary" @click="$emit('addDsn')">+ Add DSN</button>
    </div>

    <div v-if="isLoading" class="loading">Loading...</div>

    <div v-else-if="dsns.length === 0 && bridges.length === 0" class="empty-state">
      No DSNs configured. Click "Add DSN" to get started.
    </div>

    <table v-else>
      <thead>
        <tr>
          <th>Name</th>
          <th>Driver</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <!-- DSN rows -->
        <tr v-for="dsn in dsns" :key="dsn.id">
          <td>{{ dsn.name }}</td>
          <td><span :class="`badge badge-${dsn.driver}`">{{ dsn.driver }}</span></td>
          <td>{{ formatDate(dsn.created_at) }}</td>
          <td>
            <button
              v-if="hasInfo(dsn.driver)"
              class="btn btn-primary icon-btn"
              title="Connection info"
              @click="$emit('infoDsn', dsn.driver, dsn.dsn)"
            >ℹ️</button>
            <button
              class="btn btn-success icon-btn"
              title="Test connection"
              @click="handleTest(dsn.id, dsn.name, $event)"
            >🔍</button>
            <button
              class="btn btn-primary icon-btn"
              title="Edit"
              @click="$emit('editDsn', dsn)"
            >📝</button>
            <button
              class="btn btn-danger icon-btn"
              title="Delete"
              @click="$emit('deleteDsn', dsn.id)"
            >❌</button>
          </td>
        </tr>
        <!-- Bridge rows -->
        <tr v-for="bridge in bridges" :key="bridge.id">
          <td>{{ bridge.name }}</td>
          <td><span class="badge badge-sqlite-bridge">sqlite-bridge</span></td>
          <td>{{ formatDate(bridge.created_at) }}</td>
          <td>
            <button
              class="btn btn-primary icon-btn"
              title="Setup tips"
              @click="$emit('tipsBridge', bridge)"
            >ℹ️</button>
            <button
              class="icon-btn"
              style="background: none; border: none; cursor: default;"
              :title="bridge.connected ? 'Connected' : 'Disconnected'"
              disabled
            >{{ bridge.connected ? '🟩' : '🟥' }}</button>
            <button
              class="btn btn-primary icon-btn"
              title="Edit"
              @click="$emit('editBridge', bridge)"
            >📝</button>
            <button
              class="btn btn-danger icon-btn"
              title="Delete"
              @click="$emit('deleteBridge', bridge.name)"
            >❌</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}
.empty-state {
  text-align: center;
  padding: 40px;
  color: #999;
}
</style>
