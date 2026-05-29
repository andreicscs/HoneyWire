import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../../api/client'

export const useAppStore = defineStore('app', () => {
  // ==========================================
  // 1. BACKEND-SYNCED STATE (Server Truth)
  // ==========================================
  const isArmed = ref(true)
  const version = ref('1.0.0')

  // ==========================================
  // 2. PURE UI STATE (Frontend Owned)
  // ==========================================
  const viewingArchive = ref(false)
  const sidebarOpen = ref(true)
  const currentView = ref('dashboard')
  const activeTimeframe = ref('24H')
  const velocityTimeframe = ref('24H')

  // ==========================================
  // 3. SESSION STATE MACHINE
  // ==========================================
  // States: 'unknown' -> 'authenticated' | 'unauthenticated'
  const sessionState = ref('unknown') 
  const authError = ref(null)
  const setupError = ref(null)

  // ------------------------------------------
  // THE SESSION TRANSITION AUTHORITY
  // ------------------------------------------
  const transitionSession = (nextState, reason = 'Implicit') => {
    // Prevent redundant transitions
    if (sessionState.value === nextState) return

    // Prevent a lagging bootstrap from overwriting a successful login
    if (sessionState.value === 'authenticated' && nextState === 'unknown') {
      console.warn(`[AppStore] Blocked invalid session transition: authenticated -> unknown`)
      return
    }
        
    sessionState.value = nextState

    // Centralized side-effects for being logged out
    if (nextState === 'unauthenticated') {
      // (Optional: clear sensitive stores here if needed)
    }
  }

  // Computed alias for UI convenience (read-only)
  const isAuthenticated = computed(() => sessionState.value === 'authenticated')

  // ==========================================
  // 4. BOOTSTRAP LIFECYCLE
  // ==========================================
  const requiresSetup = ref(false)
  const isInitialized = ref(false) 
  const bootstrapError = ref(null)

  // --- INGESTION GATEKEEPERS ---
  
  const commitSystemState = (payload) => {
    if (payload?.is_armed !== undefined) isArmed.value = payload.is_armed
  }

  const commitVersionState = (payload) => {
    if (payload?.version !== undefined) version.value = payload.version
  }

  // Pure data-fetching (now defensively programmed)
  const fetchSystemState = async () => {
    try {
      const res = await api.get('/api/v1/system/state')
      const data = await res.json()
      commitSystemState(data)
      return { success: true }
    } catch (err) {
      console.error('Failed to fetch system state:', err)
      
      // Delegation of Authority: 
      // The fetcher doesn't decide auth state, it reports a 401 to the Gatekeeper.
      if (err.status === 401 || err.status === 403) {
        transitionSession('unauthenticated', 'System fetch received 401/403')
      }
      
      return { success: false, status: err.status }
    }
  }

  // --- ACTIONS: SYSTEM CONTROL (Dangerous Ops) ---
  
  const toggleArmed = async () => {
    const targetState = !isArmed.value
    
    // Note: We deliberately do NOT optimistically update `isArmed.value` here.
    // We let the UI use a transitional loading state (e.g., `isArming`) if desired.
    
    try {
      // 1. Dispatch intent
      await api.patch('/api/v1/system/state', { is_armed: targetState })
      
      // 2. Reconcile reality (Catches clamping, delays, or auto-reverts)
      await fetchSystemState()
      
      return { success: true }
    } catch (err) {
      console.error('Failed to toggle armed state:', err)
      // Reality check fallback
      await fetchSystemState()
      return { success: false }
    }
  }

  const changePassword = async (currentPassword, newPassword) => {
    try {
      await api.patch('/api/v1/system/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      return { success: true }
    } catch (err) {
      return { 
        success: false, 
        error: err.status === 401 ? 'Incorrect current password.' : (err.message || 'Failed to update password.')
      }
    }
  }

  const factoryReset = async (password) => {
    try {
      // Standardized to api.post
      await api.post('/api/v1/system/reset', { password })
      return { success: true }
    } catch (err) {
      if (err.status === 401) {
        return { success: false, error: 'Incorrect password.' }
      }
      return { success: false, error: err.message || 'Factory reset failed.' }
    }
  }

  // --- ACTIONS: AUTH & SETUP ---
  
  const login = async (password) => {
    authError.value = null // Clear previous errors
    try {
      await api.post('/login', { password })
      transitionSession('authenticated', 'Login successful')
      return { success: true }
    } catch (err) {
      transitionSession('unauthenticated', 'Login failed')
      authError.value = 'Invalid credentials'
      return { success: false }
    }
  }

  const logout = async () => {
    try {
      await api.post('/logout') 
    } catch (err) {
      console.error('Logout request failed', err)
    } finally {
      transitionSession('unauthenticated', 'Explicit logout')
      window.location.href = '/'
    }
  }

  const completeSetup = async (password, hubEndpoint) => {
    setupError.value = null
    try {
      await api.post('/api/v1/setup', { password, hub_endpoint: hubEndpoint })
      requiresSetup.value = false
      transitionSession('authenticated', 'Setup completed')
      return { success: true }
    } catch (err) {
      setupError.value = err.message || 'Setup failed'
      return { success: false }
    }
  }

  // --- ACTIONS: BOOTSTRAP ---
  
  const checkRequiresSetup = async () => {
    const res = await api.get('/api/v1/setup/status')
    const data = await res.json()
    requiresSetup.value = data.requires_setup || false
  }

  const checkCoreConfiguration = async () => {
    // Independent fetching. If one fails, the other can still succeed.
    const [stateRes, verRes] = await Promise.allSettled([
      api.get('/api/v1/system/state').then(r => r.json()),
      api.get('/api/v1/version').then(r => r.json())
    ])

    if (verRes.status === 'fulfilled') commitVersionState(verRes.value)

    if (stateRes.status === 'fulfilled') {
      commitSystemState(stateRes.value)
      
      // Only transition if we are still in the initial 'unknown' state.
      // If the user already logged in concurrently, the Gatekeeper protects it.
      if (sessionState.value === 'unknown') {
        transitionSession('authenticated', 'Bootstrap fetched protected data successfully')
      }
    } else {
      const isAuthError = stateRes.reason?.status === 401 || stateRes.reason?.status === 403
      if (isAuthError) {
        transitionSession('unauthenticated', 'Bootstrap rejected by 401/403')
      } else {
        // It's a genuine network or server failure, bubble it up to bootstrapError
        throw new Error('Failed to retrieve core system configuration.')
      }
    }
  }

  // SINGLE ORCHESTRATOR
  const initAppStore = async () => {
    bootstrapError.value = null
    
    try {
      await checkRequiresSetup()
      if (!requiresSetup.value) {
        await checkCoreConfiguration()
      }
    } catch (err) {
      console.error('App bootstrap failed:', err)
      bootstrapError.value = err.message || 'Failed to initialize application'
    } finally {
      // If the bootstrap finishes and we somehow still have an 'unknown' session,
      // it means the backend is reachable but setup is required, or something else failed safely.
      if (sessionState.value === 'unknown' && requiresSetup.value) {
        transitionSession('unauthenticated', 'Setup required')
      }
      isInitialized.value = true 
    }
  }

  // --- ACTIONS: UI (Pure Synchronous) ---
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

  return {
    // State
    isArmed, version, viewingArchive, sidebarOpen, currentView, 
    activeTimeframe, velocityTimeframe, sessionState, isAuthenticated, 
    requiresSetup, isInitialized, 
    
    // Errors
    bootstrapError, authError, setupError,

    // UI
    toggleTheme, toggleSidebar, setView, toggleArchive,

    // Async & Orchestration
    toggleArmed, changePassword, factoryReset, login, logout, completeSetup,
    initAppStore, fetchSystemState
  }
})