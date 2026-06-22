<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFleetStore } from '../stores/Fleet/fleet.ts'
import { useEventsStore } from '../stores/Events/events.ts'
import { useAppStore } from '../stores/System/app.ts'
import { useConfigStore } from '../stores/Config/config.ts'
import type { FleetNode } from '../stores/Fleet/fleet.ts'
import { api } from '../api/client.ts'

import NodeDetailHeader from '../components/nodedetails/NodeDetailHeader.vue'
import NodeStatWidgets from '../components/nodedetails/NodeStatWidgets.vue'
import NodeSensorGrid from '../components/nodedetails/NodeSensorGrid.vue'
import NodeCatalogGrid from '../components/nodedetails/NodeCatalogGrid.vue'
import NodeKeyModal from '../components/nodedetails/NodeKeyModal.vue'
import NodeSyncModal from '../components/nodedetails/NodeSyncModal.vue'
import NodeSensorModal from '../components/nodedetails/NodeSensorModal.vue'

const fleetStore = useFleetStore()
const eventsStore = useEventsStore()
const configStore = useConfigStore()
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()

const selectedNodeId = computed(() => fleetStore.selectedNodeId)

// --- NODE STATE ---
const node = computed(() => fleetStore.enrichedNodes.find(n => n.id === selectedNodeId.value) || null)

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
const manifests = computed(() => fleetStore.manifests)

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
        return `curl -fsSL https://get.honeywire.dev | bash -s -- --link ${hubUrl} --api-key ${node.value.apiKey}`
    }
    return `honeywire apply`
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
    if (!node.value) return
    try {
        await fleetStore.toggleSilence(node.value.id, sensor.sensorId, !sensor.isSilenced)
    } catch (e) {
        // Error handled in store
    }
}

const handleUpgradeSensor = async (sensor: any) => {
    if (!node.value) return
    await fleetStore.upgradeSensor(node.value.id, sensor.sensorId)
}

const handleUpgradeAll = async () => {
    if (!node.value) return
    await fleetStore.upgradeNode(node.value.id)
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
            router.push('/fleet')
        } else {
            alert(res.error)
        }
    }
}

const viewAllEvents = () => {
    // Keeps the current node selected in the fleetStore to act as a filter
    router.push('/dashboard')
}

// --- NAVIGATION ---

watch(() => route.params.id, async (newId, oldId) => {
    if (newId) {
        if (newId !== oldId) {
            eventsStore.clearSummaryProjection()
        }
        fleetStore.selectTarget(newId as string, null, false)
        await Promise.all([
            fleetStore.fetchNodeDetails(newId as string),
            eventsStore.fetchSummaryProjection('24H', newId as string)
        ])
    }
}, { immediate: true })

watch(() => appStore.viewingArchive, async () => {
    if (route.params.id) {
        await eventsStore.fetchSummaryProjection('24H', route.params.id as string)
    }
})

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
    await fleetStore.fetchManifests()
  } catch (error) {
    console.error(error)
    fetchError.value = true
  } finally {
    isManifestLoading.value = false
  }
})

watch(() => node.value?.hasPendingConfig, (newVal) => {
  if (newVal === false && showSyncModal.value) {
    showSyncModal.value = false
  }
})

const openSensor = (sensor: any) => {
  const apiKey = node.value?.apiKey
  selectedSensor.value = sensor
  isEditingSensor.value = false
  editingSensorId.value = null
  initialEnvVars.value = {}
  initialEnvVars.value['HW_HUB_ENDPOINT'] = configStore.config.hubEndpoint || window.location.origin
  initialEnvVars.value['HW_HUB_KEY'] = apiKey || '<YOUR_HW_NODE_KEY>'

  sensor.deployment?.env_vars?.forEach((env: any) => {
    if (!['HW_HUB_ENDPOINT', 'HW_HUB_KEY'].includes(env.name)) {
      initialEnvVars.value[env.name] = getUIDefault(env.default)
    }
  })
  document.body.style.overflow = 'hidden'
  showSensorModal.value = true
}

const editSensor = async (installedSensor: any) => {
  let manifest = manifests.value.find((m: any) => m.id === installedSensor.id || m.id === installedSensor.sensorId || m.id === installedSensor.name)
  
  if (installedSensor.deployedVersion) {
      try {
          const res = await api.get(`/api/v2/manifests/${encodeURIComponent(installedSensor.sensorId || installedSensor.id || installedSensor.name)}/versions?version=${encodeURIComponent(installedSensor.deployedVersion)}`)
          manifest = await res.json()
      } catch (err) {
          console.error("Failed to fetch deployed manifest version schema:", err)
      }
  }

  if (!manifest) {
      alert('Sensor manifest not found')
      return
  }
  
  const apiKey = node.value?.apiKey
  selectedSensor.value = manifest
  isEditingSensor.value = true
  editingSensorId.value = installedSensor.sensorId
  initialEnvVars.value = {}
  initialEnvVars.value['HW_HUB_ENDPOINT'] = configStore.config.hubEndpoint || window.location.origin
  initialEnvVars.value['HW_HUB_KEY'] = apiKey || '<YOUR_HW_NODE_KEY>'
  
  manifest.deployment?.env_vars?.forEach((env: any) => {
    if (!['HW_HUB_ENDPOINT', 'HW_HUB_KEY'].includes(env.name)) {
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
            <button @click="fleetStore.clearSelection(); router.push('/fleet')" class="flex items-center gap-1.5 text-sm font-medium text-text-m hover:text-text-h transition-colors outline-none w-max">
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
                @upgradeAll="handleUpgradeAll"
            />

            <NodeStatWidgets :node="node" :recent-activity="recentActivity" @view-all-events="viewAllEvents" />

            <NodeSensorGrid 
                :sensors="node?.installedSensors || []" 
                @edit="editSensor"
                @toggleSilence="handleToggleSensorSilence"
                @remove="handleRemoveSensor"
                @upgrade="handleUpgradeSensor"
            />
        </div>

        <NodeCatalogGrid 
            v-if="node"
            :manifests="manifests" 
            :isLoading="isManifestLoading" 
            :fetchError="fetchError"
            :installedSensors="node?.installedSensors || []"
            @open="openSensor"
            @edit="editSensor"
        />

        <NodeKeyModal :show="showKeyModal" :apiKey="node?.apiKey || null" @close="showKeyModal = false" />
        <NodeSyncModal :show="showSyncModal" :syncCommand="syncCommand" :syncComposeYaml="syncComposeYaml" @close="showSyncModal = false" />
        <NodeSensorModal :show="showSensorModal" :sensor="selectedSensor" :isEditing="isEditingSensor" :initialEnvVars="initialEnvVars" :apiKey="node?.apiKey || null" @close="closeSensor" @apply="handleApplySensor" />

    </div>
</template>