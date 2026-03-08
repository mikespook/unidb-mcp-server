<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  driver: string
  dsn: string
}>()

defineEmits<{ close: [] }>()

interface ParsedInfo {
  network: string
  user: string
  hostport: string
  dbname: string
  params: Record<string, string>
}

const info = computed<ParsedInfo>(() => {
  let network = '', user = '', hostport = '', dbname = ''
  const params: Record<string, string> = {}
  try {
    if (props.driver === 'mysql') {
      const m = props.dsn.match(/^([^:@]*)(?::[^@]*)?@([^(]+)\(([^)]*)\)\/([^?]*)(?:\?(.*))?$/)
      if (m) {
        user = m[1]; network = m[2]; hostport = m[3]; dbname = m[4]
        if (m[5]) new URLSearchParams(m[5]).forEach((v, k) => { params[k] = v })
      }
    } else if (props.driver === 'sqlite') {
      network = 'file'; dbname = props.dsn
    } else if (props.driver === 'postgres') {
      if (props.dsn.startsWith('postgres://') || props.dsn.startsWith('postgresql://')) {
        const url = new URL(props.dsn)
        network = 'tcp'; user = url.username
        hostport = url.port ? `${url.hostname}:${url.port}` : url.hostname
        dbname = url.pathname.replace(/^\//, '').split('?')[0]
        url.searchParams.forEach((v, k) => { params[k] = v })
      } else {
        const get = (key: string) => {
          const m = props.dsn.match(new RegExp(`(?:^|\\s)${key}=(?:'([^']*)'|(\\S*))`))
          return m ? (m[1] ?? m[2] ?? '') : ''
        }
        network = 'tcp'; user = get('user')
        const host = get('host') || 'localhost'; const port = get('port') || '5432'
        hostport = `${host}:${port}`; dbname = get('dbname')
        const sslmode = get('sslmode'); if (sslmode) params['sslmode'] = sslmode
      }
    }
  } catch { /* parse failed */ }
  return { network, user, hostport, dbname, params }
})

const fields = computed(() => {
  if (props.driver === 'sqlite') return [['File', info.value.dbname || '—']]
  return [
    ['Network', info.value.network || '—'],
    ['User', info.value.user || '—'],
    ['Host:Port', info.value.hostport || '—'],
    ['DB Name', info.value.dbname || '—'],
  ]
})
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>ℹ️ Connection Info</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>
      <table class="info-table">
        <tbody>
          <tr v-for="[k, v] in fields" :key="k">
            <td class="label-cell">{{ k }}</td>
            <td class="value-cell">{{ v }}</td>
          </tr>
          <template v-if="Object.keys(info.params).length > 0">
            <tr>
              <td colspan="2" class="params-header">Extra Parameters</td>
            </tr>
            <tr v-for="(v, k) in info.params" :key="k">
              <td class="label-cell">{{ k }}</td>
              <td class="value-cell">{{ v }}</td>
            </tr>
          </template>
        </tbody>
      </table>
      <div class="form-actions">
        <button class="btn btn-secondary" @click="$emit('close')">Close</button>
      </div>
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
.info-table { width: 100%; border-collapse: collapse; }
.label-cell { padding: 6px 12px 6px 0; color: #666; white-space: nowrap; }
.value-cell { padding: 6px 0; font-weight: 500; }
.params-header { padding: 10px 0 4px; color: #999; font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; }
</style>
