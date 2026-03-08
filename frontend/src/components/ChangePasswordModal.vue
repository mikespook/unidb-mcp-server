<script setup lang="ts">
import { ref } from 'vue'
import { useAlerts } from '../composables/useAlerts'
import * as api from '../api'

const { showAlert } = useAlerts()

const current = ref('')
const newPass = ref('')
const confirm = ref('')

async function submit() {
  if (!current.value || !newPass.value) {
    showAlert('Current and new password are required', 'error')
    return
  }
  if (newPass.value !== confirm.value) {
    showAlert('New passwords do not match', 'error')
    return
  }
  try {
    await api.changePassword(current.value, newPass.value)
    showAlert('Password updated successfully', 'success')
    current.value = ''
    newPass.value = ''
    confirm.value = ''
    emit('close')
  } catch (e: unknown) {
    const err = e as { error?: string }
    showAlert(err?.error || 'Failed to update password', 'error')
  }
}

const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>Change Password</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>
      <div class="form-group">
        <label>Current Password</label>
        <input v-model="current" type="password" placeholder="Current password" />
      </div>
      <div class="form-group">
        <label>New Password</label>
        <input v-model="newPass" type="password" placeholder="New password" />
      </div>
      <div class="form-group">
        <label>Confirm New Password</label>
        <input v-model="confirm" type="password" placeholder="Confirm new password" />
      </div>
      <div class="form-actions">
        <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
        <button class="btn btn-primary" @click="submit">Update Password</button>
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
.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #999;
}
.modal-close:hover { color: #333; }
</style>
