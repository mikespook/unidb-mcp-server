import { ref } from 'vue'
import * as api from '../api'

const isAuthenticated = ref(false)
const needsInit = ref(false)
const username = ref('')
const role = ref('')

export function useAuth() {
  async function checkAuth() {
    try {
      const status = await api.getInitStatus()
      if (!status.initialized) {
        needsInit.value = true
        isAuthenticated.value = false
        return
      }
      needsInit.value = false
      const me = await api.checkAuth()
      isAuthenticated.value = me.authenticated
      username.value = me.username ?? ''
      role.value = me.role ?? ''
    } catch {
      isAuthenticated.value = false
    }
  }

  async function login(uname: string, password: string): Promise<void> {
    await api.login(uname, password)
    const me = await api.checkAuth()
    isAuthenticated.value = true
    username.value = me.username ?? ''
    role.value = me.role ?? ''
  }

  async function logout() {
    await api.logout()
    isAuthenticated.value = false
    username.value = ''
    role.value = ''
  }

  return { isAuthenticated, needsInit, username, role, checkAuth, login, logout }
}
