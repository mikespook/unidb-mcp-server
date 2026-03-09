<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '../api'
import type { Team, User } from '../api'
import type { DSN } from '../types'

const props = defineProps<{ initAdminId: string }>()
const emit = defineEmits<{ close: [] }>()

// Top-level tab
type Tab = 'teams' | 'users'
const tab = ref<Tab>('teams')

// Team sub-views
type TeamView = 'list' | 'team-users' | 'team-dsns'
const teamView = ref<TeamView>('list')
const selectedTeam = ref<Team | null>(null)
const teams = ref<Team[]>([])
const allUsers = ref<User[]>([])
const teamUserIds = ref<Set<string>>(new Set())
const allDSNs = ref<DSN[]>([])
const teamDsnIds = ref<Set<string>>(new Set())
const loadingTeams = ref(false)
const loadingDetail = ref(false)

// User management
const users = ref<User[]>([])
const userTeams = ref<Record<string, Team[]>>({}) // userId -> teams
const loadingUsers = ref(false)
const userForm = ref<{ username: string; password: string; teamIds: string[] }>({ username: '', password: '', teamIds: [] })
const editingUser = ref<User | null>(null)
const editPassword = ref('')
const showUserForm = ref(false)
const showEditPassword = ref<string | null>(null) // user id being edited
const jwtSecretVisible = ref<Record<string, string>>({}) // userId -> secret

const error = ref('')

onMounted(() => {
  loadTeams()
  loadUsers()
})

// --- Teams ---

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
    teamView.value = 'team-users'
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
    teamView.value = 'team-dsns'
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

function goBackTeams() {
  teamView.value = 'list'
  selectedTeam.value = null
  error.value = ''
}

// --- Users ---

async function loadUsers() {
  loadingUsers.value = true
  error.value = ''
  try {
    const [ur, teamList] = await Promise.all([api.listUsers(), api.listTeams()])
    users.value = ur.users ?? []
    const allTeams = teamList.teams ?? []
    // Fetch membership for all teams in parallel, build userId -> Team[] map
    const teamMemberships = await Promise.all(
      allTeams.map(t => api.getTeamUsers(t.id).then(r => ({ team: t, userIds: (r.users ?? []).map(u => u.id) })))
    )
    const map: Record<string, Team[]> = {}
    for (const { team, userIds } of teamMemberships) {
      for (const uid of userIds) {
        if (!map[uid]) map[uid] = []
        map[uid].push(team)
      }
    }
    userTeams.value = map
  } catch {
    error.value = 'Failed to load users'
  } finally {
    loadingUsers.value = false
  }
}

function openAddUser() {
  userForm.value = { username: '', password: '', teamIds: [] }
  editingUser.value = null
  showUserForm.value = true
  error.value = ''
}

function cancelUserForm() {
  showUserForm.value = false
  error.value = ''
}

async function handleSaveUser() {
  error.value = ''
  if (!userForm.value.username || !userForm.value.password) {
    error.value = 'Username and password are required'
    return
  }
  try {
    const res = await api.createUser(userForm.value.username, userForm.value.password)
    // Assign selected teams
    await Promise.all(userForm.value.teamIds.map(tid => api.addUserToTeam(tid, res.user.id)))
    // Show JWT secret once
    jwtSecretVisible.value = { ...jwtSecretVisible.value, [res.user.id]: res.jwt_secret }
    showUserForm.value = false
    await loadUsers()
  } catch (e: unknown) {
    const err = e as { error?: string }
    error.value = err?.error || 'Failed to create user'
  }
}

function openEditPassword(userId: string) {
  showEditPassword.value = userId
  editPassword.value = ''
  error.value = ''
}

function cancelEditPassword() {
  showEditPassword.value = null
  editPassword.value = ''
  error.value = ''
}

async function handleSavePassword(userId: string) {
  error.value = ''
  if (!editPassword.value) {
    error.value = 'Password is required'
    return
  }
  try {
    await api.updateUserPassword(userId, editPassword.value)
    showEditPassword.value = null
    editPassword.value = ''
  } catch (e: unknown) {
    const err = e as { error?: string }
    error.value = err?.error || 'Failed to update password'
  }
}

async function handleDeleteUser(user: User) {
  if (!confirm(`Delete user "${user.username}"?`)) return
  error.value = ''
  try {
    await api.deleteUser(user.id)
    await loadUsers()
  } catch (e: unknown) {
    const err = e as { error?: string }
    error.value = err?.error || 'Failed to delete user'
  }
}

async function revealJWTSecret(userId: string) {
  if (jwtSecretVisible.value[userId]) {
    const copy = { ...jwtSecretVisible.value }
    delete copy[userId]
    jwtSecretVisible.value = copy
    return
  }
  error.value = ''
  try {
    const res = await api.getUserJWTSecret(userId)
    jwtSecretVisible.value = { ...jwtSecretVisible.value, [userId]: res.jwt_secret }
  } catch {
    error.value = 'Failed to load JWT secret'
  }
}

function switchTab(t: Tab) {
  tab.value = t
  teamView.value = 'list'
  selectedTeam.value = null
  showUserForm.value = false
  showEditPassword.value = null
  error.value = ''
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal-box">
      <!-- Header -->
      <div class="modal-header">
        <div class="modal-title-row">
          <button v-if="teamView !== 'list' && tab === 'teams'" class="btn-back" @click="goBackTeams" title="Back">
            <svg height="18" width="18" viewBox="0 0 24 24" fill="currentColor"><path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/></svg>
          </button>
          <h2 v-if="tab === 'teams' && teamView === 'list'">Access Management</h2>
          <h2 v-else-if="tab === 'teams' && teamView === 'team-users'">Users in <em>{{ selectedTeam?.name }}</em></h2>
          <h2 v-else-if="tab === 'teams' && teamView === 'team-dsns'">DSNs in <em>{{ selectedTeam?.name }}</em></h2>
          <h2 v-else>User Management</h2>
        </div>
        <button class="btn-close" @click="emit('close')" title="Close">✕</button>
      </div>

      <!-- Tab bar (only on top-level views) -->
      <div v-if="teamView === 'list' && !showUserForm && showEditPassword === null" class="tab-bar">
        <button :class="['tab-btn', tab === 'teams' ? 'active' : '']" @click="switchTab('teams')">Teams</button>
        <button :class="['tab-btn', tab === 'users' ? 'active' : '']" @click="switchTab('users')">Users</button>
      </div>

      <!-- Error -->
      <div v-if="error" class="error-msg">{{ error }}</div>

      <!-- ===== TEAMS TAB ===== -->
      <template v-if="tab === 'teams'">
        <!-- Team list -->
        <div v-if="teamView === 'list'" class="scroll-body">
          <div v-if="loadingTeams" class="loading">Loading…</div>
          <table v-else class="team-table">
            <tbody>
              <tr v-for="team in teams" :key="team.id">
                <td class="team-name">{{ team.name }}</td>
                <td class="team-actions">
                  <button class="btn btn-primary icon-btn" title="Manage Users" @click="openManageUsers(team)">👥</button>
                  <button
                    class="btn btn-primary icon-btn"
                    :disabled="team.name === 'admin'"
                    :title="team.name === 'admin' ? 'Admin team has access to all DSNs' : 'Manage DSNs'"
                    @click="openManageDSNs(team)"
                  >🗄️</button>
                  <button
                    class="btn btn-danger icon-btn"
                    :disabled="team.name === 'admin'"
                    :title="team.name === 'admin' ? 'Cannot delete the admin team' : 'Delete team'"
                    @click="handleDeleteTeam(team)"
                  >❌</button>
                </td>
              </tr>
              <tr v-if="teams.length === 0">
                <td colspan="2" class="empty">No teams found.</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Manage team users -->
        <div v-else-if="teamView === 'team-users'" class="scroll-body">
          <div v-if="loadingDetail" class="loading">Loading…</div>
          <ul v-else class="member-list">
            <li v-for="user in allUsers" :key="user.id" class="member-item">
              <label>
                <input type="checkbox" :checked="teamUserIds.has(user.id)" @change="toggleUser(user.id)" />
                <span class="member-name">{{ user.username }}</span>
              </label>
            </li>
            <li v-if="allUsers.length === 0" class="empty">No users found.</li>
          </ul>
        </div>

        <!-- Manage team DSNs -->
        <div v-else-if="teamView === 'team-dsns'" class="scroll-body">
          <div v-if="loadingDetail" class="loading">Loading…</div>
          <ul v-else class="member-list">
            <li v-for="dsn in allDSNs" :key="dsn.id" class="member-item">
              <label>
                <input type="checkbox" :checked="teamDsnIds.has(dsn.id)" @change="toggleDSN(dsn.id)" />
                <span class="member-name">{{ dsn.name }}</span>
                <span class="member-role">{{ dsn.driver }}</span>
              </label>
            </li>
            <li v-if="allDSNs.length === 0" class="empty">No DSNs found.</li>
          </ul>
        </div>

        <div v-if="teamView === 'list'" class="modal-footer">
          <button class="btn btn-primary" @click="handleCreateTeam">+ Add Team</button>
        </div>
      </template>

      <!-- ===== USERS TAB ===== -->
      <template v-else-if="tab === 'users'">
        <!-- Add user form -->
        <div v-if="showUserForm" class="user-form scroll-body">
          <div class="form-group">
            <label>Username</label>
            <input v-model="userForm.username" type="text" placeholder="username" autocomplete="off" @keydown.enter="handleSaveUser" />
          </div>
          <div class="form-group">
            <label>Password</label>
            <input v-model="userForm.password" type="password" placeholder="password" autocomplete="new-password" @keydown.enter="handleSaveUser" />
          </div>
          <div class="form-group">
            <label>Teams</label>
            <div class="team-checkboxes">
              <label v-for="team in teams" :key="team.id" class="team-checkbox-item">
                <input type="checkbox" :value="team.id" v-model="userForm.teamIds" />
                {{ team.name }}
              </label>
              <span v-if="teams.length === 0" class="no-teams">No teams available</span>
            </div>
          </div>
          <div class="form-actions">
            <button class="btn" @click="cancelUserForm">Cancel</button>
            <button class="btn btn-primary" @click="handleSaveUser">Create User</button>
          </div>
        </div>

        <!-- User list -->
        <div v-else class="scroll-body">
          <div v-if="loadingUsers" class="loading">Loading…</div>
          <ul v-else class="member-list">
            <li v-for="user in users" :key="user.id" class="user-item">
              <div class="user-row">
                <div class="user-info">
                  <span class="member-name">{{ user.username }}</span>
                  <span v-for="t in (userTeams[user.id] ?? [])" :key="t.id" class="team-badge">{{ t.name }}</span>
                </div>
                <div class="user-actions">
                  <button class="btn btn-primary icon-btn" @click="revealJWTSecret(user.id)" :title="jwtSecretVisible[user.id] ? 'Hide JWT secret' : 'Show JWT secret'">🔑</button>
                  <button class="btn btn-primary icon-btn" @click="openEditPassword(user.id)" title="Change password">📝</button>
                  <button
                    class="btn btn-danger icon-btn"
                    :disabled="user.id === props.initAdminId"
                    :title="user.id === props.initAdminId ? 'Cannot delete the initial admin user' : 'Delete user'"
                    @click="handleDeleteUser(user)"
                  >❌</button>
                </div>
              </div>
              <!-- Inline password edit -->
              <div v-if="showEditPassword === user.id" class="inline-edit">
                <input
                  v-model="editPassword"
                  type="password"
                  placeholder="New password"
                  autocomplete="new-password"
                  @keydown.enter="handleSavePassword(user.id)"
                  @keydown.esc="cancelEditPassword"
                />
                <button class="btn btn-sm btn-primary" @click="handleSavePassword(user.id)">Save Password</button>
                <button class="btn btn-sm" @click="cancelEditPassword">Cancel</button>
              </div>
              <!-- JWT secret reveal -->
              <div v-if="jwtSecretVisible[user.id]" class="jwt-reveal">
                <code>{{ jwtSecretVisible[user.id] }}</code>
                <button class="btn-copy-sm" @click="() => navigator.clipboard.writeText(jwtSecretVisible[user.id])" title="Copy">Copy</button>
              </div>
            </li>
            <li v-if="users.length === 0" class="empty">No users found.</li>
          </ul>
        </div>

        <div v-if="!showUserForm" class="modal-footer">
          <button class="btn btn-primary" @click="openAddUser">+ Add User</button>
        </div>
      </template>
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
  max-width: 560px;
  max-height: 82vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.2);
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px 12px;
  border-bottom: 1px solid #eee;
  flex-shrink: 0;
}
.modal-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.modal-header h2 { margin: 0; font-size: 1.1rem; color: #2c3e50; }
.modal-header h2 em { font-style: normal; color: #0f3460; }
.btn-back { background: none; border: none; cursor: pointer; color: #555; padding: 2px; display: flex; align-items: center; }
.btn-back:hover { color: #0f3460; }
.btn-close { background: none; border: none; font-size: 1rem; cursor: pointer; color: #888; padding: 4px 8px; border-radius: 4px; }
.btn-close:hover { background: #f0f0f0; color: #333; }

.tab-bar {
  display: flex;
  border-bottom: 1px solid #eee;
  flex-shrink: 0;
}
.tab-btn {
  flex: 1;
  padding: 10px;
  background: none;
  border: none;
  font-size: 0.9rem;
  font-weight: 500;
  color: #888;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.tab-btn:hover { color: #2c3e50; }
.tab-btn.active { color: #0f3460; border-bottom-color: #0f3460; font-weight: 700; }

.error-msg { background: #fff0f0; border-bottom: 1px solid #ffcccc; color: #c00; padding: 8px 20px; font-size: 0.9rem; flex-shrink: 0; }

.scroll-body { overflow-y: auto; flex: 1; }
.loading { padding: 24px 20px; color: #888; text-align: center; }
.empty { padding: 20px; color: #aaa; text-align: center; list-style: none; }

/* Teams */
.team-table { width: 100%; border-collapse: collapse; }
.team-table tr { border-bottom: 1px solid #f0f0f0; }
.team-table tr:last-child { border-bottom: none; }
.team-name { padding: 12px 20px; font-weight: 600; color: #2c3e50; }
.team-actions { padding: 8px 16px 8px 0; display: flex; gap: 6px; justify-content: flex-end; align-items: center; }

/* Members (team-user/dsn checkboxes) */
.member-list { list-style: none; margin: 0; padding: 0; }
.member-item { border-bottom: 1px solid #f5f5f5; }
.member-item:last-child { border-bottom: none; }
.member-item label { display: flex; align-items: center; gap: 10px; padding: 11px 20px; cursor: pointer; }
.member-item label:hover { background: #fafafa; }
.member-item input[type="checkbox"] { width: 16px; height: 16px; cursor: pointer; }
.member-name { flex: 1; font-weight: 500; color: #2c3e50; }
.member-role { font-size: 0.78rem; color: #888; background: #f0f0f0; padding: 2px 8px; border-radius: 10px; }

/* Users tab */
.user-item { border-bottom: 1px solid #f0f0f0; }
.user-item:last-child { border-bottom: none; }
.user-row { display: flex; align-items: center; justify-content: space-between; padding: 10px 16px 10px 20px; }
.user-info { display: flex; align-items: center; gap: 10px; }
.user-actions { display: flex; gap: 6px; }
.inline-edit { display: flex; gap: 6px; align-items: center; padding: 0 16px 10px 20px; }
.inline-edit input { flex: 1; padding: 5px 8px; border: 1px solid #ddd; border-radius: 5px; font-size: 0.85rem; }
.inline-edit input:focus { outline: none; border-color: #0f3460; }
.jwt-reveal { display: flex; align-items: center; gap: 8px; padding: 0 16px 10px 20px; }
.jwt-reveal code { flex: 1; font-size: 0.78rem; font-family: monospace; color: #1a1a2e; word-break: break-all; background: #f5f5f5; padding: 5px 8px; border-radius: 4px; }
.btn-copy-sm { background: #0f3460; color: white; border: none; border-radius: 4px; padding: 4px 10px; font-size: 0.78rem; cursor: pointer; white-space: nowrap; }
.btn-copy-sm:hover { background: #1a4a80; }

/* User form */
.user-form { padding: 20px; }
.form-group { margin-bottom: 14px; }
.form-group label { display: block; font-weight: 600; font-size: 0.85rem; color: #333; margin-bottom: 5px; }
.form-group input, .form-group select { width: 100%; padding: 8px 10px; border: 1px solid #ddd; border-radius: 5px; font-size: 0.9rem; box-sizing: border-box; }
.form-group input:focus, .form-group select:focus { outline: none; border-color: #0f3460; }
.form-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 6px; }
.team-checkboxes { display: flex; flex-direction: column; gap: 6px; border: 1px solid #ddd; border-radius: 5px; padding: 8px 10px; max-height: 120px; overflow-y: auto; }
.team-checkbox-item { display: flex; align-items: center; gap: 8px; font-size: 0.9rem; cursor: pointer; }
.team-checkbox-item input[type="checkbox"] { width: 15px; height: 15px; cursor: pointer; }
.no-teams { font-size: 0.85rem; color: #aaa; }

.team-badge { font-size: 0.72rem; background: #e8f4fd; color: #1976d2; padding: 2px 7px; border-radius: 10px; }

/* Buttons */
.btn-sm { padding: 4px 10px; font-size: 0.8rem; }

.modal-footer { padding: 10px 20px; border-top: 1px solid #eee; display: flex; justify-content: flex-end; flex-shrink: 0; }
</style>
