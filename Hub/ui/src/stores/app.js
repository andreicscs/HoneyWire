import { defineStore } from 'pinia'
import { ref } from 'vue'

/**
 * App Store (Global UI State)
 * 
 * Manages system-wide UI settings: armed state, theme, archive view, sidebar, and current view.
 * Auth checking is handled in App.vue, not here.
 */

export const useAppStore = defineStore('app', () => {
  // --- STATE ---
  const isArmed = ref(true)
  const version = ref('1.0.0')
  const viewingArchive = ref(false)
  const sidebarOpen = ref(true)
  const currentView = ref('dashboard')
  const activeTimeframe = ref('24H')
  const velocityTimeframe = ref('24H')

  // --- ACTIONS ---

  /**
   * Toggle the armed state of the system
   * Makes a PATCH request to /api/v1/system/state
   */
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

  /**
   * Toggle between light and dark theme
   */
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

  /**
   * Logout and redirect to login page
   * Makes a POST request to /logout
   */
  const logout = async () => {
    try {
      await fetch('/logout', { method: 'POST' })
      window.location.href = '/'
    } catch (err) {
      console.error('Logout failed', err)
    }
  }

  /**
   * Fetch system version and state from backend (called during init)
   */
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

  return {
    // State
    isArmed,
    version,
    viewingArchive,
    sidebarOpen,
    currentView,
    activeTimeframe,
    velocityTimeframe,
    // Actions
    toggleArmed,
    toggleTheme,
    logout,
    checkSetupStatus,
  }
})
