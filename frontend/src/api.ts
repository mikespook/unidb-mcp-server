import type { DSN } from './types'

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
    ...init,
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw data
  return data as T
}

// Init
export const getInitStatus = () =>
  apiFetch<{ initialized: boolean }>('/api/ui/init-status')

export const initSetup = (username: string, password: string, jwtSecret?: string) =>
  apiFetch<{ success: boolean; jwt_secret: string; username: string; created_at: string }>('/init', {
    method: 'POST',
    body: JSON.stringify({ username, password, jwt_secret: jwtSecret }),
  })

// Auth
export const checkAuth = () =>
  apiFetch<{ authenticated: boolean; username?: string; init_admin_id?: string }>('/api/ui/me')

export const login = (username: string, password: string) =>
  apiFetch<{ success: boolean }>('/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
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

// Users
export interface User {
  id: string
  username: string
  created_at: string
  updated_at: string
}

export const listUsers = () =>
  apiFetch<{ users: User[] }>('/api/users')

export const createUser = (username: string, password: string) =>
  apiFetch<{ success: boolean; user: User; jwt_secret: string }>('/api/users', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })

export const updateUserPassword = (id: string, password: string) =>
  apiFetch<{ success: boolean }>(`/api/users?id=${id}`, {
    method: 'PUT',
    body: JSON.stringify({ password }),
  })

export const deleteUser = (id: string) =>
  apiFetch<{ success: boolean }>(`/api/users?id=${id}`, { method: 'DELETE' })

export const getUserJWTSecret = (id: string) =>
  apiFetch<{ jwt_secret: string }>(`/api/users/${id}/jwt-secret`)

export const refreshUserJWTSecret = (id: string) =>
  apiFetch<{ jwt_secret: string }>(`/api/users/${id}/jwt-secret`, { method: 'POST' })

// Teams
export interface Team {
  id: string
  name: string
  created_at: string
}

export const listTeams = () =>
  apiFetch<{ teams: Team[] }>('/api/teams')

export const createTeam = (name: string) =>
  apiFetch<{ success: boolean; team: Team }>('/api/teams', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })

export const deleteTeam = (id: string) =>
  apiFetch<{ success: boolean }>(`/api/teams?id=${id}`, { method: 'DELETE' })

export const getTeamUsers = (teamId: string) =>
  apiFetch<{ users: User[] }>(`/api/teams/${teamId}/users`)

export const getTeamDSNs = (teamId: string) =>
  apiFetch<{ dsns: DSN[] }>(`/api/teams/${teamId}/dsns`)

export const addUserToTeam = (teamId: string, userId: string) =>
  apiFetch<{ success: boolean }>(`/api/teams/${teamId}/users`, {
    method: 'POST',
    body: JSON.stringify({ user_id: userId }),
  })

export const removeUserFromTeam = (teamId: string, userId: string) =>
  apiFetch<{ success: boolean }>(`/api/teams/${teamId}/users`, {
    method: 'DELETE',
    body: JSON.stringify({ user_id: userId }),
  })

export const addDSNToTeam = (teamId: string, dsnId: string) =>
  apiFetch<{ success: boolean }>(`/api/teams/${teamId}/dsns`, {
    method: 'POST',
    body: JSON.stringify({ dsn_id: dsnId }),
  })

export const removeDSNFromTeam = (teamId: string, dsnId: string) =>
  apiFetch<{ success: boolean }>(`/api/teams/${teamId}/dsns`, {
    method: 'DELETE',
    body: JSON.stringify({ dsn_id: dsnId }),
  })
