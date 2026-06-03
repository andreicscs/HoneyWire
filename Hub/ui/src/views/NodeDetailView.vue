<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useAppStore } from '../stores/System/app.ts'
import { useFleetStore } from '../stores/Fleet/fleet.ts'
import { useEventsStore } from '../stores/Events/events.ts'
import { useConfigStore } from '../stores/Config/config.ts'
import type { FleetNode } from '../stores/Fleet/fleet.ts'

import NodeDetailHeader from '../components/ui/layout/NodeDetailHeader.vue'
import NodeStatWidgets from '../components/ui/layout/NodeStatWidgets.vue'
import NodeSensorGrid from '../components/ui/layout/NodeSensorGrid.vue'
import NodeCatalogGrid from '../components/ui/layout/NodeCatalogGrid.vue'
import NodeKeyModal from '../components/ui/feedback/NodeKeyModal.vue'
import NodeSyncModal from '../components/ui/feedback/NodeSyncModal.vue'
import NodeSensorModal from '../components/ui/feedback/NodeSensorModal.vue'

const appStore = useAppStore()
const fleetStore = useFleetStore()
const eventsStore = useEventsStore()
const configStore = useConfigStore()

const selectedNodeId = computed(() => fleetStore.selectedNodeId)

// --- NODE STATE ---
const node = computed(() => fleetStore.getNode(selectedNodeId.value))

// --- LAST EVENT (from events store) ---
const lastEventTime = computed(() => {
    const events = eventsStore.filteredEvents
    if (!events || events.length === 0) return 'None'
    const latest = events.reduce((a, b) => new Date(a.timestamp) > new Date(b.timestamp) ? a : b)
    return timeAgo(latest.timestamp)
})

// --- MANIFEST LOOKUP (for icon/OSI enrichment) ---
const isManifestLoading = ref(true)
const fetchError = ref(false)
const sensors = ref<any[]>([])
const manifestMap = computed(() => {
  const map = new Map()
  for (const s of sensors.value) {
    map.set(s.id, s)
    map.set(s.sensorId, s)
    map.set(s.name, s)
  }
  return map
})

const getManifestForSensor = (installedSensor: any) => {
  const manifest = manifestMap.value.get(installedSensor.id)
    || manifestMap.value.get(installedSensor.name)
    || manifestMap.value.get(installedSensor.sensorId)
  return manifest
}

const showSensorModal = ref(false)
const selectedSensor = ref<any>(null)
const isEditingSensor = ref(false)
const editingSensorId = ref<string | null>(null)
const initialEnvVars = ref<Record<string, any>>({})

// --- MODAL STATE ---
const showKeyModal = ref(false)
const showSyncModal = ref(false)
const syncComposeYaml = ref('')

const syncCommand = computed(() => {
    if (!node.value) return ''
    const hubUrl = configStore.config.hubEndpoint || window.location.origin
    if (!node.value.lastHeartbeat) {
        return `./wizard --link ${hubUrl} ${node.value.apiKey}\n./wizard apply`
    }
    return `./wizard apply`
})

// --- ENV VAR HELPERS ---
const getUIDefault = (def: any) => {
  if (def === undefined || def === null) return ''
  const strDef = String(def)
  if (!strDef.includes('{{')) return strDef
  const elseMatch = strDef.match(/\{\{\s*else\s*\}\}(.*?)\{\{\s*end\s*\}\}/)
  if (elseMatch) return elseMatch[1].trim()
  const funcMatch = strDef.match(/\{\{\s*[a-zA-Z]+\s+([0-9]+)\s*\}\}/)
  if (funcMatch) return funcMatch[1].trim()
  return ''
}

// --- SENSOR ACTIONS (delegated to store) ---

const handleToggleSensorSilence = async (sensor: any) => {
  if (!node.value?.id || !sensor.sensorId) return
  try {
    await fleetStore.toggleSilence(node.value.id, sensor.sensorId, !sensor.isSilenced)
  } catch (err) {
    alert('Unable to change sensor silence state. Please try again.')
  }
}

const handleRemoveSensor = async (sensor: any) => {
  if (!node.value?.id || !sensor.sensorId) return
  if (!confirm('Remove this sensor? The node will be marked for deployment sync.')) return
  const res = await fleetStore.removeSensor(node.value.id, sensor.sensorId)
  if (!res.success) {
    alert(res.error)
  }
}

const handleApplySensor = async (configValues: Record<string, string>) => {
  if (!selectedSensor.value || !node.value?.id) return

  try {
    if (isEditingSensor.value && editingSensorId.value) {
      await fleetStore.updateSensor(node.value.id, editingSensorId.value, {
        customName: selectedSensor.value.name || selectedSensor.value.id,
        configValues,
      })
    } else {
      await fleetStore.addSensor(node.value.id, {
        sensorId: selectedSensor.value.id || selectedSensor.value.sensorId || selectedSensor.value.name,
        customName: selectedSensor.value.name || selectedSensor.value.id,
        configValues,
      })
    }
    closeSensor()
    await fleetStore.fetchUptime()
  } catch (err) {
    alert(isEditingSensor.value ? 'Could not update sensor. Please try again.' : 'Could not add sensor to this node. Please try again.')
  }
}

const handleUpdateNode = async (updates: Partial<FleetNode>) => {
    if (!node.value) return
    try {
        await fleetStore.updateNode(node.value.id, {
            alias: node.value.alias,
            tags: node.value.tags,
            publicIp: node.value.publicIp || '',
            privateIp: node.value.privateIp || '',
            ...updates
        })
    } catch (err) {
        // Store handles optimistic update rollback
    }
}

const triggerManualSync = async () => {
  if (!node.value?.id) return
  const res = await fleetStore.syncNode(node.value.id)
  if (res.success && res.yaml) {
    syncComposeYaml.value = res.yaml
    showSyncModal.value = true
  } else {
    alert(res.error || 'Unable to sync this node. Please try again.')
  }
}

const handleSilenceNode = () => {
    if (!node.value?.id) return
    fleetStore.silenceNode(node.value.id)
}

const handleDeleteNode = async () => {
    if (!node.value?.id) return
    if (confirm(`Delete node "${node.value.alias}"? This cannot be undone.`)) {
        const res = await fleetStore.deleteNode(node.value.id)
        if (res.success) {
            appStore.setView('fleet')
        } else {
            alert(res.error)
        }
    }
}

const viewAllEvents = () => {
    // Keeps the current node selected in the fleetStore to act as a filter
    appStore.setView('dashboard')
}

// --- NAVIGATION ---

watch(selectedNodeId, async (value) => {
    if (!value) {
        if (appStore.currentView === 'node-detail') {
            appStore.setView('fleet')
        }
        return
    }
        await fleetStore.fetchNodeDetails(value)
}, { immediate: true })

const timeAgo = (dateStr: string) => {
    if (!dateStr) return 'Unknown'
    const diff = Math.floor((new Date().getTime() - new Date(dateStr).getTime()) / 1000)
    if (diff < 60) return `${diff}s ago`
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
    return `${Math.floor(diff / 86400)}d ago`
}

const recentActivity = computed(() => {
  return eventsStore.filteredEvents.slice(0, 5).map(e => ({
    id: e.id,
    time: timeAgo(e.timestamp),
    severity: e.severity || 'info',
    eventTrigger: e.eventTrigger || 'Alert',
    source: e.source || 'Unknown',
    sensorId: e.sensorId || ''
  }))
})

// --- SENSOR CATALOG ---

onMounted(async () => {
  isManifestLoading.value = true
  try {
    sensors.value = await fleetStore.fetchManifests()
  } catch (error) {
    console.error(error)
    fetchError.value = true
  } finally {
    isManifestLoading.value = false
  }
})

const openSensor = (sensor: any) => {
  const apiKey = node.value?.apiKey
  selectedSensor.value = sensor
  isEditingSensor.value = false
  editingSensorId.value = null
  initialEnvVars.value = {}
  initialEnvVars.value['HW_SEVERITY'] = 'critical'
  initialEnvVars.value['HW_HUB_ENDPOINT'] = configStore.config.hubEndpoint || window.location.origin
  initialEnvVars.value['HW_HUB_KEY'] = apiKey || '<YOUR_HW_NODE_KEY>'

  sensor.deployment?.env_vars?.forEach((env: any) => {
    if (!['HW_HUB_ENDPOINT', 'HW_HUB_KEY', 'HW_SEVERITY'].includes(env.name)) {
      initialEnvVars.value[env.name] = getUIDefault(env.default)
    }
  })
  document.body.style.overflow = 'hidden'
  showSensorModal.value = true
}

const editSensor = (installedSensor: any) => {
  const manifest = getManifestForSensor(installedSensor)
  if (!manifest) {
      alert('Sensor manifest not found')
      return
  }
  
  const apiKey = node.value?.apiKey
  selectedSensor.value = manifest
  isEditingSensor.value = true
  editingSensorId.value = installedSensor.sensorId
  initialEnvVars.value = {}
  initialEnvVars.value['HW_SEVERITY'] = 'critical'
  initialEnvVars.value['HW_HUB_ENDPOINT'] = configStore.config.hubEndpoint || window.location.origin
  initialEnvVars.value['HW_HUB_KEY'] = apiKey || '<YOUR_HW_NODE_KEY>'
  
  manifest.deployment?.env_vars?.forEach((env: any) => {
    if (!['HW_HUB_ENDPOINT', 'HW_HUB_KEY', 'HW_SEVERITY'].includes(env.name)) {
      initialEnvVars.value[env.name] = getUIDefault(env.default)
    }
  })
  
  if (installedSensor.envVars) {
      Object.keys(installedSensor.envVars).forEach(key => {
          initialEnvVars.value[key] = installedSensor.envVars[key]
      })
  }
  
  document.body.style.overflow = 'hidden'
  showSensorModal.value = true
}

const closeSensor = () => {
  showSensorModal.value = false
  selectedSensor.value = null
  isEditingSensor.value = false
  editingSensorId.value = null
  initialEnvVars.value = {}
  document.body.style.overflow = ''
}
</script>

<template>
    <div class="min-h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6 pb-4 sm:pb-6">
        
        <div class="mt-4 sm:mt-6 mb-4 shrink-0">
            <button @click="fleetStore.clearSelection()" class="flex items-center gap-1.5 text-sm font-medium text-text-m hover:text-text-h transition-colors outline-none w-max">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg>
                Back to Fleet
            </button>
        </div>

        <div v-if="node" class="bg-bg-base border border-border-default rounded-lg p-5 sm:p-6 mb-8 shrink-0 shadow-sm flex flex-col gap-6">
            <NodeDetailHeader 
                :node="node" 
                :last-event-time="lastEventTime"
                @update="handleUpdateNode"
                @silence="handleSilenceNode"
                @delete="handleDeleteNode"
                @sync="triggerManualSync"
                @manage-key="showKeyModal = true"
            />

            <NodeStatWidgets :node="node" :recent-activity="recentActivity" @view-all-events="viewAllEvents" />

            <NodeSensorGrid 
                :sensors="node?.installedSensors || []" 
                :manifests="sensors" 
                :isManifestLoading="isManifestLoading"
                @edit="editSensor"
                @toggleSilence="handleToggleSensorSilence"
                @remove="handleRemoveSensor"
            />
        </div>

        <NodeCatalogGrid 
            v-if="node"
            :manifests="sensors" 
            :isLoading="isManifestLoading" 
            :fetchError="fetchError"
            @open="openSensor"
        />

        <NodeKeyModal :show="showKeyModal" :apiKey="node?.apiKey || null" @close="showKeyModal = false" />
        <NodeSyncModal :show="showSyncModal" :syncCommand="syncCommand" :syncComposeYaml="syncComposeYaml" @close="showSyncModal = false" />
        <NodeSensorModal :show="showSensorModal" :sensor="selectedSensor" :isEditing="isEditingSensor" :initialEnvVars="initialEnvVars" :apiKey="node?.apiKey || null" @close="closeSensor" @apply="handleApplySensor" />

    </div>
</template>