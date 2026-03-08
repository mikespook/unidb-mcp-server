<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  bridgeName: string
  bridgeSecret: string
}>()

defineEmits<{ close: [] }>()

const unidbUrl = computed(() => window.location.origin)
const secretDisplay = computed(() => props.bridgeSecret || '<your-bridge-secret>')

const binaryCmd = computed(() =>
  `unidb-sqlite-bridge \\\n  -name "${props.bridgeName}" \\\n  -file /path/to/your/database.db \\\n  -unidb ${unidbUrl.value} \\\n  -secret "${secretDisplay.value}"`
)

const dockerCmd = computed(() =>
  `docker run \\\n  -v /path/to/your/database.db:/data/sqlite.db:ro \\\n  -e BRIDGE_NAME=${props.bridgeName} \\\n  -e BRIDGE_SECRET=${secretDisplay.value} \\\n  -e UNIDB_URL=${unidbUrl.value} \\\n  unidb-bridge`
)
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>🔧 Bridge Setup</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>
      <p style="margin-bottom: 16px;">Start the SQLite Bridge using one of the following methods:</p>
      <div class="form-group">
        <label>Binary</label>
        <pre class="code-block">{{ binaryCmd }}</pre>
      </div>
      <div class="form-group">
        <label>Docker</label>
        <pre class="code-block">{{ dockerCmd }}</pre>
      </div>
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
.code-block {
  background: #f4f4f4;
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
  font-size: 13px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
