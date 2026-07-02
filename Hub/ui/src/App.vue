<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'

import Sidebar from './components/layout/Sidebar.vue'
import Header from './components/layout/Header.vue'
import Setup from './views/Setup.vue'
import Login from './views/Login.vue'

import { useConfigStore } from './stores/Config/config'
import { useAppStore } from './stores/System/app'
import { useFleetStore } from './stores/Fleet/fleet'
import { useEventsStore } from './stores/Events/events'
import { HoneyWireWS } from './services/ws'

const configStore = useConfigStore()
const appStore = useAppStore()
const fleetStore = useFleetStore()
const eventsStore = useEventsStore()

const router = useRouter()
const route = useRoute()

const { isInitialized, requiresSetup, isAuthenticated, viewingArchive, bootstrapError } = storeToRefs(appStore)

watch([viewingArchive, () => fleetStore.selectedNode?.id, () => fleetStore.selectedSensor?.sensorId],
  ([isArchived, nodeId, sensorId]) => {
    eventsStore.fetchEvents(isArchived as boolean, (nodeId as string) || null, (sensorId as string) || null)
})

watch(() => fleetStore.activeTimeframe, (newTimeframe) => {
    fleetStore.fetchUptime(newTimeframe)
})

const wsService = new HoneyWireWS()

const isDataLoaded = ref(false)

const loadAppData = async () => {
  isDataLoaded.value = false
  try {
    await configStore.fetchConfig().catch(e => console.warn("Config fetch non-fatal error:", e))

    // Reconcile system state (isArmed) now that we have an active session
    await appStore.fetchSystemState().catch(() => {})

    await Promise.all([
      fleetStore.fetchFleet().catch(e => console.error("Fleet fetch error:", e)),
      fleetStore.fetchUptime(fleetStore.activeTimeframe).catch(e => console.error("Uptime fetch error:", e)),
      eventsStore.fetchEvents().catch(e => console.error("Events fetch error:", e)),
    ])

    // TODO: REMOVE DEBUG OVERRIDE BEFORE PRODUCTION
    // You can test the "First Startup" UI state at any time by running:
    // localStorage.setItem('DEBUG_FIRST_STARTUP', 'true') in your browser console.
    if ((fleetStore.nodes.length === 0 || localStorage.getItem('DEBUG_FIRST_STARTUP') === 'true') && route.name !== 'fleet') {
      await router.push('/fleet').catch(() => {})
    }

    wsService.on('onNewEvent', (payload: any) => {
      eventsStore.handleWsEvent(payload)
      fleetStore.handleWsUpdate('NEW_EVENT', payload)
    })
    wsService.on('onNewSensor', (payload: any) => fleetStore.handleWsUpdate('NEW_SENSOR', payload))
    wsService.on('onDeleteSensor', (payload: any) => fleetStore.handleWsUpdate('DELETE_SENSOR', payload))
    wsService.on('onSilenceSensor', (payload: any) => fleetStore.handleWsUpdate('SILENCE_SENSOR', payload))
    wsService.on('onSensorHeartbeat', (payload: any) => fleetStore.handleWsUpdate('SENSOR_HEARTBEAT', payload))
    wsService.on('onNewNode', (payload: any) => fleetStore.handleWsUpdate('NEW_NODE', payload))
    wsService.on('onUpdateNode', (payload: any) => fleetStore.handleWsUpdate('UPDATE_NODE', payload))
    wsService.on('onDeleteNode', (payload: any) => {
      fleetStore.handleWsUpdate('DELETE_NODE', payload)
      eventsStore.fetchEvents().catch(() => {})
      eventsStore.fetchSeverityProjection('alltime', fleetStore.selectedNodeId, fleetStore.selectedSensorId).catch(() => {})
      eventsStore.fetchSummaryProjection('24H', fleetStore.selectedNodeId, fleetStore.selectedSensorId).catch(() => {})
      eventsStore.invalidateThreatVelocityProjection()
    })
    wsService.on('onNodeSynced', (payload: any) => fleetStore.handleWsUpdate('NODE_SYNCED', payload))
    wsService.on('onCatalogUpdated', (payload: any) => fleetStore.handleWsUpdate('CATALOG_UPDATED', payload))

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
  } finally {
    isDataLoaded.value = true
  }
}

const checkAuthAndInit = async () => {
  // TODO remove debug override before production
  const urlParams = new URLSearchParams(window.location.search)
  if (urlParams.get('debug') === 'setup') {
    appStore.enableDebugSetup()
    return
  }

  try {
    await appStore.initAppStore()

    if (requiresSetup.value) {
      return
    }

    if (bootstrapError.value) {
      return
    }

  } catch (e) {
    console.error("Hub connection error:", e)
  }
}

watch(isAuthenticated, (isAuth) => {
  if (isAuth) {
    loadAppData()
  } else {
    isDataLoaded.value = false
  }
})

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
  <div v-if="!isInitialized || (isAuthenticated && !isDataLoaded)" class="h-screen bg-bg flex items-center justify-center z-50">
     <div class="animate-pulse flex flex-col items-center gap-4">
         <svg class="w-10 h-10 text-primary-main animate-spin" fill="none" viewBox="0 0 24 24">
             <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
             <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
         </svg>
         <span class="text-text-m font-medium tracking-wide">{{ !isInitialized ? 'Initializing...' : 'Loading Telemetry...' }}</span>
     </div>
  </div>

  <div v-else-if="requiresSetup" class="h-screen bg-bg">
    <Setup @toggle-theme="appStore.toggleTheme" />
  </div>

  <div v-else-if="!isAuthenticated" class="h-screen bg-bg">
    <Login @toggle-theme="appStore.toggleTheme" />
  </div>

  <div v-else class="flex h-screen overflow-hidden bg-bg text-text-h transition-colors duration-200">
    <Sidebar />
    <main class="flex-1 flex flex-col min-w-0 bg-grid">
      <Header />
      <div class="flex-1 overflow-auto custom-scroll p-4 sm:p-6">
        <router-view />
      </div>
    </main>
  </div>
</template>