<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Bridge } from '../types'

const props = defineProps<{ bridge: Bridge | null }>()
const emit = defineEmits<{
  save: [{ oldName: string; name: string; secret: string }]
  close: []
}>()

const name = ref('')
const secret = ref('')

watch(() => props.bridge, (b) => {
  name.value = b?.name ?? ''
  secret.value = ''
}, { immediate: true })

function generateSecret() {
  const arr = new Uint8Array(32)
  crypto.getRandomValues(arr)
  secret.value = Array.from(arr).map(b => b.toString(16).padStart(2, '0')).join('')
}

function save() {
  emit('save', { oldName: props.bridge!.name, name: name.value, secret: secret.value })
}
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>Edit Bridge</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>
      <div class="form-group">
        <label>Name</label>
        <input v-model="name" type="text" required placeholder="Bridge name" />
      </div>
      <div class="form-group">
        <label>Secret</label>
        <div style="display: flex; gap: 10px;">
          <input v-model="secret" type="text" required placeholder="Bridge secret" />
          <button type="button" class="btn btn-primary" @click="generateSecret">Regenerate</button>
        </div>
        <small class="warning-text">Changing the secret will disconnect the running bridge until it is restarted with the new secret.</small>
      </div>
      <div class="form-actions">
        <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
        <button class="btn btn-success" @click="save">Save</button>
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
.warning-text { color: #e65100; font-size: 13px; margin-top: 5px; display: block; }
</style>
