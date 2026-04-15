<script setup>
  import { ref, onMounted } from 'vue'
  import Sidebar from './components/Sidebar.vue'
  import Header from './components/Header.vue'
  import Dashboard from './views/Dashboard.vue'
  import Login from './views/Login.vue'
  import { useSentinel } from './api/useSentinel'
  import Store from './views/Store.vue'
  import Settings from './views/Settings.vue'
  import { useConfig } from './api/useConfig'
  import Setup from './views/Setup.vue'

  const { 
    version, 
    isArmed, 
    unreadCount, 
    viewingArchive, 
    startRealtimeSync, 
    toggleArmed, 
    markAllRead,
    events,
    logout
  } = useSentinel()
  const { fetchConfig } = useConfig()

  const requiresSetup = ref(false)
  const isAuthenticated = ref(false)
  const currentView = ref('dashboard')
  const sidebarOpen = ref(true)
 
  const checkAuthAndInit = async () => {
    try {
        const setupRes = await fetch('/api/v1/setup/status')
        if (setupRes.ok) {
            const setupData = await setupRes.json()
            if (setupData.requires_setup) {
                requiresSetup.value = true
                isAuthenticated.value = false
                return
            }
        }
        
        requiresSetup.value = false

        const res = await fetch('/api/v1/system/state')
        if (res.ok) {
            isAuthenticated.value = true
            await fetchConfig() 
            
            startRealtimeSync()
        } else {
            isAuthenticated.value = false
        }
    } catch (e) {
        console.error("Hub connection error:", e)
        isAuthenticated.value = false
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

  // --- DRYRUN PURGE LOGIC ---
  const clearLogs = async () => {
    try {
        //Perform the Dry Run to get the exact count
        const dryRes = await fetch('/api/v1/events?dryrun=true', { method: 'DELETE' })
        if (!dryRes.ok) throw new Error("Failed to fetch dryrun data")
        
        const dryData = await dryRes.json()
        const count = dryData.would_delete || 0

        if (count === 0) {
            alert("The database is already empty.")
            return
        }

        //Ask user with the specific count
        if (confirm(`Confirm Database Purge?\n\nThis will permanently delete ${count} active and archived event logs.\n\nThis action cannot be undone.`)) {
            
            // Optimistic UI wipe
            if (events) events.value = [] 
            if (unreadCount) unreadCount.value = 0 
            
            //The actual deletion
            const response = await fetch('/api/v1/events?dryrun=false', {
                method: 'DELETE',
                headers: { 'Content-Type': 'application/json' }
            })

            if (!response.ok) {
                console.error("Failed to purge logs")
                alert("Failed to purge logs. Check server console.")
            } else {
                console.log("Database purged successfully.")
            }
        }
    } catch (error) {
        console.error("Network error while purging logs:", error)
        alert("Network error. Could not reach the Hub.")
    }
  }

  onMounted(() => {
    checkAuthAndInit()
  })
</script>

<script>
  if (localStorage.theme === 'dark' || (!('theme' in localStorage) && 
      window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    document.documentElement.classList.add('dark')
  }
</script>

<template>
  <div v-if="requiresSetup" class="h-screen bg-slate-100 dark:bg-[#0a0a0c]">
    <Setup @setup-complete="checkAuthAndInit" @toggle-theme="toggleTheme" />
  </div>
  
  <div v-if="!isAuthenticated" class="h-screen bg-slate-100 dark:bg-[#0a0a0c]">
    <Login 
      @login-success="checkAuthAndInit" 
      @toggle-theme="toggleTheme"
    /> 
  </div>

  <div v-else class="flex h-screen overflow-hidden bg-slate-200/60 dark:bg-[#0a0a0c] text-slate-700 dark:text-zinc-200 transition-colors duration-200">
    
    <Sidebar 
      :isOpen="sidebarOpen" 
      :currentView="currentView" 
      :version="version" 
      :viewingArchive="viewingArchive"
      @change-view="v => currentView = v" 
      @toggle-archive="viewingArchive = !viewingArchive" 
      @clear-logs="clearLogs"
      @toggle-sidebar="sidebarOpen = !sidebarOpen" 
    />

    <main class="flex-1 flex flex-col min-w-0 bg-grid">
      
      <Header 
        :currentView="currentView" 
        :isArmed="isArmed" 
        :unreadCount="unreadCount" 
        @toggle-theme="toggleTheme" 
        @toggle-armed="toggleArmed" 
        @mark-all-read="markAllRead" 
        @logout="logout" 
      />
      
      <div class="flex-1 overflow-auto custom-scroll p-4 sm:p-6">
        
        <div v-if="currentView === 'dashboard'">
          <Dashboard /> 
        </div>

        <div v-else-if="currentView === 'store'">
          <Store />
        </div>

        <div v-else-if="currentView === 'settings'">
          <Settings />
        </div>

      </div>
    </main>
  </div>
</template>