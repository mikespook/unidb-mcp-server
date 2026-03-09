import { ref } from 'vue'
import * as api from '../api'

const isAuthenticated = ref(false)
const needsInit = ref(false)
const username = ref('')
const initAdminId = ref('')
const isAdmin = ref(false)

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
      initAdminId.value = me.init_admin_id ?? ''
      isAdmin.value = me.is_admin ?? false
    } catch {
      isAuthenticated.value = false
    }
  }

  async function login(uname: string, password: string): Promise<void> {
    await api.login(uname, password)
    const me = await api.checkAuth()
    isAuthenticated.value = true
    username.value = me.username ?? ''
    initAdminId.value = me.init_admin_id ?? ''
    isAdmin.value = me.is_admin ?? false
  }

  async function logout() {
    await api.logout()
    isAuthenticated.value = false
    username.value = ''
    initAdminId.value = ''
    isAdmin.value = false
  }

  return { isAuthenticated, needsInit, username, initAdminId, isAdmin, checkAuth, login, logout }
}
