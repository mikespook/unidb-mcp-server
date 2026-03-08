import type { DSN, Bridge } from './types'

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
    ...init,
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw data
  return data as T
}

// Auth
export const checkAuth = () =>
  apiFetch<{ authenticated: boolean }>('/api/ui/me')

export const login = (password: string) =>
  apiFetch<{ success: boolean }>('/login', {
    method: 'POST',
    body: JSON.stringify({ password }),
  })

export const logout = () =>
  apiFetch<{ success: boolean }>('/logout', { method: 'POST' })

export const changePassword = (current: string, newPass: string) =>
  apiFetch<{ success: boolean }>('/api/ui/password', {
    method: 'POST',
    body: JSON.stringify({ current, new: newPass }),
  })

// DSNs
export const listDSNs = () =>
  apiFetch<{ dsns: DSN[] }>('/api/dsns')

export const createDSN = (name: string, driver: string, dsn: string) =>
  apiFetch<{ success: boolean; dsn: DSN }>('/api/dsns', {
    method: 'POST',
    body: JSON.stringify({ name, driver, dsn }),
  })

export const updateDSN = (id: string, name: string, driver: string, dsn: string) =>
  apiFetch<{ success: boolean; dsn: DSN }>(`/api/dsns?id=${id}`, {
    method: 'PUT',
    body: JSON.stringify({ name, driver, dsn }),
  })

export const deleteDSN = (id: string) =>
  apiFetch<{ success: boolean }>(`/api/dsns?id=${id}`, { method: 'DELETE' })

export const testDSN = (id: string) =>
  apiFetch<{ success: boolean; duration?: string; error?: string }>(`/api/dsns/${id}/test`, {
    method: 'POST',
  })

// Bridges
export const listBridges = () =>
  apiFetch<{ bridges: Bridge[] }>('/api/bridges')

export const registerBridge = (name: string, secret: string, type: string) =>
  apiFetch<{ success: boolean; bridge: Bridge }>('/api/bridges/register', {
    method: 'POST',
    body: JSON.stringify({ name, secret, type }),
  })

export const updateBridge = (oldName: string, name: string, secret: string) =>
  apiFetch<{ success: boolean; bridge: Bridge }>(`/api/bridges?name=${encodeURIComponent(oldName)}`, {
    method: 'PUT',
    body: JSON.stringify({ name, secret }),
  })

export const deleteBridge = (name: string) =>
  apiFetch<{ success: boolean }>(`/api/bridges?name=${encodeURIComponent(name)}`, {
    method: 'DELETE',
  })
