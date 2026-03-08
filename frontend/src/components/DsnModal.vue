<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { DSN, DriverName } from '../types'

const props = defineProps<{ editDsn: DSN | null }>()
const emit = defineEmits<{
  save: [{ id?: string; name: string; driver: string; dsn: string; secret?: string }]
  close: []
}>()

const id = ref('')
const name = ref('')
const driver = ref<DriverName | ''>('')
const dsnString = ref('')
const bridgeSecret = ref('')

watch(() => props.editDsn, (d) => {
  if (d) {
    id.value = d.id; name.value = d.name; driver.value = d.driver; dsnString.value = d.dsn
  } else {
    id.value = ''; name.value = ''; driver.value = ''; dsnString.value = ''; bridgeSecret.value = ''
  }
}, { immediate: true })

const isBridge = computed(() => driver.value === 'sqlite-bridge')

const saveDisabled = computed(() => {
  if (!driver.value) return true
  if (isBridge.value) return !bridgeSecret.value
  return !dsnString.value
})

const hints: Record<string, { placeholder: string; format: string; examples: string[] }> = {
  mysql: {
    placeholder: 'user:pass@tcp(host:3306)/dbname',
    format: 'Format: user:password@tcp(host:port)/dbname?param=value',
    examples: [
      'root:secret@tcp(localhost:3306)/mydb',
      'app_user:pass@tcp(db.example.com:3306)/production?charset=utf8mb4&parseTime=true',
    ],
  },
  postgres: {
    placeholder: 'postgres://user:pass@host:5432/dbname',
    format: 'URL: postgres://user:password@host:port/dbname?sslmode=disable  or key=value: host=... user=... password=... dbname=...',
    examples: [
      'postgres://admin:secret@localhost:5432/mydb?sslmode=disable',
      'host=db.example.com port=5432 user=app password=pass dbname=prod sslmode=require',
    ],
  },
  sqlite: {
    placeholder: '/path/to/database.db',
    format: 'Format: absolute or relative file path to the SQLite database',
    examples: ['/data/myapp.db', './local.db'],
  },
}

const hint = computed(() => hints[driver.value] ?? null)
const placeholder = computed(() => hints[driver.value]?.placeholder ?? 'Connection string')

function generateSecret() {
  const arr = new Uint8Array(32)
  crypto.getRandomValues(arr)
  bridgeSecret.value = Array.from(arr).map(b => b.toString(16).padStart(2, '0')).join('')
}

function submit() {
  emit('save', {
    id: id.value || undefined,
    name: name.value,
    driver: driver.value,
    dsn: dsnString.value,
    secret: isBridge.value ? bridgeSecret.value : undefined,
  })
}
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>{{ id ? 'Edit DSN' : 'Add DSN' }}</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>
      <form @submit.prevent="submit">
        <div class="form-group">
          <label>Name</label>
          <input v-model="name" type="text" required placeholder="e.g., mysql-production" />
        </div>
        <div class="form-group">
          <label>Driver</label>
          <select v-model="driver" required>
            <option value="">Select driver...</option>
            <option value="mysql">MySQL</option>
            <option value="postgres">PostgreSQL</option>
            <option value="mssql">SQL Server</option>
            <option value="mongodb">MongoDB</option>
            <option value="redis">Redis</option>
            <option value="etcd">etcd</option>
            <option value="sqlite">SQLite (local file)</option>
            <option value="sqlite-bridge">SQLite Bridge</option>
          </select>
        </div>
        <div v-if="!isBridge" class="form-group">
          <label>Connection String (DSN)</label>
          <input v-model="dsnString" type="text" :placeholder="placeholder" />
        </div>
        <div v-if="isBridge" class="form-group">
          <label>Bridge Secret</label>
          <div style="display: flex; gap: 10px;">
            <input v-model="bridgeSecret" type="text" readonly placeholder="Click Generate to create" />
            <button type="button" class="btn btn-primary" @click="generateSecret">Generate</button>
          </div>
          <small style="color: #666; margin-top: 5px; display: block;">Save this secret to configure the bridge</small>
        </div>
        <div v-if="hint" class="hint-box">
          <div class="hint-format">{{ hint.format }}</div>
          <div class="hint-label">Examples:</div>
          <code v-for="ex in hint.examples" :key="ex" class="hint-example">{{ ex }}</code>
        </div>
        <div class="form-actions">
          <button type="button" class="btn btn-secondary" @click="$emit('close')">Cancel</button>
          <button type="submit" class="btn btn-success" :disabled="saveDisabled">Save</button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}
.modal-content {
  background: white;
  border-radius: 8px;
  padding: 30px;
  width: 100%;
  max-width: 500px;
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.modal-close { background: none; border: none; font-size: 24px; cursor: pointer; color: #999; }
.modal-close:hover { color: #333; }
.hint-box {
  background: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 12px 14px;
  margin-bottom: 12px;
  font-size: 13px;
  color: #555;
}
.hint-format { margin-bottom: 6px; }
.hint-label { color: #888; font-size: 12px; margin-bottom: 4px; }
.hint-example {
  display: block;
  margin: 2px 0;
  color: #c7254e;
  background: #f9f2f4;
  padding: 2px 5px;
  border-radius: 3px;
}
</style>
