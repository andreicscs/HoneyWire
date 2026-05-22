import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'

export const useAppStore = defineStore('app', () => {
  // --- STATE ---
  const isArmed = ref(true)
  const version = ref('1.0.0')
  const viewingArchive = ref(false)
  const sidebarOpen = ref(true)
  const currentView = ref('dashboard')
  const activeTimeframe = ref('24H')
  const velocityTimeframe = ref('24H')

  // Auth/setup state (new, not used by old components)
  const isAuthenticated = ref(false)
  const requiresSetup = ref(false)
  const isInitialized = ref(false)

  // --- ACTIONS: UI ---

  const toggleArmed = async () => {
    const next = !isArmed.value
    const previous = isArmed.value

    try {
      const response = await fetch('/api/v1/system/state', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_armed: next }),
      })
      if (!response.ok) throw new Error(`Server error: ${response.status}`)
      isArmed.value = next
    } catch (err) {
      console.error('Failed to toggle armed state:', err)
      isArmed.value = previous
      alert(`Failed to ${next ? 'arm' : 'disarm'} system. Please try again.`)
    }
  }

  const toggleTheme = () => {
    const html = document.documentElement
    if (html.classList.contains('dark')) {
      html.classList.remove('dark')
      localStorage.setItem('theme', 'light')
    } else {
      html.classList.add('dark')
      localStorage.setItem('theme', 'dark')
    }
  }

  const toggleSidebar = () => { sidebarOpen.value = !sidebarOpen.value }
  const setView = (view) => { currentView.value = view }
  const toggleArchive = () => { viewingArchive.value = !viewingArchive.value }

  // --- ACTIONS: AUTH ---

  const login = async (password) => {
    try {
      await api.post('/login', { password })
      // Do NOT set isAuthenticated here.
      // App.vue controls when to reveal the authenticated shell
      // (after loadAppData completes), so the Login component
      // stays mounted long enough to emit 'login-success'.
      return { success: true }
    } catch (err) {
      isAuthenticated.value = false
      return { success: false, status: err.status || 0 }
    }
  }

  const logout = async () => {
    try {
      await fetch('/logout', { method: 'POST' })
      window.location.href = '/'
    } catch (err) {
      console.error('Logout failed', err)
    }
  }

  // --- ACTIONS: SETUP ---

  const completeSetup = async (password, hubEndpoint) => {
    try {
      await api.post('/api/v1/setup', {
        password,
        hub_endpoint: hubEndpoint,
      })
      requiresSetup.value = false
      isAuthenticated.value = true
      return { success: true }
    } catch (err) {
      return { success: false, error: err.message }
    }
  }

  // --- ACTIONS: SECURITY ---

  const changePassword = async (currentPassword, newPassword) => {
    try {
      await api.patch('/api/v1/system/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      return { success: true }
    } catch (err) {
      if (err.status === 401) {
        return { success: false, error: 'Incorrect current password.' }
      }
      return { success: false, error: err.message || 'Failed to update password.' }
    }
  }

  const factoryReset = async (password) => {
    try {
      await api.request('/api/v1/system/reset', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      })
      return { success: true }
    } catch (err) {
      if (err.status === 401) {
        return { success: false, error: 'Incorrect password.' }
      }
      return { success: false, error: err.message || 'Factory reset failed.' }
    }
  }

  // --- ACTIONS: BOOTSTRAP ---

  const checkSetupStatus = async () => {
    try {
      const [stateRes, verRes] = await Promise.all([
        fetch('/api/v1/system/state').then(r => r.json()).catch(() => ({ is_armed: isArmed.value })),
        fetch('/api/v1/version').then(r => r.json()).catch(() => ({ version: version.value })),
      ])

      isArmed.value = stateRes.is_armed !== undefined ? stateRes.is_armed : isArmed.value
      version.value = verRes.version || version.value
    } catch (e) {
      console.error('Failed to fetch system status', e)
    }
  }

  const checkSystemState = async () => {
    try {
      await api.get('/api/v1/system/state')
      return true
    } catch (e) {
      return false
    }
  }

  const checkRequiresSetup = async () => {
    try {
      const res = await api.get('/api/v1/setup/status')
      const data = await res.json()
      return data.requires_setup || false
    } catch (e) {
      console.error('Failed to check setup status:', e)
      return false
    }
  }

  return {
    // State — ALL original properties preserved
    isArmed,
    version,
    viewingArchive,
    sidebarOpen,
    currentView,
    activeTimeframe,
    velocityTimeframe,

    // Auth/setup state (new)
    isAuthenticated,
    requiresSetup,
    isInitialized,

    // Actions: UI (original signatures preserved)
    toggleArmed,
    toggleTheme,
    toggleSidebar,
    setView,
    toggleArchive,

    // Actions: auth (new)
    login,
    logout,

    // Actions: setup (new)
    completeSetup,

    // Actions: security (new)
    changePassword,
    factoryReset,

    // Actions: bootstrap (original + new)
    checkSetupStatus,
    checkSystemState,
    checkRequiresSetup,
  }
})