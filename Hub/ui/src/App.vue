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
    eventsStore.fetchEvents(isArchived, node?.id, sensor?.sensorId)
})

watch(() => fleetStore.activeTimeframe, (newTimeframe) => {
    fleetStore.fetchUptime(newTimeframe)
})

const wsService = new HoneyWireWS()

const loadAppData = async () => {
  try {
    await fetchConfig()
    await appStore.checkSetupStatus()

    await Promise.all([
      fleetStore.fetchFleet(),
      fleetStore.fetchUptime(fleetStore.activeTimeframe),
      eventsStore.fetchEvents(),
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
        fleetStore.fetchFleet(),
        fleetStore.fetchUptime(fleetStore.activeTimeframe),
        eventsStore.fetchEvents(),
      ])
    })

    wsService.on('onSyncCharts', () => {
      fleetStore.fetchUptime(fleetStore.activeTimeframe)
    })

    wsService.connect()

    appStore.isAuthenticated = true
    appStore.isInitialized = true

  } catch (e) {
    console.error("Failed to load application data:", e)
    appStore.isAuthenticated = true
    appStore.isInitialized = true
  }
}

const checkAuthAndInit = async () => {
  const urlParams = new URLSearchParams(window.location.search)
  if (urlParams.get('debug') === 'setup') {
    appStore.requiresSetup = true
    appStore.isAuthenticated = false
    return
  }

  try {
    const needsSetup = await appStore.checkRequiresSetup()
    if (needsSetup) {
      appStore.requiresSetup = true
      appStore.isAuthenticated = false
      return
    }

    appStore.requiresSetup = false

    const authenticated = await appStore.checkSystemState()
    if (!authenticated) {
      appStore.isAuthenticated = false
      return
    }

    await loadAppData()

  } catch (e) {
    console.error("Hub connection error:", e)
    appStore.isAuthenticated = false
  }
}

const onLoginSuccess = async () => {
  appStore.requiresSetup = false
  await loadAppData()
}

const onSetupComplete = async () => {
  appStore.requiresSetup = false
  await loadAppData()
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
  <div v-if="appStore.requiresSetup" class="h-screen bg-bg">
    <Setup @setup-complete="onSetupComplete" @toggle-theme="toggleTheme" />
  </div>

  <div v-else-if="!appStore.isAuthenticated" class="h-screen bg-bg">
    <Login @login-success="onLoginSuccess" @toggle-theme="toggleTheme" />
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