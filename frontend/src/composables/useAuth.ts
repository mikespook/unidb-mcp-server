import { ref } from 'vue'
import * as api from '../api'

const isAuthenticated = ref(false)

export function useAuth() {
  async function checkAuth() {
    try {
      await api.checkAuth()
      isAuthenticated.value = true
    } catch {
      isAuthenticated.value = false
    }
  }

  async function login(password: string): Promise<void> {
    await api.login(password)
    isAuthenticated.value = true
  }

  async function logout() {
    await api.logout()
    isAuthenticated.value = false
  }

  return { isAuthenticated, checkAuth, login, logout }
}
