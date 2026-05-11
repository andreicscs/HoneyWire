<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { storeToRefs } from 'pinia'

// Components
import Sidebar from './components/Sidebar.vue'
import Header from './components/Header.vue'
import Dashboard from './views/Dashboard.vue'
import Login from './views/Login.vue'
import Store from './views/Store.vue'
import Settings from './views/Settings.vue'
import Setup from './views/Setup.vue'

// Services & Stores
import { useConfig } from './api/useConfig'
import { useAppStore } from './stores/app'
import { useFleetStore } from './stores/fleet'
import { useEventsStore } from './stores/events'
import { HoneyWireWS } from './services/ws' 

const { fetchConfig } = useConfig()
const appStore = useAppStore()
const fleetStore = useFleetStore()
const eventsStore = useEventsStore()

const { currentView, viewingArchive } = storeToRefs(appStore)

watch([viewingArchive, () => fleetStore.selectedNode, () => fleetStore.selectedSensor], 
  ([isArchived, node, sensor]) => {
    eventsStore.fetchEvents(isArchived, node, sensor)
})

watch(() => fleetStore.activeTimeframe, (newTimeframe) => {
    fleetStore.fetchUptime(newTimeframe)
})

const requiresSetup = ref(false)
const isAuthenticated = ref(false)

const wsService = new HoneyWireWS()
let healthSyncInterval = null

const checkAuthAndInit = async () => {
    // ----------------------------------------------------
    // TODO DEBUG OVERRIDE: 
    // Access http://localhost:8080/?debug=setup to force UI rendering
    // ----------------------------------------------------
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('debug') === 'setup') {
        requiresSetup.value = true;
        isAuthenticated.value = false;
        return;
    }

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
            
            await Promise.all([
                fleetStore.fetchFleet(),
                fleetStore.fetchUptime(fleetStore.activeTimeframe),
                eventsStore.fetchEvents()
            ])

            wsService.on('onNewEvent', (payload) => eventsStore.handleWsEvent(payload))
            wsService.on('onNewSensor', (payload) => fleetStore.handleWsUpdate('NEW_SENSOR', payload))
            wsService.on('onDeleteSensor', (payload) => fleetStore.handleWsUpdate('DELETE_SENSOR', payload))
            wsService.on('onSilenceSensor', (payload) => fleetStore.handleWsUpdate('SILENCE_SENSOR', payload))
            wsService.on('onSensorHeartbeat', (payload) => fleetStore.handleWsUpdate('SENSOR_HEARTBEAT', payload))

            wsService.connect()
            
            healthSyncInterval = setInterval(() => { 
                fleetStore.fetchFleet()
                fleetStore.fetchUptime(fleetStore.activeTimeframe) 
            }, 30000)

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

onMounted(() => {
    checkAuthAndInit()
})

onUnmounted(() => {
    if (healthSyncInterval) clearInterval(healthSyncInterval)
    wsService.disconnect()
})
</script>

<script>
if (localStorage.theme === 'dark' || (!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    document.documentElement.classList.add('dark')
}
</script>

<template>
  <div v-if="requiresSetup" class="h-screen bg-bg">
    <Setup @setup-complete="checkAuthAndInit" @toggle-theme="toggleTheme" />
  </div>
  
  <div v-else-if="!isAuthenticated" class="h-screen bg-bg">
    <Login @login-success="checkAuthAndInit" @toggle-theme="toggleTheme" /> 
  </div>

  <div v-else class="flex h-screen overflow-hidden bg-bg text-text-main transition-colors duration-200">
    <Sidebar />
    <main class="flex-1 flex flex-col min-w-0 bg-grid">
      <Header />
      <div class="flex-1 overflow-auto custom-scroll p-4 sm:p-6">
        <Dashboard v-if="currentView === 'dashboard'" />
        <Store v-else-if="currentView === 'store'" />
        <Settings v-else-if="currentView === 'settings'" />
      </div>
    </main>
  </div>
</template>