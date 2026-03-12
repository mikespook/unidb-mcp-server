<script setup lang="ts">
import { ref, computed } from 'vue'
import * as api from '../api'
import { useClipboard } from '../composables/useClipboard'

const emit = defineEmits<{ complete: [] }>()
const { copyToClipboard } = useClipboard()

const step = ref(1)
const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)

const jwtSecret = ref('')
const adminUsername = ref('')

function generateSecret(): string {
  const bytes = new Uint8Array(32)
  crypto.getRandomValues(bytes)
  return btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function submitStep1() {
  error.value = ''
  if (!username.value || !password.value) {
    error.value = 'Username and password are required'
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }
  jwtSecret.value = generateSecret()
  adminUsername.value = username.value
  step.value = 2
}

function copySecret() {
  copyToClipboard(jwtSecret.value)
}

function copyMcpExample() {
  copyToClipboard(mcpExample.value)
}

async function finish() {
  error.value = ''
  loading.value = true
  try {
    await api.initSetup(username.value, password.value, jwtSecret.value)
    try { copyToClipboard(jwtSecret.value) } catch { /* best-effort */ }
    emit('complete')
  } catch (e: unknown) {
    const err = e as { error?: string }
    error.value = err?.error || 'Setup failed'
    step.value = 1
  } finally {
    loading.value = false
  }
}

const mcpExample = computed(() => `{
  "mcpServers": {
    "unidb": {
      "url": "${window.location.origin}/api/mcp",
      "headers": {
        "Authorization": "Bearer ${jwtSecret.value}"
      }
    }
  }
}`)

const jwtNote = computed(() => `Your JWT secret is used to sign tokens for MCP client authentication.
Use a tool like jwt.io or the jwt CLI to generate a Bearer token:

  jwt encode --secret "${jwtSecret.value}" '{"sub":"me"}'

Then use it in your MCP client config as shown above.`)
</script>

<template>
  <div class="init-backdrop">
    <div class="init-card">
      <div class="init-header">
        <h1>Welcome to UniDB MCP</h1>
        <p class="init-subtitle">First-time setup</p>
        <div class="steps">
          <span :class="['step', step >= 1 ? 'active' : '']">1. Admin Setup</span>
          <span class="step-sep">›</span>
          <span :class="['step', step >= 2 ? 'active' : '']">2. JWT Secret</span>
          <span class="step-sep">›</span>
          <span :class="['step', step >= 3 ? 'active' : '']">3. Review</span>
        </div>
      </div>

      <!-- Step 1: Admin credentials -->
      <div v-if="step === 1" class="step-content">
        <h2>Create Admin Account</h2>
        <p class="step-desc">Set up the administrator username and password for the web UI.</p>
        <div v-if="error" class="error-msg">{{ error }}</div>
        <div class="form-group">
          <label>Username</label>
          <input v-model="username" type="text" placeholder="admin" autocomplete="username" @keyup.enter="submitStep1" />
        </div>
        <div class="form-group">
          <label>Password</label>
          <input v-model="password" type="password" placeholder="Choose a strong password" autocomplete="new-password" @keyup.enter="submitStep1" />
        </div>
        <div class="form-group">
          <label>Confirm Password</label>
          <input v-model="confirmPassword" type="password" placeholder="Repeat password" autocomplete="new-password" @keyup.enter="submitStep1" />
        </div>
        <div class="form-actions">
          <button class="btn btn-primary" :disabled="loading" @click="submitStep1">
            {{ loading ? 'Creating...' : 'Create Admin & Continue' }}
          </button>
        </div>
      </div>

      <!-- Step 2: JWT Secret -->
      <div v-if="step === 2" class="step-content">
        <h2>Your MCP JWT Secret</h2>
        <p class="step-desc">
          This secret is used to sign Bearer tokens for MCP client authentication.
          <strong>Save it now</strong> — it will not be shown again in plaintext.
        </p>
        <div class="secret-box">
          <code>{{ jwtSecret }}</code>
          <button class="btn-copy" @click="copySecret" title="Copy to clipboard">Copy</button>
        </div>
        <div class="info-box">
          <pre>{{ jwtNote }}</pre>
        </div>
        <div class="code-box-header">
          <h3>Claude Desktop Config Example</h3>
          <button class="btn-copy" @click="copyMcpExample" title="Copy to clipboard">Copy</button>
        </div>
        <div class="code-box">
          <pre>{{ mcpExample }}</pre>
        </div>
        <div class="form-actions">
          <button class="btn btn-primary" @click="step = 3">I've saved it — Continue</button>
        </div>
      </div>

      <!-- Step 3: Review -->
      <div v-if="step === 3" class="step-content">
        <h2>Setup Complete</h2>
        <p class="step-desc">Here's a summary of what was configured:</p>
        <div class="review-list">
          <div class="review-item">
            <span class="review-label">Admin username</span>
            <span class="review-value">{{ adminUsername }}</span>
          </div>
          <div class="review-item">
            <span class="review-label">JWT secret</span>
            <span class="review-value review-masked">{{ jwtSecret.slice(0, 8) }}••••••••</span>
          </div>
          <div class="review-item">
            <span class="review-label">Default team</span>
            <span class="review-value">default (created)</span>
          </div>
        </div>
        <div v-if="error" class="error-msg">{{ error }}</div>
        <div class="form-actions">
          <button class="btn btn-primary" :disabled="loading" @click="finish">
            {{ loading ? 'Setting up...' : 'Start Using UniDB' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.init-backdrop {
  position: fixed;
  inset: 0;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}
.init-card {
  background: white;
  border-radius: 12px;
  padding: 40px;
  width: 100%;
  max-width: 600px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.4);
}
.init-header {
  text-align: center;
  margin-bottom: 32px;
}
.init-header h1 {
  margin: 0 0 4px;
  font-size: 1.8rem;
  color: #1a1a2e;
}
.init-subtitle {
  color: #666;
  margin: 0 0 20px;
}
.steps {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 0.85rem;
}
.step { color: #aaa; font-weight: 500; }
.step.active { color: #0f3460; font-weight: 700; }
.step-sep { color: #ccc; }

.step-content h2 { margin: 0 0 8px; color: #1a1a2e; }
.step-desc { color: #555; margin: 0 0 24px; line-height: 1.5; }

.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-weight: 600; margin-bottom: 6px; color: #333; font-size: 0.9rem; }
.form-group input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 1rem;
  box-sizing: border-box;
}
.form-group input:focus { outline: none; border-color: #0f3460; box-shadow: 0 0 0 2px rgba(15,52,96,0.15); }

.error-msg {
  background: #fff0f0;
  border: 1px solid #ffcccc;
  color: #c00;
  padding: 10px 14px;
  border-radius: 6px;
  margin-bottom: 16px;
  font-size: 0.9rem;
}

.form-actions { margin-top: 24px; display: flex; justify-content: flex-end; }

.btn { padding: 10px 22px; border-radius: 6px; font-size: 1rem; cursor: pointer; border: none; }
.btn-primary { background: #0f3460; color: white; font-weight: 600; }
.btn-primary:hover:not(:disabled) { background: #1a4a80; }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

.secret-box {
  display: flex;
  align-items: center;
  gap: 10px;
  background: #f5f5f5;
  border: 1px solid #ddd;
  border-radius: 6px;
  padding: 12px 16px;
  margin-bottom: 20px;
  word-break: break-all;
}
.secret-box code { flex: 1; font-family: monospace; font-size: 0.95rem; color: #1a1a2e; }
.btn-copy {
  background: #0f3460;
  color: white;
  border: none;
  border-radius: 4px;
  padding: 6px 12px;
  font-size: 0.85rem;
  cursor: pointer;
  white-space: nowrap;
}
.btn-copy:hover { background: #1a4a80; }

.info-box {
  background: #fffbe6;
  border: 1px solid #ffe58f;
  border-radius: 6px;
  padding: 14px 16px;
  margin-bottom: 20px;
}
.info-box pre { margin: 0; font-size: 0.82rem; white-space: pre-wrap; color: #5a4000; }

h3 { font-size: 1rem; margin: 0; color: #333; }
.code-box-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.code-box {
  background: #1a1a2e;
  border-radius: 6px;
  padding: 14px 16px;
  margin-bottom: 20px;
  overflow: hidden;
}
.code-box pre { margin: 0; color: #e0e0e0; font-size: 0.85rem; white-space: pre; overflow: hidden; text-overflow: ellipsis; }

.review-list { border: 1px solid #eee; border-radius: 8px; overflow: hidden; margin-bottom: 24px; }
.review-item { display: flex; padding: 14px 16px; border-bottom: 1px solid #eee; }
.review-item:last-child { border-bottom: none; }
.review-label { color: #666; font-size: 0.9rem; width: 160px; flex-shrink: 0; }
.review-value { font-weight: 600; color: #1a1a2e; word-break: break-all; }
.review-masked { font-family: monospace; letter-spacing: 1px; }
</style>
