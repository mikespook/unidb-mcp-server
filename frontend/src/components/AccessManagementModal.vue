<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '../api'
import type { Team, User } from '../api'
import type { DSN } from '../types'

const emit = defineEmits<{ close: [] }>()

type View = 'teams' | 'users' | 'dsns'

const view = ref<View>('teams')
const selectedTeam = ref<Team | null>(null)
const teams = ref<Team[]>([])
const allUsers = ref<User[]>([])
const teamUserIds = ref<Set<string>>(new Set())
const allDSNs = ref<DSN[]>([])
const teamDsnIds = ref<Set<string>>(new Set())
const loadingTeams = ref(false)
const loadingDetail = ref(false)
const error = ref('')

onMounted(loadTeams)

async function loadTeams() {
  loadingTeams.value = true
  error.value = ''
  try {
    teams.value = (await api.listTeams()).teams ?? []
  } catch {
    error.value = 'Failed to load teams'
  } finally {
    loadingTeams.value = false
  }
}

async function openManageUsers(team: Team) {
  selectedTeam.value = team
  loadingDetail.value = true
  error.value = ''
  try {
    const [ur, tr] = await Promise.all([api.listUsers(), api.getTeamUsers(team.id)])
    allUsers.value = ur.users ?? []
    teamUserIds.value = new Set((tr.users ?? []).map((u: User) => u.id))
    view.value = 'users'
  } catch {
    error.value = 'Failed to load users'
  } finally {
    loadingDetail.value = false
  }
}

async function openManageDSNs(team: Team) {
  selectedTeam.value = team
  loadingDetail.value = true
  error.value = ''
  try {
    const [dr, tr] = await Promise.all([api.listDSNs(), api.getTeamDSNs(team.id)])
    allDSNs.value = dr.dsns ?? []
    teamDsnIds.value = new Set((tr.dsns ?? []).map((d: DSN) => d.id))
    view.value = 'dsns'
  } catch {
    error.value = 'Failed to load DSNs'
  } finally {
    loadingDetail.value = false
  }
}

async function toggleUser(userId: string) {
  error.value = ''
  try {
    if (teamUserIds.value.has(userId)) {
      await api.removeUserFromTeam(selectedTeam.value!.id, userId)
      teamUserIds.value = new Set([...teamUserIds.value].filter(id => id !== userId))
    } else {
      await api.addUserToTeam(selectedTeam.value!.id, userId)
      teamUserIds.value = new Set([...teamUserIds.value, userId])
    }
  } catch {
    error.value = 'Operation failed'
  }
}

async function toggleDSN(dsnId: string) {
  error.value = ''
  try {
    if (teamDsnIds.value.has(dsnId)) {
      await api.removeDSNFromTeam(selectedTeam.value!.id, dsnId)
      teamDsnIds.value = new Set([...teamDsnIds.value].filter(id => id !== dsnId))
    } else {
      await api.addDSNToTeam(selectedTeam.value!.id, dsnId)
      teamDsnIds.value = new Set([...teamDsnIds.value, dsnId])
    }
  } catch {
    error.value = 'Operation failed'
  }
}

async function handleCreateTeam() {
  const name = prompt('Team name:')
  if (!name) return
  error.value = ''
  try {
    await api.createTeam(name)
    await loadTeams()
  } catch (e: unknown) {
    const err = e as { error?: string }
    error.value = err?.error || 'Failed to create team'
  }
}

async function handleDeleteTeam(team: Team) {
  if (!confirm(`Delete team "${team.name}"?`)) return
  error.value = ''
  try {
    await api.deleteTeam(team.id)
    await loadTeams()
  } catch (e: unknown) {
    const err = e as { error?: string }
    error.value = err?.error || 'Failed to delete team'
  }
}

function goBack() {
  view.value = 'teams'
  selectedTeam.value = null
  error.value = ''
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal-box">
      <!-- Header -->
      <div class="modal-header">
        <div class="modal-title-row">
          <button v-if="view !== 'teams'" class="btn-back" @click="goBack" title="Back">
            <svg height="18" width="18" viewBox="0 0 24 24" fill="currentColor"><path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/></svg>
          </button>
          <h2 v-if="view === 'teams'">Access Management</h2>
          <h2 v-else-if="view === 'users'">Users in <em>{{ selectedTeam?.name }}</em></h2>
          <h2 v-else>DSNs in <em>{{ selectedTeam?.name }}</em></h2>
        </div>
        <button class="btn-close" @click="emit('close')" title="Close">✕</button>
      </div>

      <!-- Error -->
      <div v-if="error" class="error-msg">{{ error }}</div>

      <!-- Team list view -->
      <div v-if="view === 'teams'">
        <div v-if="loadingTeams" class="loading">Loading…</div>
        <table v-else class="team-table">
          <tbody>
            <tr v-for="team in teams" :key="team.id">
              <td class="team-name">{{ team.name }}</td>
              <td class="team-actions">
                <button class="btn btn-sm" @click="openManageUsers(team)">Manage Users</button>
                <button class="btn btn-sm" @click="openManageDSNs(team)">Manage DSNs</button>
                <button class="btn btn-sm btn-danger" @click="handleDeleteTeam(team)" title="Delete team">✕</button>
              </td>
            </tr>
            <tr v-if="teams.length === 0">
              <td colspan="2" class="empty">No teams found.</td>
            </tr>
          </tbody>
        </table>
        <div class="modal-footer">
          <button class="btn btn-primary" @click="handleCreateTeam">+ New Team</button>
        </div>
      </div>

      <!-- Manage Users view -->
      <div v-else-if="view === 'users'">
        <div v-if="loadingDetail" class="loading">Loading…</div>
        <ul v-else class="member-list">
          <li v-for="user in allUsers" :key="user.id" class="member-item">
            <label>
              <input
                type="checkbox"
                :checked="teamUserIds.has(user.id)"
                @change="toggleUser(user.id)"
              />
              <span class="member-name">{{ user.username }}</span>
              <span class="member-role">{{ user.role }}</span>
            </label>
          </li>
          <li v-if="allUsers.length === 0" class="empty">No users found.</li>
        </ul>
      </div>

      <!-- Manage DSNs view -->
      <div v-else-if="view === 'dsns'">
        <div v-if="loadingDetail" class="loading">Loading…</div>
        <ul v-else class="member-list">
          <li v-for="dsn in allDSNs" :key="dsn.id" class="member-item">
            <label>
              <input
                type="checkbox"
                :checked="teamDsnIds.has(dsn.id)"
                @change="toggleDSN(dsn.id)"
              />
              <span class="member-name">{{ dsn.name }}</span>
              <span class="member-role">{{ dsn.driver }}</span>
            </label>
          </li>
          <li v-if="allDSNs.length === 0" class="empty">No DSNs found.</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}
.modal-box {
  background: white;
  border-radius: 10px;
  width: 100%;
  max-width: 540px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.2);
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 20px 14px;
  border-bottom: 1px solid #eee;
}
.modal-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.modal-header h2 {
  margin: 0;
  font-size: 1.1rem;
  color: #2c3e50;
}
.modal-header h2 em { font-style: normal; color: #0f3460; }
.btn-back {
  background: none;
  border: none;
  cursor: pointer;
  color: #555;
  padding: 2px;
  display: flex;
  align-items: center;
}
.btn-back:hover { color: #0f3460; }
.btn-close {
  background: none;
  border: none;
  font-size: 1rem;
  cursor: pointer;
  color: #888;
  padding: 4px 8px;
  border-radius: 4px;
}
.btn-close:hover { background: #f0f0f0; color: #333; }

.error-msg {
  background: #fff0f0;
  border-bottom: 1px solid #ffcccc;
  color: #c00;
  padding: 8px 20px;
  font-size: 0.9rem;
}

.loading { padding: 24px 20px; color: #888; text-align: center; }

.team-table {
  width: 100%;
  border-collapse: collapse;
  overflow-y: auto;
}
.team-table tr { border-bottom: 1px solid #f0f0f0; }
.team-table tr:last-child { border-bottom: none; }
.team-name {
  padding: 12px 20px;
  font-weight: 600;
  color: #2c3e50;
}
.team-actions {
  padding: 8px 20px 8px 0;
  text-align: right;
  white-space: nowrap;
  display: flex;
  gap: 6px;
  justify-content: flex-end;
  align-items: center;
}
.empty { padding: 20px; color: #aaa; text-align: center; }

.btn { padding: 6px 14px; border-radius: 5px; font-size: 0.85rem; cursor: pointer; border: 1px solid #ddd; background: white; color: #333; }
.btn:hover { background: #f5f5f5; }
.btn-sm { padding: 5px 10px; font-size: 0.8rem; }
.btn-primary { background: #0f3460; color: white; border-color: #0f3460; font-weight: 600; }
.btn-primary:hover { background: #1a4a80; }
.btn-danger { color: #c00; border-color: #ffcccc; }
.btn-danger:hover { background: #fff0f0; }

.modal-footer {
  padding: 12px 20px;
  border-top: 1px solid #eee;
  display: flex;
  justify-content: flex-end;
}

.member-list {
  list-style: none;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  max-height: calc(80vh - 120px);
}
.member-item {
  border-bottom: 1px solid #f5f5f5;
}
.member-item:last-child { border-bottom: none; }
.member-item label {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  cursor: pointer;
}
.member-item label:hover { background: #fafafa; }
.member-item input[type="checkbox"] { width: 16px; height: 16px; cursor: pointer; }
.member-name { flex: 1; font-weight: 500; color: #2c3e50; }
.member-role { font-size: 0.8rem; color: #888; background: #f0f0f0; padding: 2px 8px; border-radius: 10px; }
</style>
