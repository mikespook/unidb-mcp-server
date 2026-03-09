<script setup lang="ts">
import type { DSN } from '../types'

defineProps<{
  dsns: DSN[]
  isLoading: boolean
}>()

const emit = defineEmits<{
  testDsn: [id: string, name: string, btn: HTMLButtonElement]
  editDsn: [dsn: DSN]
  infoDsn: [dsn: DSN]
  deleteDsn: [id: string]
  addDsn: []
}>()

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
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

    <div v-else-if="dsns.length === 0" class="empty-state">
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
        <tr v-for="dsn in dsns" :key="dsn.id">
          <td>{{ dsn.name }}</td>
          <td><span :class="`badge badge-${dsn.driver}`">{{ dsn.driver }}</span></td>
          <td>{{ formatDate(dsn.created_at) }}</td>
          <td>
            <button
              class="btn btn-primary icon-btn"
              title="Info / Setup tips"
              @click="$emit('infoDsn', dsn)"
            >ℹ️</button>
            <button
              v-if="dsn.driver === 'sqlite-bridge'"
              class="icon-btn"
              style="background: none; border: none; cursor: default;"
              :title="dsn.connected ? 'Connected' : 'Disconnected'"
              disabled
            >{{ dsn.connected ? '🟩' : '🟥' }}</button>
            <button
              v-if="dsn.driver !== 'sqlite-bridge'"
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
