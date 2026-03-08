<script setup lang="ts">
import { ref } from 'vue'
import { useAuth } from '../composables/useAuth'

const { login } = useAuth()

const username = ref('')
const password = ref('')
const error = ref('')

async function handleLogin() {
  error.value = ''
  try {
    await login(username.value, password.value)
    username.value = ''
    password.value = ''
  } catch (e: unknown) {
    const err = e as { error?: string }
    error.value = err?.error || 'Login failed'
  }
}
</script>

<template>
  <div class="login-overlay">
    <div class="login-box">
      <h2>🗄️ UniDB MCP</h2>
      <div class="login-error">{{ error }}</div>
      <div class="form-group">
        <label for="login-username">Username</label>
        <input
          id="login-username"
          v-model="username"
          type="text"
          placeholder="Enter username"
          autocomplete="username"
          @keydown.enter="handleLogin"
          autofocus
        />
      </div>
      <div class="form-group">
        <label for="login-password">Password</label>
        <input
          id="login-password"
          v-model="password"
          type="password"
          placeholder="Enter password"
          autocomplete="current-password"
          @keydown.enter="handleLogin"
        />
      </div>
      <button class="btn btn-primary" style="width: 100%" @click="handleLogin">Login</button>
    </div>
  </div>
</template>

<style scoped>
.login-overlay {
  position: fixed;
  inset: 0;
  background: #f5f5f5;
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 2000;
}
.login-box {
  background: white;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  padding: 40px;
  width: 100%;
  max-width: 360px;
}
.login-box h2 {
  margin-bottom: 24px;
  color: #2c3e50;
  text-align: center;
}
.login-error {
  color: #e74c3c;
  font-size: 14px;
  margin-bottom: 12px;
  min-height: 20px;
}
</style>
