import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, ApiError } from '../../api/client'

export type SessionState = 'unknown' | 'authenticated' | 'unauthenticated'

export interface SystemStatePayload {
  isArmed?: boolean
}

export interface VersionPayload {
  version?: string
}

export interface AppState {
  isArmed: boolean
  version: string
  viewingArchive: boolean
  sidebarOpen: boolean
  activeTimeframe: string
  velocityTimeframe: string
  sessionState: SessionState
  authError: string | null
  setupError: string | null
  requiresSetup: boolean
  isInitialized: boolean
  bootstrapError: string | null
}

export const useAppStore = defineStore('app', () => {
  // SINGLE STATE TREE
  const state = ref<AppState>({
    isArmed: true,
    version: '2.0.0',
    viewingArchive: false,
    sidebarOpen: true,
    activeTimeframe: '24H',
    velocityTimeframe: '24H',
    sessionState: 'unknown',
    authError: null,
    setupError: null,
    requiresSetup: false,
    isInitialized: false,
    bootstrapError: null
  })

  // ------------------------------------------
  // THE SESSION TRANSITION AUTHORITY
  // ------------------------------------------
  const transitionSession = (nextState: SessionState): void => {
    if (state.value.sessionState === nextState) return

    const validTransitions: Record<SessionState, SessionState[]> = {
      unknown: ['authenticated', 'unauthenticated'],
      authenticated: ['unauthenticated'],
      unauthenticated: ['authenticated']
    }

    if (!validTransitions[state.value.sessionState].includes(nextState)) {
      console.warn(`[AppStore] Blocked invalid session transition: ${state.value.sessionState} -> ${nextState}`)
      return
    }
        
    state.value.sessionState = nextState
    
    // Safe side-effect: If successfully authenticated, clear the setup wall flag
    if (nextState === 'authenticated') {
      state.value.requiresSetup = false
    }
  }

  // ==========================================
  // ENCAPSULATED GETTERS (Public API)
  // ==========================================
  const isArmed = computed<boolean>(() => state.value.isArmed)
  const version = computed<string>(() => state.value.version)
  const viewingArchive = computed<boolean>(() => state.value.viewingArchive)
  const sidebarOpen = computed<boolean>(() => state.value.sidebarOpen)
  const activeTimeframe = computed<string>(() => state.value.activeTimeframe)
  const velocityTimeframe = computed<string>(() => state.value.velocityTimeframe)
  const sessionState = computed<SessionState>(() => state.value.sessionState)
  const authError = computed<string | null>(() => state.value.authError)
  const setupError = computed<string | null>(() => state.value.setupError)
  const requiresSetup = computed<boolean>(() => state.value.requiresSetup)
  const isInitialized = computed<boolean>(() => state.value.isInitialized)
  const bootstrapError = computed<string | null>(() => state.value.bootstrapError)

  // DERIVED STATE
  const isAuthenticated = computed<boolean>(() => state.value.sessionState === 'authenticated')
  const isReady = computed<boolean>(() => state.value.isInitialized && state.value.bootstrapError === null)
  const isBootstrapping = computed<boolean>(() => !state.value.isInitialized)
  const canAccessDashboard = computed<boolean>(() => isAuthenticated.value && !state.value.requiresSetup)

  // --- INGESTION GATEKEEPERS ---
  
  const commitSystemState = (payload: SystemStatePayload): void => {
    if (payload?.isArmed !== undefined) state.value.isArmed = payload.isArmed
  }

  const commitVersionState = (payload: VersionPayload): void => {
    if (payload?.version !== undefined) state.value.version = payload.version
  }

  const fetchSystemState = async (): Promise<{ success: boolean; status?: number }> => {
    try {
      const res = await api.get('/api/v1/system/state')
      const data = (await res.json()) as SystemStatePayload
      commitSystemState(data)
      return { success: true }
    } catch (err: any) {
      console.error('Failed to fetch system state:', err)
      
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
        transitionSession('unauthenticated')
      }
      
      return { success: false, status: err.status }
    }
  }

  // --- ACTIONS: SYSTEM CONTROL ---
  
  const toggleArmed = async (): Promise<{ success: boolean }> => {
    const targetState = !state.value.isArmed
    
    try {
      await api.patch('/api/v1/system/state', { isArmed: targetState })
      await fetchSystemState()
      return { success: true }
    } catch (err) {
      console.error('Failed to toggle armed state:', err)
      await fetchSystemState()
      return { success: false }
    }
  }

  const changePassword = async (currentPassword: string, newPassword: string): Promise<{ success: boolean; error?: string }> => {
    try {
      await api.patch('/api/v1/system/password', { currentPassword, newPassword })
      return { success: true }
    } catch (err: any) {
      return { 
        success: false, 
        error: err.status === 401 ? 'Incorrect current password.' : (err.message || 'Failed to update password.')
      }
    }
  }

  const factoryReset = async (password: string, dryrun: boolean = false): Promise<{ success: boolean; error?: string; stats?: any }> => {
    try {
      const url = dryrun ? '/api/v1/system/reset?dryrun=true' : '/api/v1/system/reset'
      const response = await api.post(url, { password })
      
      let stats = undefined
      if (dryrun) {
          const data = await response.json()
          stats = data.stats
      }
      
      return { success: true, stats }
    } catch (err: any) {
      if (err.status === 401) {
        return { success: false, error: 'Incorrect password.' }
      }
      return { success: false, error: err.message || 'Factory reset failed.' }
    }
  }

  // --- ACTIONS: AUTH & SETUP ---
  
  const login = async (password: string): Promise<{ success: boolean; status?: number }> => {
    state.value.authError = null
    try {
      await api.post('/login', { password })
      transitionSession('authenticated')
      return { success: true }
    } catch (err: any) {
      transitionSession('unauthenticated')
      state.value.authError = 'Invalid credentials'
      return { success: false, status: err.status }
    }
  }

  const logout = async (): Promise<void> => {
    try {
      await api.post('/logout') 
    } catch (err) {
      console.error('Logout request failed', err)
    } finally {
      window.location.href = '/'
    }
  }

  const completeSetup = async (password: string, hubEndpoint: string): Promise<{ success: boolean; error?: string }> => {
    state.value.setupError = null
    try {
      await api.post('/api/v1/setup', { password, hubEndpoint })
      state.value.requiresSetup = false
      transitionSession('unauthenticated')
      return { success: true }
    } catch (err: any) {
      state.value.setupError = err.message || 'Setup failed'
      return { success: false, error: err.message || 'Setup failed' }
    }
  }

  // --- ACTIONS: BOOTSTRAP ---
  
  const checkRequiresSetup = async (): Promise<void> => {
    const res = await api.get('/api/v1/setup/status')
    const data = (await res.json()) as { requiresSetup: boolean }
    state.value.requiresSetup = data.requiresSetup || false
  }

  const checkCoreConfiguration = async (): Promise<void> => {
    const [stateRes, verRes] = await Promise.allSettled([
      api.get('/api/v1/system/state').then(r => r.json()),
      api.get('/api/v1/version').then(r => r.json())
    ])

    if (verRes.status === 'fulfilled') commitVersionState(verRes.value as VersionPayload)

    if (stateRes.status === 'fulfilled') {
      commitSystemState(stateRes.value as SystemStatePayload)
      if (state.value.sessionState === 'unknown') {
        transitionSession('authenticated')
      }
    } else {
      if (stateRes.status === 'rejected') {
        const err = stateRes.reason
        if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
          transitionSession('unauthenticated')
        } else {
          throw new Error('Failed to retrieve core system configuration.')
        }
      }
    }
  }

  const initAppStore = async (): Promise<void> => {
    state.value.bootstrapError = null
    try {
      await checkRequiresSetup()
      if (!state.value.requiresSetup) {
        await checkCoreConfiguration()
      }
    } catch (err: any) {
      console.error('App bootstrap failed:', err)
      state.value.bootstrapError = err.message || 'Failed to initialize application'
    } finally {
      if (state.value.sessionState === 'unknown') {
        transitionSession('unauthenticated')
      }
      state.value.isInitialized = true 
    }
  }

  // --- ACTIONS: UI ---
  const toggleTheme = (): void => {
    const html = document.documentElement
    if (html.classList.contains('dark')) {
      html.classList.remove('dark')
      localStorage.setItem('theme', 'light')
    } else {
      html.classList.add('dark')
      localStorage.setItem('theme', 'dark')
    }
  }
  const toggleSidebar = (): void => { state.value.sidebarOpen = !state.value.sidebarOpen }
  const toggleArchive = (): void => { state.value.viewingArchive = !state.value.viewingArchive }
  const setVelocityTimeframe = (timeframe: string): void => { state.value.velocityTimeframe = timeframe }

  // Mock for Dev Environment (invoked if URL params contain debug=setup)
  const enableDebugSetup = (): void => {
    state.value.requiresSetup = true
    state.value.isInitialized = true
  }

  return {
    isArmed, version, viewingArchive, sidebarOpen, 
    activeTimeframe, velocityTimeframe, sessionState, 
    authError, setupError, requiresSetup, isInitialized, bootstrapError,
    isAuthenticated, isReady, isBootstrapping, canAccessDashboard,
    toggleTheme, toggleSidebar, toggleArchive, setVelocityTimeframe,
    toggleArmed, changePassword, factoryReset, login, logout, completeSetup,
    initAppStore, fetchSystemState, enableDebugSetup
  }
})