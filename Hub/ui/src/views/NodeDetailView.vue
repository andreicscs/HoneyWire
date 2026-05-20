<script setup>
import { ref, watch, computed, onMounted, nextTick } from 'vue'
import { useAppStore } from '../stores/app'
import { useFleetStore } from '../stores/fleet'
import { useEventsStore } from '../stores/events'
import { useConfig } from '../api/useConfig'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import BaseInput from '../components/ui/forms/BaseInput.vue'
import BaseStatusDot from '../components/ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../components/ui/navigation/BaseMeatballMenu.vue'
import BaseModal from '../components/ui/feedback/BaseModal.vue'

const appStore = useAppStore()
const fleetStore = useFleetStore()
const eventsStore = useEventsStore()
const { config } = useConfig()

const selectedNodeId = computed(() => fleetStore.selectedNode)

// --- NODE STATE ---
// Single source of truth: the live reactive object from the store.
const node = computed(() => fleetStore.getNode(selectedNodeId.value))

const maxSensorEvents = computed(() => {
  if (!node.value?.installedSensors?.length) return 1
  return Math.max(...node.value.installedSensors.map(s => s.events24h || 0), 1)
})

const sortedInstalledSensors = computed(() => {
  return [...(node.value?.installedSensors || [])]
    .sort((a, b) => (b.events24h || 0) - (a.events24h || 0))
})

// --- SENSOR CATALOG STATE ---
const isManifestLoading = ref(true)
const fetchError = ref(false)
const sensors = ref([])

// --- SENSOR MODAL STATE ---
const selectedSensor = ref(null)
const activeTab = ref('readme')
const envVarValues = ref({})
const activeEnvVar = ref(null)
const isSeverityOpen = ref(false)
const rawCompose = ref('')
const highlightedCompose = ref('')
const composePre = ref(null)

// --- MODAL STATE ---
const showKeyModal = ref(false)
const showSyncModal = ref(false)
const syncComposeYaml = ref('')
const showEditModal = ref(false)
const editNodeForm = ref({ id: '', alias: '', publicIp: '', privateIp: '' })
const tagInput = ref('')
const tagsList = ref([])

// --- SEVERITY CONFIG ---
const severityOptions = [
  { value: 'info', label: 'Info', textClass: 'text-info', hoverClass: 'hover:bg-info/10 hover:text-info' },
  { value: 'low', label: 'Low', textClass: 'text-low', hoverClass: 'hover:bg-low/10 hover:text-low' },
  { value: 'medium', label: 'Medium', textClass: 'text-medium', hoverClass: 'hover:bg-medium/10 hover:text-medium' },
  { value: 'high', label: 'High', textClass: 'text-high', hoverClass: 'hover:bg-high/10 hover:text-high' },
  { value: 'critical', label: 'Critical', textClass: 'text-critical', hoverClass: 'hover:bg-critical/10 hover:text-critical' }
]

const currentSeverity = computed(() =>
  severityOptions.find(s => s.value === envVarValues.value['HW_SEVERITY']) || severityOptions[3]
)

const toggleSeverity = () => {
  isSeverityOpen.value = !isSeverityOpen.value
  activeEnvVar.value = isSeverityOpen.value ? 'HW_SEVERITY' : null
}
const closeSeverity = () => { isSeverityOpen.value = false; activeEnvVar.value = null }
const selectSeverity = (val) => { envVarValues.value['HW_SEVERITY'] = val; closeSeverity() }

// --- ENV VAR HELPERS ---
const getUIDefault = (def) => {
  if (!def) return ''
  if (!def.includes('{{')) return def
  const elseMatch = def.match(/\{\{\s*else\s*\}\}(.*?)\{\{\s*end\s*\}\}/)
  if (elseMatch) return elseMatch[1].trim()
  const funcMatch = def.match(/\{\{\s*[a-zA-Z]+\s+([0-9]+)\s*\}\}/)
  if (funcMatch) return funcMatch[1].trim()
  return ''
}

const coreVars = ['HW_HUB_ENDPOINT', 'HW_HUB_KEY', 'HW_SENSOR_ID', 'HW_SEVERITY', 'HW_TEST_MODE', 'HW_LOG_LEVEL']

const sortedEnvVars = computed(() => {
  if (!selectedSensor.value?.deployment?.env_vars) return []
  return [...selectedSensor.value.deployment.env_vars]
    .filter(env => !env.hidden)
    .sort((a, b) => {
      const aIsCore = coreVars.includes(a.name)
      const bIsCore = coreVars.includes(b.name)
      if (aIsCore && !bIsCore) return -1
      if (!aIsCore && bIsCore) return 1
      if (aIsCore && bIsCore) return coreVars.indexOf(a.name) - coreVars.indexOf(b.name)
      return a.name.localeCompare(b.name)
    })
})

// --- SENSOR ACTIONS (delegated to store) ---

const handleToggleSensorSilence = async (sensor) => {
  if (!node.value?.id || !sensor.id) return
  try {
    await fleetStore.toggleSilence(node.value.id, sensor.id, !sensor.isSilenced)
  } catch (err) {
    alert('Unable to change sensor silence state. Please try again.')
  }
}

const handleRemoveSensor = async (sensor) => {
  if (!node.value?.id || !sensor.id) return
  try {
    await fleetStore.removeSensor(node.value.id, sensor.id)
  } catch (err) {
    alert('Could not remove sensor. Please try again.')
  }
}

const handleAddSensorToNode = async () => {
  if (!selectedSensor.value || !node.value?.id) return

  const safeEnvValues = Object.fromEntries(
    Object.entries(envVarValues.value).map(([k, v]) => [k, v !== undefined && v !== null ? String(v) : ''])
  )

  try {
    await fleetStore.addSensor(node.value.id, {
      sensorId: selectedSensor.value.id || selectedSensor.value.sensor_id || selectedSensor.value.name,
      customName: selectedSensor.value.name || selectedSensor.value.id,
      configValues: safeEnvValues,
    })
    closeSensor()
  } catch (err) {
    alert('Could not add sensor to this node. Please try again.')
  }
}

// --- NODE ACTIONS (delegated to store) ---

const handleManageKey = () => { showKeyModal.value = true }

const copyNodeKeyToClipboard = async () => {
  const apiKey = node.value?.apiKey
  if (!apiKey) return
  try {
    await navigator.clipboard.writeText(apiKey)
    alert('Node API Key copied to clipboard')
  } catch (err) {
    alert('Unable to copy Node API Key. Please copy it manually.')
  }
}

const copySyncYamlToClipboard = async () => {
  if (!syncComposeYaml.value) return
  try {
    await navigator.clipboard.writeText(syncComposeYaml.value)
    alert('Compose YAML copied to clipboard')
  } catch (err) {
    alert('Unable to copy compose YAML. Please copy it manually.')
  }
}

const triggerManualSync = async () => {
  if (!node.value?.id) return
  try {
    syncComposeYaml.value = await fleetStore.syncNode(node.value.id)
    showSyncModal.value = true
  } catch (err) {
    alert('Unable to sync this node. Please try again.')
  }
}

// --- NAVIGATION ---

watch(selectedNodeId, async (value) => {
  if (!value) {
    appStore.currentView = 'fleet'
    return
  }
  await fleetStore.fetchNodeDetails(value)
  eventsStore.fetchEvents(false, value)
}, { immediate: true })

const timeAgo = (dateStr) => {
  if (!dateStr) return 'Unknown'
  const diff = Math.floor((new Date() - new Date(dateStr)) / 1000)
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}

const recentActivity = computed(() => {
  return eventsStore.filteredEvents.slice(0, 10).map(e => ({
    id: e.id,
    time: timeAgo(e.timestamp),
    severity: e.severity || 'info',
    event_trigger: e.event_trigger || 'Alert',
    source: e.source || 'Unknown'
  }))
})

// --- EDIT MODAL ---

const openEditModal = () => {
  if (!node.value) return
  editNodeForm.value = {
    id: node.value.id,
    alias: node.value.alias,
    publicIp: node.value.publicIp || '',
    privateIp: node.value.privateIp || ''
  }
  tagsList.value = [...(node.value.tags || [])]
  showEditModal.value = true
}

const addTag = () => {
  const val = tagInput.value.trim()
  if (val && !tagsList.value.includes(val)) tagsList.value.push(val)
  tagInput.value = ''
}

const removeTag = (index) => { tagsList.value.splice(index, 1) }

const handleEditSubmit = async () => {
  try {
    await fleetStore.updateNode(editNodeForm.value.id, {
      alias: editNodeForm.value.alias,
      tags: tagsList.value,
      publicIp: editNodeForm.value.publicIp,
      privateIp: editNodeForm.value.privateIp,
    })
    showEditModal.value = false
  } catch (err) {
    alert('Failed to update node. Please try again.')
  }
}

// --- SENSOR CATALOG ---

watch(envVarValues, () => { fetchYamlFromHub() }, { deep: true })
watch(activeEnvVar, () => { applyHighlighting() })

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

const openSensor = (sensor) => {
  const apiKey = node.value?.apiKey
  selectedSensor.value = sensor
  activeTab.value = 'readme'
  envVarValues.value = {}
  envVarValues.value['HW_SEVERITY'] = 'critical'
  envVarValues.value['HW_HUB_ENDPOINT'] = config.hubEndpoint || window.location.origin
  envVarValues.value['HW_HUB_KEY'] = apiKey || '<YOUR_HW_NODE_KEY>'
  sensor.deployment?.env_vars?.forEach(env => {
    if (!['HW_HUB_ENDPOINT', 'HW_HUB_KEY', 'HW_SEVERITY'].includes(env.name)) {
      envVarValues.value[env.name] = getUIDefault(env.default)
    }
  })
  document.body.style.overflow = 'hidden'
  fetchYamlFromHub()
}

const closeSensor = () => {
  selectedSensor.value = null
  envVarValues.value = {}
  activeEnvVar.value = null
  document.body.style.overflow = ''
}

const fetchYamlFromHub = async () => {
  if (!selectedSensor.value || !node.value?.id) return

  const apiKey = node.value.apiKey
  const safeEnvValues = Object.fromEntries(
    Object.entries(envVarValues.value).map(([k, v]) => [k, v !== undefined && v !== null ? String(v) : ''])
  )

  try {
    rawCompose.value = await fleetStore.generateCompose({
      node_id: node.value.id,
      hub_endpoint: config.hubEndpoint || window.location.origin,
      hub_key: apiKey || '<YOUR_HW_NODE_KEY>',
      sensors: [{
        sensor_id: selectedSensor.value.id,
        env_values: safeEnvValues,
        manifest: selectedSensor.value
      }]
    })
    applyHighlighting()
  } catch (e) {
    rawCompose.value = 'services:\n  error:\n    error_generating_yaml'
    highlightedCompose.value = rawCompose.value
  }
}

const applyHighlighting = () => {
  let htmlYaml = rawCompose.value
  if (activeEnvVar.value) {
    const escapedName = activeEnvVar.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    const regex = new RegExp(`^.*\\b${escapedName}\\b.*$`, 'gm')
    htmlYaml = htmlYaml.replace(regex, `<span class="bg-highlight-bg text-highlight-text ring-1 ring-highlight-ring px-1 rounded transition-colors duration-[var(--duration-fast)] active-highlight">$&</span>`)
  }
  highlightedCompose.value = htmlYaml
  nextTick(() => {
    if (composePre.value) {
      const highlightEl = composePre.value.querySelector('.active-highlight')
      if (highlightEl) {
        const scrollPos = highlightEl.offsetTop - (composePre.value.clientHeight / 2) + (highlightEl.clientHeight / 2)
        composePre.value.scrollTo({ top: Math.max(0, scrollPos), behavior: 'smooth' })
      }
    }
  })
}
</script>

<template>
    <div class="h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6 pb-10 overflow-y-auto custom-scroll">
        
        <div class="mt-4 sm:mt-6 mb-4 shrink-0">
            <button @click="appStore.currentView = 'fleet'" class="flex items-center gap-1.5 text-sm font-medium text-text-m hover:text-text-h transition-colors outline-none w-max">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg>
                Back to Fleet
            </button>
        </div>

        <div v-if="node" class="bg-bg-base border border-border-default rounded-[var(--radius-lg)] p-5 sm:p-6 mb-8 shrink-0 shadow-sm flex flex-col gap-6">
            
            <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
                <div>
                    <div class="flex items-center gap-3 mb-3">
                        <h1 class="text-[length:var(--text-h1)] font-semibold text-text-h leading-tight">{{ node.alias }}</h1>
                        <BaseStatusDot :status="node.status" />
                    </div>
                    
                    <div class="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-text-m">
                        <div class="flex items-center gap-1.5" title="Public IP">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                            <span class="font-mono">{{ node.publicIp || 'Unknown' }}</span>
                        </div>
                        <div class="flex items-center gap-1.5" title="Private IP">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="2" y="14" width="8" height="6" rx="2" ry="2"/><rect x="14" y="14" width="8" height="6" rx="2" ry="2"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 14v-2a2 2 0 012-2h8a2 2 0 012 2v2M12 2v8"/><rect x="8" y="2" width="8" height="6" rx="2" ry="2"/></svg>
                            <span class="font-mono">{{ node.privateIp || 'Unknown' }}</span>
                        </div>
                        <div class="h-4 w-px bg-border-default hidden sm:block"></div>
                        <div class="flex items-center gap-1.5">
                            <span class="text-text-h font-medium">Last Event:</span> {{ node.lastEvent }}
                        </div>
                        <div class="h-4 w-px bg-border-default hidden sm:block"></div>
                        <div class="flex items-center gap-1.5 flex-wrap">
                            <span v-for="tag in node.tags" :key="tag" class="px-2 py-0.5 bg-bg-inset border border-border-default text-text-m text-[10px] font-medium rounded-md tracking-wider">{{ tag }}</span>
                            <button @click.stop="openEditModal" class="px-1.5 py-0.5 border border-dashed border-border-default text-text-m text-[10px] rounded-md hover:text-text-h transition-colors">
                                + Tag
                            </button>
                        </div>
                    </div>
                </div>

                <div class="flex items-center gap-3 shrink-0">
                    <BaseButton variant="secondary" class="!py-1.5 !px-3 !text-sm flex items-center gap-2" @click="handleManageKey">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/></svg>
                        Manage Key
                    </BaseButton>
                    <BaseMeatballMenu id="node-super-menu">
                        <button @click="openEditModal" class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">Rename / Edit Node</button>
                        <button class="w-full text-left px-3 py-2 text-sm text-danger-text hover:bg-danger-bg transition-colors border-t border-border-default mt-1 pt-2">Delete Node</button>
                    </BaseMeatballMenu>
                </div>
            </div>

            <div v-if="node.hasPendingConfig" class="flex items-center justify-between bg-high/10 border border-high/30 rounded-lg p-4 transition-all duration-normal">
                <div class="flex items-start gap-3">
                    <svg class="w-5 h-5 text-high mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                    <div>
                        <h4 class="text-sm font-semibold text-high">Pending Sync</h4>
                        <p class="text-sm text-text-h opacity-90 mt-0.5">Sensors have been added or modified. Sync this node to apply changes.</p>
                    </div>
                </div>
                <BaseButton @click="triggerManualSync" variant="secondary" class="!border-high/30 !bg-bg-surface hover:!bg-high/10 !text-high shrink-0">Sync Node</BaseButton>
            </div>

            <div class="grid grid-cols-1 xl:grid-cols-2 gap-5">
                
                <div class="bg-bg-surface border border-border-default rounded-lg p-5 shadow-sm">
                    <h3 class="text-sm font-semibold text-text-h mb-5">Sensor Volume (24h)</h3>
                    <div v-if="sortedInstalledSensors.length > 0" class="space-y-4">
                        <div v-for="sensor in sortedInstalledSensors" :key="sensor.id">
                            <div class="flex items-center justify-between mb-1.5">
                                <span class="text-sm font-medium text-text-h truncate pr-4">{{ sensor.display }}</span>
                                <span class="text-sm font-mono text-text-m">{{ sensor.events24h }}</span>
                            </div>
                            <div class="w-full bg-bg-inset border border-border-default rounded-full h-2">
                                <div class="bg-text-m h-full rounded-full transition-all duration-normal" :style="`width: ${(sensor.events24h / maxSensorEvents) * 100}%`"></div>
                            </div>
                        </div>
                    </div>
                    <div v-else class="text-sm text-text-m italic">No events recorded.</div>
                </div>

                <div class="bg-bg-surface border border-border-default rounded-lg flex flex-col overflow-hidden shadow-sm">
                    <div class="px-5 py-3 border-b border-border-default flex items-center justify-between bg-bg-surface shrink-0">
                        <h3 class="text-sm font-semibold text-text-h">Recent Activity</h3>
                        <button class="text-xs font-medium text-text-m hover:text-text-h transition-colors">View All &rarr;</button>
                    </div>
                    
                    <div class="flex-1 overflow-y-auto custom-scroll bg-bg-surface">
                        <table class="w-full text-left border-collapse">
                            <tbody>
                                <tr v-if="recentActivity.length === 0">
                                    <td colspan="4" class="px-5 py-6 text-center text-sm text-text-m">No recent activity on this node.</td>
                                </tr>
                                <tr v-for="event in recentActivity" :key="event.id" 
                                    class="hover:bg-secondary-hover cursor-pointer transition-colors duration-[var(--duration-fast)] relative z-0 group"
                                    :class="'bleed-' + event.severity.toLowerCase()">
                                    
                                    <td class="px-3 py-2.5 border-b border-border-default border-l-[3px]"
                                        :style="{ borderLeftColor: `var(--color-${event.severity.toLowerCase()})` }">
                                        <span class="px-2 py-0.5 rounded border text-xs font-medium bg-bg-surface whitespace-nowrap capitalize flex justify-center w-max" 
                                              :style="{ borderColor: `var(--color-${event.severity.toLowerCase()})`, color: `var(--color-${event.severity.toLowerCase()})` }">
                                            {{ event.severity }}
                                        </span>
                                    </td>
                                    
                                    <td class="px-3 py-2.5 border-b border-border-default w-full max-w-0">
                                        <div class="text-sm text-text-h font-medium truncate">{{ event.event_trigger }}</div>
                                    </td>
                                    
                                    <td class="px-3 py-2.5 border-b border-border-default whitespace-nowrap">
                                        <div class="flex items-center gap-1.5">
                                            <span class="text-[9px] uppercase font-bold text-text-m tracking-wider bg-bg-inset border border-border-default/50 px-1 rounded">SRC</span>
                                            <span class="text-xs text-text-m font-mono truncate max-w-[120px]">{{ event.source }}</span>
                                        </div>
                                    </td>
                                    
                                    <td class="px-4 py-2.5 border-b border-border-default text-right">
                                        <span class="text-xs text-text-m font-mono whitespace-nowrap">{{ event.time }}</span>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>

            <div>
                <h3 class="text-sm font-semibold text-text-h mb-4 mt-2">Deployed Sensors</h3>
                <div v-if="node.installedSensors.length > 0" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-5 gap-4">
                    <div v-for="sensor in node.installedSensors" :key="sensor.id" class="bg-bg-surface border border-border-default rounded-lg p-4 flex flex-col group hover:border-text-m transition-colors shadow-sm relative overflow-hidden">
                        
                        <div class="absolute top-0 left-0 right-0 h-1 transition-colors" :class="sensor.status === 'up' ? 'bg-success-main' : 'bg-danger-main'"></div>

                        <div class="flex justify-between items-start mt-1">
                            <div class="flex items-center gap-3 min-w-0">
                                <div class="w-8 h-8 rounded bg-bg-inset border border-border-default/50 flex items-center justify-center text-text-h shrink-0">
                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="sensor.icon"></path></svg>
                                </div>
                                <div class="min-w-0">
                                    <h4 class="text-sm font-semibold text-text-h truncate">{{ sensor.display }}</h4>
                                    <span class="text-xs font-mono text-text-m block truncate">{{ sensor.name }}</span>
                                </div>
                            </div>
                            <BaseMeatballMenu :id="`sensor-menu-${sensor.id}`">
                                <button class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">Edit Configuration</button>
                                <button @click="handleToggleSensorSilence(sensor)" class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">
                                    {{ sensor.isSilenced ? 'Unsilence Alerts' : 'Silence Alerts' }}
                                </button>
                                <button @click="handleRemoveSensor(sensor)" class="w-full text-left px-3 py-2 text-sm text-danger-text hover:bg-danger-bg transition-colors border-t border-border-default mt-1 pt-2">Remove Sensor</button>
                            </BaseMeatballMenu>
                        </div>
                        <div class="mt-3 pt-3 border-t border-border-default flex justify-between items-center">
                             <span class="px-1.5 py-0.5 rounded text-[9px] font-medium tracking-wider bg-bg-inset text-text-m border border-border-default/50 uppercase">{{ sensor.osi }}</span>
                             <svg v-if="sensor.isSilenced" class="w-3.5 h-3.5 text-medium shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" title="Alerts Silenced"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/></svg>
                        </div>
                    </div>
                </div>
                <div v-else class="border border-dashed border-border-default rounded-lg p-8 flex flex-col items-center justify-center text-center bg-bg-surface/50">
                    <p class="text-sm text-text-h font-medium">No sensors deployed</p>
                    <p class="text-xs text-text-m mt-1">Select a sensor from the catalog below.</p>
                </div>
            </div>
        </div>

        <div class="shrink-0 mb-6">
            <h2 class="text-lg font-semibold text-text-h mb-4">Sensor Catalog</h2>
            <div v-if="isManifestLoading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                <div v-for="i in 4" :key="i" class="bg-bg-surface border border-border-default rounded-lg p-5 h-36 animate-pulse"></div>
            </div>
            <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                <div v-for="s in sensors" :key="s.id" @click="openSensor(s)" class="bg-bg-surface border border-border-default rounded-lg p-4 shadow-sm hover:border-primary-main hover:shadow-md cursor-pointer transition-all duration-normal group flex flex-col">
                    <div class="flex justify-between items-start mb-3">
                        <div class="w-10 h-10 rounded-md bg-bg-base border border-border-default/50 text-text-h flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform duration-normal">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="s.icon_svg"></path></svg>
                        </div>
                        <span class="px-2 py-0.5 rounded text-[10px] font-medium tracking-wider bg-bg-inset text-text-m border border-border-default/50">{{ s.osi_layer }}</span>
                    </div>
                    <h3 class="text-sm font-semibold text-text-h mb-1">{{ s.name }}</h3>
                    <p class="text-xs text-text-m leading-relaxed line-clamp-2">{{ s.description }}</p>
                </div>
            </div>
        </div>

        <BaseModal :show="showKeyModal" @close="showKeyModal = false" title="Manage Node Key">
            <div class="space-y-4">
                <p class="text-sm text-text-m">This is the unique API key for this node. It is used to authenticate the node with the hub.</p>
                <div class="bg-bg-surface border border-border-default rounded-lg p-4 font-mono text-sm break-all">
                    <div class="text-text-h font-semibold mb-2">Node API Key</div>
                    <div>{{ node.apiKey || 'Unavailable' }}</div>
                </div>
                <div class="flex justify-end gap-2">
                    <BaseButton variant="secondary" @click="showKeyModal = false">Close</BaseButton>
                    <BaseButton variant="primary" @click="copyNodeKeyToClipboard">Copy API Key</BaseButton>
                </div>
            </div>
        </BaseModal>

        <BaseModal :show="showSyncModal" @close="showSyncModal = false" title="Node Compose YAML">
            <div class="space-y-4">
                <p class="text-sm text-text-m">This is the generated docker-compose.yml for this node. Use it to deploy the node with the selected sensors and config.</p>
                <div class="relative bg-bg-surface border border-border-default rounded-lg p-4 font-mono text-xs text-text-h overflow-auto max-h-[40vh]">
                    <pre class="whitespace-pre-wrap break-words">{{ syncComposeYaml || 'No compose output available.' }}</pre>
                </div>
                <div class="flex justify-end gap-2">
                    <BaseButton variant="secondary" @click="showSyncModal = false">Close</BaseButton>
                    <BaseButton variant="primary" @click="copySyncYamlToClipboard">Copy Compose</BaseButton>
                </div>
            </div>
        </BaseModal>

        <BaseModal :show="showEditModal" @close="showEditModal = false" title="Node Settings">
            <div class="space-y-4">
                <div>
                    <label class="block text-sm font-medium text-text-h mb-1.5">Node Alias</label>
                    <BaseInput v-model="editNodeForm.alias" />
                </div>
                
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label class="block text-sm font-medium text-text-h mb-1.5">Public IP</label>
                        <BaseInput v-model="editNodeForm.publicIp" placeholder="e.g. 203.0.113.5" />
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-text-h mb-1.5">Private IP</label>
                        <BaseInput v-model="editNodeForm.privateIp" placeholder="e.g. 10.0.1.50" />
                    </div>
                </div>
                
                <div>
                    <label class="block text-sm font-medium text-text-h mb-1.5">Tags</label>
                    <div class="flex flex-col gap-2.5">
                        <div v-if="tagsList.length > 0" class="flex flex-wrap gap-1.5">
                            <span v-for="(tag, index) in tagsList" :key="tag" 
                                class="flex items-center gap-1.5 px-2 py-1 bg-bg-inset border border-border-default text-text-m text-xs font-medium rounded-md tracking-wider">
                                {{ tag }}
                                <button @click="removeTag(index)" class="text-text-m hover:text-danger-text transition-colors outline-none">
                                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                                </button>
                            </span>
                        </div>
                        <BaseInput v-model="tagInput" @keydown.enter.prevent="addTag" placeholder="Type a tag and press Enter..." />
                    </div>
                </div>

                <div class="flex justify-end pt-5 border-t border-border-default mt-6">
                    <BaseButton variant="secondary" class="mr-2" @click="showEditModal = false">Cancel</BaseButton>
                    <BaseButton variant="primary" @click="handleEditSubmit">Save Changes</BaseButton>
                </div>
            </div>
        </BaseModal>

        <Teleport to="body">
            <transition enter-active-class="transition duration-normal ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-[var(--duration-fast)] ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <div v-if="selectedSensor" class="fixed inset-0 z-[var(--z-modal)] flex justify-center items-center p-4 sm:p-6 bg-black/60 backdrop-blur-sm" @click.self="closeSensor">
                    
                    <div class="bg-bg-base w-full max-w-4xl h-full max-h-[85vh] rounded-[var(--radius-lg)] shadow-2xl flex flex-col overflow-hidden border border-border-default transform transition-all">
                        
                        <div class="px-6 py-5 border-b border-border-default flex justify-between items-start bg-bg-surface shrink-0">
                            <div class="flex items-center gap-4">
                                <div class="w-12 h-12 rounded-md bg-bg-inset border border-border-default/50 text-text-h flex items-center justify-center shrink-0 shadow-sm">
                                    <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="selectedSensor.icon_svg"></path></svg>
                                </div>
                                <div>
                                    <h2 class="text-xl font-bold text-text-h">{{ selectedSensor.name }}</h2>
                                    <p class="text-sm text-text-m mt-0.5">{{ selectedSensor.description }}</p>
                                </div>
                            </div>
                            <button @click="closeSensor" class="p-2 -mr-2 text-text-m hover:text-text-h transition-colors duration-[var(--duration-fast)] rounded-full hover:bg-secondary-hover focus:outline-none">
                                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path></svg>
                            </button>
                        </div>

                        <div class="flex border-b border-border-default px-6 shrink-0 bg-bg-base">
                            <button @click="activeTab = 'readme'" class="py-3 px-2 mr-6 text-sm border-b-2 transition-colors duration-[var(--duration-fast)] focus:outline-none" :class="activeTab === 'readme' ? 'border-primary-main text-text-h font-semibold' : 'border-transparent text-text-m hover:text-text-h'">Overview</button>
                            <button @click="activeTab = 'config'" class="py-3 px-2 text-sm border-b-2 transition-colors duration-[var(--duration-fast)] focus:outline-none" :class="activeTab === 'config' ? 'border-primary-main text-text-h font-semibold' : 'border-transparent text-text-m hover:text-text-h'">Configuration</button>
                        </div>

                        <div class="flex-1 overflow-y-auto custom-scroll bg-bg-base">
                            <div v-show="activeTab === 'readme'" class="p-6 md:p-8 readme-container text-text-m text-sm">
                                <p class="mb-6 text-sm font-medium text-text-h">{{ selectedSensor.documentation?.summary }}</p>
                                <div v-for="section in selectedSensor.documentation?.sections" :key="section.title" class="mb-6">
                                    <h3 class="text-sm font-semibold text-text-h mb-3">{{ section.title }}</h3>
                                    <ul v-if="section.type === 'list'" class="list-disc pl-5 space-y-1">
                                        <li v-for="item in section.content" :key="item">{{ item }}</li>
                                    </ul>
                                </div>
                            </div>

                            <div v-show="activeTab === 'config'" class="p-6 md:p-8 relative h-full flex flex-col">
                                <div class="mb-6">
                                    <h4 class="text-sm font-semibold text-text-h mb-3">Sensor Settings</h4>
                                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                        
                                        <div class="space-y-1 relative">
                                            <label class="block text-sm text-text-h mb-0.5">Alert Severity</label>
                                            <div @click="toggleSeverity" class="w-full px-3 py-2 text-sm bg-input-bg border rounded-md cursor-pointer flex justify-between items-center transition-all duration-[var(--duration-fast)] shadow-sm select-none" :class="isSeverityOpen ? 'border-primary-main ring-1 ring-focus-ring' : 'border-input-border hover:border-border-default'">
                                                <span :class="currentSeverity.textClass" class="font-medium flex items-center gap-2"><span class="w-2 h-2 rounded-full" :class="currentSeverity.textClass.replace('text-', 'bg-')"></span>{{ currentSeverity.label }}</span>
                                                <svg class="w-4 h-4 text-text-m" :class="isSeverityOpen ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                                            </div>
                                            <div v-if="isSeverityOpen" @click="closeSeverity" class="fixed inset-0 z-[var(--z-elevated)]"></div>
                                            <transition enter-active-class="transition duration-100 ease-out" enter-from-class="transform scale-95 opacity-0" enter-to-class="transform scale-100 opacity-100" leave-active-class="transition duration-75 ease-in" leave-from-class="transform scale-100 opacity-100" leave-to-class="transform scale-95 opacity-0">
                                                <ul v-if="isSeverityOpen" class="absolute z-[var(--z-dropdown)] w-full mt-1 bg-bg-surface border border-border-default rounded-md shadow-lg py-1 overflow-hidden">
                                                    <li v-for="option in severityOptions" :key="option.value" @click="selectSeverity(option.value)" class="px-3 py-2 cursor-pointer transition-colors text-sm font-medium duration-[var(--duration-fast)] flex items-center gap-2" :class="[option.textClass, option.hoverClass]">
                                                        <span class="w-2 h-2 rounded-full" :class="option.textClass.replace('text-', 'bg-')"></span>
                                                        {{ option.label }}
                                                    </li>
                                                </ul>
                                            </transition>
                                        </div>

                                        <template v-for="env in sortedEnvVars" :key="env.name">
                                            <div v-if="env.name !== 'HW_SEVERITY'" class="space-y-1">
                                                <label class="block text-sm text-text-h mb-0.5">{{ env.name }}</label>
                                                <input v-model="envVarValues[env.name]" :type="env.type === 'int' ? 'number' : 'text'" :placeholder="getUIDefault(env.default)" @focus="activeEnvVar = env.name" @blur="activeEnvVar = null" class="w-full px-3 py-2 text-sm text-text-h bg-input-bg border border-input-border rounded-md focus:outline-none focus:border-primary-main focus:ring-1 focus:ring-focus-ring transition-all duration-[var(--duration-fast)] shadow-sm placeholder:text-text-m/50" />
                                                <p class="text-[11px] text-text-m mt-1">{{ env.description }}</p>
                                            </div>
                                        </template>

                                    </div>
                                </div>
                                
                                <div class="relative flex-1 min-h-[250px] mb-6">
                                    <pre ref="composePre" v-html="highlightedCompose" class="absolute inset-0 w-full h-full bg-bg-surface text-text-m p-4 rounded-md text-xs font-mono custom-scroll border border-border-default leading-relaxed overflow-auto focus:outline-none scroll-smooth shadow-inner"></pre>
                                </div>

                                <div class="mt-auto border-t border-border-default pt-4 flex justify-end">
                                    <BaseButton variant="primary" @click="handleAddSensorToNode" class="px-6">Add to Node</BaseButton>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </transition>
        </Teleport>

    </div>
</template>

<style scoped>
.readme-container :deep(h3) { font-size: var(--text-base); font-weight: var(--font-weight-medium); color: var(--text-h); margin-top: 1.5rem; margin-bottom: 0.75rem; }
.readme-container :deep(p) { line-height: 1.6; margin-bottom: 1rem; }
.readme-container :deep(code) { font-family: var(--font-mono); background-color: var(--input-bg); color: var(--text-h); padding: 0.1rem 0.3rem; border-radius: var(--radius-sm); font-size: var(--text-sm); border: 1px solid var(--input-border); }
</style>