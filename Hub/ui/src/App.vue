<script setup>
  import { ref, onMounted } from 'vue'
  import Sidebar from './components/Sidebar.vue'
  import Header from './components/Header.vue'
  import Dashboard from './views/Dashboard.vue'
  import Login from './views/Login.vue'
  import { useSentinel } from './api/useSentinel'

  const { 
    version, 
    isArmed, 
    unreadCount, 
    viewingArchive, 
    startPolling, 
    toggleArmed, 
    markAllRead 
  } = useSentinel()

  const isAuthenticated = ref(false)
  const currentView = ref('dashboard')
  const sidebarOpen = ref(true)
 
  const checkAuthAndInit = async () => {
    try {
        const res = await fetch('/api/v1/system/state')
        if (res.ok) {
            isAuthenticated.value = true
            startPolling()
        }
    } catch (e) {
        // Not authenticated, leave at false
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

  const clearLogs = async () => {
    if (confirm("Confirm Database Purge?\n\nThis will permanently delete ALL active and archived event logs. This action cannot be undone.")) {
        
        try {
            const response = await fetch('/api/v1/events', {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json'
                }
            })

            if (!response.ok) {
                const errText = await response.text()
                console.error("Failed to purge logs:", errText)
                alert("Failed to purge logs. See console for details.")
            } else {
                console.log("Database purged successfully. UI will update on next poll.")
                // Optional: If you want it to clear instantly without waiting for the 5s poll, 
                // you can do a soft reload here:
                // window.location.reload()
            }
            
        } catch (error) {
            console.error("Network error while purging logs:", error)
            alert("Network error. Could not reach the Hub to purge logs.")
        }
    }
  }

  onMounted(() => {
    checkAuthAndInit()
    
    if (localStorage.getItem('theme') === 'light' || (!('theme' in localStorage) && !window.matchMedia('(prefers-color-scheme: dark)').matches)) {
        document.documentElement.classList.remove('dark')
    } else {
        document.documentElement.classList.add('dark')
    }
  })
</script>

<template>
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
      />
      
      <div class="flex-1 overflow-auto custom-scroll p-4 sm:p-6">
        
        <div v-if="currentView === 'dashboard'">
          <Dashboard /> 
        </div>

        <div v-else-if="currentView === 'store'">
          <h1 class="text-xl font-bold">Sensor Store Placeholder</h1>
        </div>

        <div v-else-if="currentView === 'settings'">
          <h1 class="text-xl font-bold">Settings Placeholder</h1>
        </div>

      </div>
    </main>
  </div>
</template>