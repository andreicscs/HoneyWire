<script setup>
import { onMounted, onUnmounted, watch } from 'vue'
import { storeToRefs } from 'pinia'

import Sidebar from './components/layout/Sidebar.vue'
import Header from './components/layout/Header.vue'
import Dashboard from './views/Dashboard.vue'
import FleetView from './views/FleetView.vue'
import NodeDetailView from './views/NodeDetailView.vue'
import Login from './views/Login.vue'
import Store from './views/Store.vue'
import Settings from './views/Settings.vue'
import Setup from './views/Setup.vue'

import { useConfig } from './api/useConfig'
import { useAppStore } from './stores/System/app'
import { useFleetStore } from './stores/Fleet/fleet'
import { useEventsStore } from './stores/Events/events'
import { HoneyWireWS } from './services/ws'

const { fetchConfig } = useConfig()
const appStore = useAppStore()
const fleetStore = useFleetStore()
const eventsStore = useEventsStore()

const { currentView, viewingArchive } = storeToRefs(appStore)

watch([viewingArchive, () => fleetStore.selectedNode, () => fleetStore.selectedSensor],
  ([isArchived, node, sensor]) => {
    eventsStore.fetchEvents(isArchived, node?.id, sensor?.sensorId)
})

watch(() => fleetStore.activeTimeframe, (newTimeframe) => {
    fleetStore.fetchUptime(newTimeframe)
})

const wsService = new HoneyWireWS()

const loadAppData = async () => {
  try {
    await fetchConfig().catch(e => console.warn("Config fetch non-fatal error:", e))

    // Reconcile system state (isArmed) now that we have an active session
    await appStore.fetchSystemState().catch(() => {})

    await Promise.all([
      fleetStore.fetchFleet().catch(e => console.error("Fleet fetch error:", e)),
      fleetStore.fetchUptime(fleetStore.activeTimeframe).catch(e => console.error("Uptime fetch error:", e)),
      eventsStore.fetchEvents().catch(e => console.error("Events fetch error:", e)),
    ])

    wsService.on('onNewEvent', (payload) => eventsStore.handleWsEvent(payload))
    wsService.on('onNewSensor', (payload) => fleetStore.handleWsUpdate('NEW_SENSOR', payload))
    wsService.on('onDeleteSensor', (payload) => fleetStore.handleWsUpdate('DELETE_SENSOR', payload))
    wsService.on('onSilenceSensor', (payload) => fleetStore.handleWsUpdate('SILENCE_SENSOR', payload))
    wsService.on('onSensorHeartbeat', (payload) => fleetStore.handleWsUpdate('SENSOR_HEARTBEAT', payload))
    wsService.on('onNewNode', (payload) => fleetStore.handleWsUpdate('NEW_NODE', payload))
    wsService.on('onUpdateNode', (payload) => fleetStore.handleWsUpdate('UPDATE_NODE', payload))
    wsService.on('onDeleteNode', (payload) => fleetStore.handleWsUpdate('DELETE_NODE', payload))
    wsService.on('onNodeSynced', (payload) => fleetStore.handleWsUpdate('NODE_SYNCED', payload))

    wsService.on('onReconnect', async () => {
      console.log("WebSocket Reconnected: Syncing missed data...")
      await Promise.all([
        fleetStore.fetchFleet().catch(() => {}),
        fleetStore.fetchUptime(fleetStore.activeTimeframe).catch(() => {}),
        eventsStore.fetchEvents().catch(() => {}),
      ])
    })

    wsService.on('onSyncCharts', () => {
      fleetStore.fetchUptime(fleetStore.activeTimeframe)
    })

    wsService.connect()

  } catch (e) {
    console.error("Critical failure during loadAppData:", e)
  }
}

const checkAuthAndInit = async () => {
  const urlParams = new URLSearchParams(window.location.search)
  if (urlParams.get('debug') === 'setup') {
    appStore.requiresSetup = true
    return
  }

  try {
    await appStore.initAppStore()

    if (appStore.requiresSetup) {
      return
    }

    if (appStore.bootstrapError) {
      return
    }

  } catch (e) {
    console.error("Hub connection error:", e)
  }
}

watch(() => appStore.isAuthenticated, (isAuth) => {
  if (isAuth) {
    appStore.requiresSetup = false
    loadAppData()
  }
})

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
  // Restore saved theme before any rendering
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark') {
    document.documentElement.classList.add('dark')
  } else if (savedTheme === 'light') {
    document.documentElement.classList.remove('dark')
  } else {
    // No preference saved — check system preference
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      document.documentElement.classList.add('dark')
    }
  }

  checkAuthAndInit()
})

onUnmounted(() => {
  wsService.disconnect()
})
</script>

<template>
  <div v-if="!appStore.isInitialized" class="h-screen bg-bg flex items-center justify-center z-50">
     <div class="animate-pulse flex flex-col items-center gap-4">
         <svg class="w-10 h-10 text-primary-main animate-spin" fill="none" viewBox="0 0 24 24">
             <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
             <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
         </svg>
         <span class="text-text-m font-medium tracking-wide">Initializing Sentinel...</span>
     </div>
  </div>

  <div v-else-if="appStore.requiresSetup" class="h-screen bg-bg">
    <Setup @toggle-theme="toggleTheme" />
  </div>

  <div v-else-if="!appStore.isAuthenticated" class="h-screen bg-bg">
    <Login @toggle-theme="toggleTheme" />
  </div>

  <div v-else class="flex h-screen overflow-hidden bg-bg text-text-h transition-colors duration-200">
    <Sidebar />
    <main class="flex-1 flex flex-col min-w-0 bg-grid">
      <Header />
      <div class="flex-1 overflow-auto custom-scroll p-4 sm:p-6">
        <Dashboard v-if="currentView === 'dashboard'" />
        <FleetView v-else-if="currentView === 'fleet'" />
        <NodeDetailView v-else-if="currentView === 'node-detail'" />
        <Store v-else-if="currentView === 'store'" />
        <Settings v-else-if="currentView === 'settings'" />
      </div>
    </main>
  </div>
</template>