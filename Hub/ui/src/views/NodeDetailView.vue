<script setup>
import { ref, watch, computed, onMounted, nextTick } from 'vue'
import { useAppStore } from '../stores/System/app'
import { useFleetStore } from '../stores/Fleet/fleet'
import { useEventsStore } from '../stores/Events/events'
import { useConfig } from '../api/useConfig'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import BaseStatusDot from '../components/ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../components/ui/navigation/BaseMeatballMenu.vue'
import BaseModal from '../components/ui/feedback/BaseModal.vue'
import { formatSensorId } from '../utils/formatSensorId'
import { useClipboard } from '../utils/useClipboard'


const appStore = useAppStore()
const fleetStore = useFleetStore()
const eventsStore = useEventsStore()
const { config } = useConfig()

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

const maxSensorEvents = computed(() => {
  if (!node.value?.installedSensors?.length) return 1
  return Math.max(...node.value.installedSensors.map(s => s.events24h || 0), 1)
})

const topSensors = computed(() => {
  return [...(node.value?.installedSensors || [])]
    .sort((a, b) => (b.events24h || 0) - (a.events24h || 0))
    .slice(0, 5)
})

// --- MANIFEST LOOKUP (for icon/OSI enrichment) ---
const isManifestLoading = ref(true)
const fetchError = ref(false)
const sensors = ref([])
const manifestMap = computed(() => {
  const map = new Map()
  for (const s of sensors.value) {
    map.set(s.id, s)
    map.set(s.sensorId, s)
    map.set(s.name, s)
  }
  return map
})

const getManifestForSensor = (installedSensor) => {
  const manifest = manifestMap.value.get(installedSensor.id)
    || manifestMap.value.get(installedSensor.name)
    || manifestMap.value.get(installedSensor.sensorId)
  return manifest
}

const sensorIcon = (installedSensor) => {
  const manifest = getManifestForSensor(installedSensor)
  return manifest?.icon_svg || installedSensor.icon || ''
}

const sensorOsi = (installedSensor) => {
  const manifest = getManifestForSensor(installedSensor)
  return manifest?.osi_layer || installedSensor.osi || ''
}

// --- INLINE RENAME STATE (ephemeral UI) ---
const editingAliasNodeId = ref(null)
const rawAliasValue = ref('')
const aliasInputRefs = ref({})

const enableAliasEdit = async (n) => {
    editingAliasNodeId.value = n.id
    rawAliasValue.value = n.alias
    await nextTick()
    if (aliasInputRefs.value[n.id]) {
        aliasInputRefs.value[n.id].focus()
        aliasInputRefs.value[n.id].select()
    }
}

const cancelAliasEdit = () => {
    editingAliasNodeId.value = null
    rawAliasValue.value = ''
}

const saveAlias = async (n) => {
    if (editingAliasNodeId.value !== n.id) return
    const val = rawAliasValue.value.trim()
    if (val && val !== n.alias) {
        try {
            await fleetStore.updateNode(n.id, {
                alias: val,
                tags: n.tags,
                publicIp: n.publicIp,
                privateIp: n.privateIp,
            })
        } catch (err) {
            // Store handles rollback
        }
    }
    editingAliasNodeId.value = null
    rawAliasValue.value = ''
}

// --- INLINE TAGS STATE (ephemeral UI) ---
const editingTagNodeId = ref(null)
const newTagValue = ref('')
const tagInputRefs = ref({})

const enableTagEdit = async (nodeId) => {
    editingTagNodeId.value = nodeId
    await nextTick()
    if (tagInputRefs.value[nodeId]) {
        tagInputRefs.value[nodeId].focus()
    }
}

const cancelTag = () => {
    editingTagNodeId.value = null
    newTagValue.value = ''
}

const saveTag = async (n) => {
    const val = newTagValue.value.trim()
    if (val && !n.tags.includes(val)) {
        try {
            await fleetStore.updateNode(n.id, {
                alias: n.alias,
                tags: [...n.tags, val],
                publicIp: n.publicIp,
                privateIp: n.privateIp,
            })
        } catch (err) {
            // Store handles rollback
        }
    }
    cancelTag()
}

const removeTag = async (n, index) => {
    const newTags = [...n.tags]
    newTags.splice(index, 1)
    try {
        await fleetStore.updateNode(n.id, {
            alias: n.alias,
            tags: newTags,
            publicIp: n.publicIp,
            privateIp: n.privateIp,
        })
    } catch (err) {
        // Store handles rollback
    }
}

// --- SENSOR MODAL STATE ---
const selectedSensor = ref(null)
const isEditingSensor = ref(false)
const editingSensorId = ref(null)
const activeTab = ref('readme')
const envVarValues = ref({})
const activeEnvVar = ref(null)
const openBooleanDropdown = ref(null)
const isSeverityOpen = ref(false)
const rawCompose = ref('')
const highlightedCompose = ref('')
const composePre = ref(null)

// --- MODAL STATE ---
const showKeyModal = ref(false)
const showSyncModal = ref(false)
const syncComposeYaml = ref('')
const showManualSync = ref(false)

watch(showSyncModal, (val) => {
    if (!val) showManualSync.value = false
})

const syncCommand = computed(() => {
    if (!node.value) return ''
    const hubUrl = config.hubEndpoint || window.location.origin
    if (!node.value.lastHeartbeat) {
        return `./wizard --link ${hubUrl} ${node.value.apiKey}\n./wizard apply`
    }
    return `./wizard apply`
})

// --- COPY STATE (ephemeral UI) ---
const { copiedStates, handleCopy } = useClipboard()

// --- FORMATTERS ---
const formatEventType = (type) => type ? type.replace(/_/g, ' ') : ''

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
  if (def === undefined || def === null) return ''
  const strDef = String(def)
  if (!strDef.includes('{{')) return strDef
  const elseMatch = strDef.match(/\{\{\s*else\s*\}\}(.*?)\{\{\s*end\s*\}\}/)
  if (elseMatch) return elseMatch[1].trim()
  const funcMatch = strDef.match(/\{\{\s*[a-zA-Z]+\s+([0-9]+)\s*\}\}/)
  if (funcMatch) return funcMatch[1].trim()
  return ''
}

const getEnvType = (env) => {
  if (env.type === 'boolean' || env.type === 'bool') return 'boolean'
  if (env.type === 'int' || env.type === 'number') return 'number'
  
  const def = getUIDefault(env.default).trim()
  if (def === 'true' || def === 'false') return 'boolean'
  if (def !== '' && !isNaN(Number(def))) return 'number'
  
  return 'text'
}

const incrementEnvVar = (envName, defaultVal) => {
  const current = envVarValues.value[envName] !== undefined && envVarValues.value[envName] !== '' 
    ? envVarValues.value[envName] 
    : getUIDefault(defaultVal)
  envVarValues.value[envName] = String(Number(current || 0) + 1)
}

const decrementEnvVar = (envName, defaultVal) => {
  const current = envVarValues.value[envName] !== undefined && envVarValues.value[envName] !== '' 
    ? envVarValues.value[envName] 
    : getUIDefault(defaultVal)
  envVarValues.value[envName] = String(Number(current || 0) - 1)
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
  if (!node.value?.id || !sensor.sensorId) return
  try {
    await fleetStore.toggleSilence(node.value.id, sensor.sensorId, !sensor.isSilenced)
  } catch (err) {
    alert('Unable to change sensor silence state. Please try again.')
  }
}

const handleRemoveSensor = async (sensor) => {
  if (!node.value?.id || !sensor.sensorId) return
  try {
    await fleetStore.removeSensor(node.value.id, sensor.sensorId)
  } catch (err) {
    alert('Could not remove sensor. Please try again.')
  }
}

const handleApplySensor = async () => {
  if (!selectedSensor.value || !node.value?.id) return

  const safeEnvValues = Object.fromEntries(
    Object.entries(envVarValues.value).map(([k, v]) => [k, v !== undefined && v !== null ? String(v) : ''])
  )

  try {
    if (isEditingSensor.value && editingSensorId.value) {
      await fleetStore.updateSensor(node.value.id, editingSensorId.value, {
        customName: selectedSensor.value.name || selectedSensor.value.id,
        configValues: safeEnvValues,
      })
    } else {
      await fleetStore.addSensor(node.value.id, {
        sensorId: selectedSensor.value.id || selectedSensor.value.sensorId || selectedSensor.value.name,
        customName: selectedSensor.value.name || selectedSensor.value.id,
        configValues: safeEnvValues,
      })
    }
    closeSensor()
  } catch (err) {
    alert(isEditingSensor.value ? 'Could not update sensor. Please try again.' : 'Could not add sensor to this node. Please try again.')
  }
}

// --- NODE ACTIONS (delegated to store) ---

const handleManageKey = () => { showKeyModal.value = true }

const triggerManualSync = async () => {
  if (!node.value?.id) return
  try {
    syncComposeYaml.value = await fleetStore.syncNode(node.value.id)
    showSyncModal.value = true
  } catch (err) {
    alert('Unable to sync this node. Please try again.')
  }
}

const handleSilenceNode = () => {
    if (!node.value?.id) return
    fleetStore.silenceNode(node.value.id)
}

const handleDeleteNode = () => {
    if (!node.value?.id) return
    if (confirm(`Delete node "${node.value.alias}"? This cannot be undone.`)) {
        fleetStore.deleteNode(node.value.id)
        appStore.currentView = 'fleet'
    }
}

// --- NAVIGATION ---

watch(selectedNodeId, async (value) => {
    if (!value) {
        appStore.currentView = 'fleet'
        return
    }
    await Promise.all([
        fleetStore.fetchNodeDetails(value),
        eventsStore.fetchEvents(false, value)
    ])
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
  isEditingSensor.value = false
  editingSensorId.value = null
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

const editSensor = (installedSensor) => {
  const manifest = getManifestForSensor(installedSensor)
  if (!manifest) {
      alert('Sensor manifest not found')
      return
  }
  
  const apiKey = node.value?.apiKey
  selectedSensor.value = manifest
  isEditingSensor.value = true
  editingSensorId.value = installedSensor.sensorId
  activeTab.value = 'config'
  envVarValues.value = {}
  envVarValues.value['HW_SEVERITY'] = 'critical'
  envVarValues.value['HW_HUB_ENDPOINT'] = config.hubEndpoint || window.location.origin
  envVarValues.value['HW_HUB_KEY'] = apiKey || '<YOUR_HW_NODE_KEY>'
  
  manifest.deployment?.env_vars?.forEach(env => {
    if (!['HW_HUB_ENDPOINT', 'HW_HUB_KEY', 'HW_SEVERITY'].includes(env.name)) {
      envVarValues.value[env.name] = getUIDefault(env.default)
    }
  })
  
  if (installedSensor.envVars) {
      Object.keys(installedSensor.envVars).forEach(key => {
          envVarValues.value[key] = installedSensor.envVars[key]
      })
  }
  
  document.body.style.overflow = 'hidden'
  fetchYamlFromHub()
}

const closeSensor = () => {
  selectedSensor.value = null
  isEditingSensor.value = false
  editingSensorId.value = null
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
      hubEndpoint: config.hubEndpoint || window.location.origin,
      hubKey: apiKey || '<YOUR_HW_NODE_KEY>',
      sensors: [{
        sensorId: selectedSensor.value.id,
        envValues: safeEnvValues,
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
    <div class="h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6">
        
        <div class="mt-4 sm:mt-6 mb-4 shrink-0">
            <button @click="fleetStore.clearSelection()" class="flex items-center gap-1.5 text-sm font-medium text-text-m hover:text-text-h transition-colors outline-none w-max">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg>
                Back to Fleet
            </button>
        </div>

        <div v-if="node" class="bg-bg-base border border-border-default rounded-lg p-5 sm:p-6 mb-8 shrink-0 shadow-sm flex flex-col gap-6">
            
            <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
                <div>
                    <div class="flex items-center gap-3 mb-3">
                        <h1 v-if="editingAliasNodeId !== node.id"
                            @click="enableAliasEdit(node)"
                            class="text-[length:var(--text-h1)] font-semibold text-text-h leading-tight truncate max-w-[400px] cursor-edit hover:text-primary-main border-b border-dashed border-transparent hover:border-primary-main transition-colors select-none"
                            :title="`Click to rename ${node.alias}`">
                            {{ node.alias }}
                        </h1>
                        <input v-else
                            :ref="el => { if (el) aliasInputRefs[node.id] = el }"
                            v-model="rawAliasValue"
                            @keyup.enter="saveAlias(node)"
                            @keyup.esc="cancelAliasEdit"
                            @blur="saveAlias(node)"
                            class="text-[length:var(--text-h1)] font-semibold text-text-h bg-input-bg border border-primary-main rounded px-2 py-0.5 focus:outline-none ring-1 ring-focus-ring max-w-[400px] truncate"
                        />
                        <BaseStatusDot :status="node.status" />
                        <span v-if="node.hasPendingConfig"
                            class="shrink-0 text-high"
                            title="Pending sync — click Sync Node below to apply changes">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                        </span>
                    </div>
                    
                    <div class="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-text-m">
                        <div @click="handleCopy('detail-pub', node.publicIp)"
                             class="flex items-center gap-1.5 cursor-pointer transition-colors duration-[var(--duration-fast)] group/pub w-max rounded px-1 -ml-1 py-0.5 border border-transparent"
                             :class="copiedStates['detail-pub'] ? 'bg-success-bg text-success-text border-success-border' : 'text-text-m hover:text-text-h hover:bg-secondary-hover'">
                            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                            <span class="font-mono">{{ copiedStates['detail-pub'] ? 'Copied!' : (node.publicIp || 'Unknown') }}</span>
                        </div>
                        <div @click="handleCopy('detail-priv', node.privateIp)"
                             class="flex items-center gap-1.5 cursor-pointer transition-colors duration-[var(--duration-fast)] group/priv w-max rounded px-1 -ml-1 py-0.5 border border-transparent"
                             :class="copiedStates['detail-priv'] ? 'bg-success-bg text-success-text border-success-border' : 'text-text-m hover:text-text-h hover:bg-secondary-hover'">
                            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="2" y="14" width="8" height="6" rx="2" ry="2"/><rect x="14" y="14" width="8" height="6" rx="2" ry="2"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 14v-2a2 2 0 012-2h8a2 2 0 012 2v2M12 2v8"/><rect x="8" y="2" width="8" height="6" rx="2" ry="2"/></svg>
                            <span class="font-mono">{{ copiedStates['detail-priv'] ? 'Copied!' : (node.privateIp || 'Unknown') }}</span>
                        </div>
                        <div class="h-4 w-px bg-border-default hidden sm:block"></div>
                        <div class="flex items-center gap-1.5">
                            <span class="text-text-h font-medium">Last Event:</span> {{ lastEventTime }}
                        </div>
                        <div class="h-4 w-px bg-border-default hidden sm:block"></div>
                        <div class="flex items-center gap-1.5 flex-wrap">
                            <span v-for="(tag, index) in node.tags" :key="tag"
                                class="px-2 py-0.5 bg-bg-inset border border-border-default text-text-m text-sm font-medium rounded-md tracking-wider flex items-center gap-1.5 group/tag transition-colors hover:border-text-m">
                                {{ tag }}
                                <button @click.stop="removeTag(node, index)" class="opacity-0 group-hover/tag:opacity-100 text-text-m hover:text-danger-text transition-all outline-none focus:opacity-100">
                                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                                </button>
                            </span>

                            <div v-if="editingTagNodeId === node.id" class="relative flex items-center">
                                <span class="absolute left-2 text-text-m text-sm pointer-events-none">+</span>
                                <input
                                    :ref="el => { if (el) tagInputRefs[node.id] = el }"
                                    v-model="newTagValue"
                                    @keyup.enter="saveTag(node)"
                                    @keyup.esc="cancelTag"
                                    @blur="cancelTag"
                                    class="pl-5 pr-2 py-0.5 bg-input-bg border border-primary-main text-text-h text-sm rounded-md focus:outline-none ring-1 ring-focus-ring w-28 shadow-sm transition-all placeholder:text-text-m/50"
                                    placeholder="tag name..."
                                />
                            </div>

                            <button v-else @click.stop="enableTagEdit(node.id)"
                                    class="px-1.5 py-0.5 border border-dashed border-border-default text-text-m text-sm rounded-md hover:text-text-h hover:border-text-m transition-colors outline-none focus:ring-1 focus:ring-focus-ring">
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
                        <button @click="handleSilenceNode" class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">
                            {{ fleetStore.isNodeSilenced(node.id) ? 'Unsilence Node' : 'Silence Node' }}
                        </button>
                        
                        <button @click="handleDeleteNode" class="w-full text-left px-3 py-2 text-sm text-danger-text hover:bg-danger-bg transition-colors border-t border-border-default mt-1 pt-2">Delete Node</button>
                    </BaseMeatballMenu>
                </div>
            </div>

            <div v-if="node.hasPendingConfig" class="flex items-center justify-between w-full max-w-xl bg-high/10 border border-high/30 rounded-lg p-4 transition-all duration-normal">
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
                
                <!-- Sensor Volume (24h) -->
                <div class="bg-bg-surface w-full max-w-2xl border border-border-default rounded-lg p-5 shadow-sm flex flex-col">
                    <h3 class="text-sm font-semibold text-text-h mb-4">Sensor Volume (24h)</h3>
                    <div v-if="topSensors.length > 0" class="space-y-3 overflow-y-auto custom-scroll max-h-[240px] pr-1">
                        <div v-for="sensor in topSensors" :key="sensor.id">
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

                <!-- Recent Activity — mini event table -->
                <div class="bg-bg-surface border border-border-default rounded-lg flex flex-col overflow-hidden shadow-sm">
                    <div class="px-4 py-3 border-b border-border-default flex items-center justify-between bg-bg-surface shrink-0">
                        <h3 class="text-sm font-semibold text-text-h">Recent Activity</h3>
                        <button class="text-sm font-medium text-text-m hover:text-text-h transition-colors">View All &rarr;</button>
                    </div>
                    
                    <div class="flex-1 overflow-y-auto custom-scroll bg-bg-surface max-h-[240px]">
                        <table class="w-full text-left border-collapse">
                            <thead class="text-sm font-medium text-text-m tracking-wider sticky top-0 bg-bg-surface shadow-[0_1px_0_0_var(--color-border-default)]">
                                <tr>
                                    <th class="px-3 py-2 w-14"></th>
                                    <th class="px-3 py-2">Event</th>
                                    <th class="px-3 py-2">Source</th>
                                    <th class="px-3 py-2">Sensor</th>
                                    <th class="px-3 py-2 text-right">Time</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-if="recentActivity.length === 0">
                                    <td colspan="5" class="px-3 py-4 text-center text-sm text-text-m">No recent activity on this node.</td>
                                </tr>
                                <tr v-for="event in recentActivity" :key="event.id" 
                                    class="hover:bg-secondary-hover cursor-pointer transition-colors duration-[var(--duration-fast)] relative z-0"
                                    :class="'bleed-' + event.severity.toLowerCase()">
                                    
                                    <td class="px-3 py-2 border-b border-border-default">
                                        <span class="px-1.5 py-0.5 rounded border text-sm font-medium bg-bg-base whitespace-nowrap capitalize" 
                                              :style="{ borderColor: `var(--color-${event.severity.toLowerCase()})`, color: `var(--color-${event.severity.toLowerCase()})` }">
                                            {{ event.severity }}
                                        </span>
                                    </td>
                                    
                                    <td class="px-3 py-2 border-b border-border-default">
                                        <span class="text-sm text-text-h font-medium capitalize">{{ formatEventType(event.eventTrigger) }}</span>
                                    </td>

                                    <td class="px-3 py-2 border-b border-border-default">
                                        <span class="text-sm text-text-m font-mono truncate block max-w-[100px]">{{ event.source }}</span>
                                    </td>

                                    <td class="px-3 py-2 border-b border-border-default">
                                        <span class="text-sm text-text-m font-mono truncate block max-w-[80px]">{{ formatSensorId(event.sensorId) }}</span>
                                    </td>
                                    
                                    <td class="px-3 py-2 border-b border-border-default text-right">
                                        <span class="text-sm text-text-m font-mono whitespace-nowrap">{{ event.time }}</span>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>

            <div>
                <h3 class="text-sm font-semibold text-text-h mb-4 mt-2">Deployed Sensors</h3>
                <div v-if="node.installedSensors?.length > 0" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-5 gap-4">
                    <div v-for="sensor in node.installedSensors" :key="sensor.id" class="bg-bg-surface border border-border-default rounded-lg p-4 flex flex-col group hover:border-text-m transition-colors shadow-sm relative overflow-hidden">
                        
                        <div class="absolute top-0 left-0 right-0 h-1 transition-colors" :class="sensor.status === 'up' ? 'bg-success-main' : 'bg-danger-main'"></div>

                        <div class="flex justify-between items-start mt-1">
                            <div class="flex items-center gap-3 min-w-0">
                                <div class="w-8 h-8 rounded bg-bg-inset border border-border-default/50 flex items-center justify-center text-text-h shrink-0">
                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="sensorIcon(sensor)"></path></svg>
                                </div>
                                <div class="min-w-0">
                                    <h4 class="text-sm font-semibold text-text-h truncate">{{ sensor.display }}</h4>
                                    <span class="text-sm text-text-m font-mono block truncate">{{ formatSensorId(sensor.name) }}</span>
                                </div>
                            </div>
                            <BaseMeatballMenu :id="`sensor-menu-${sensor.id}`">
                                <button @click="editSensor(sensor)" class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">Edit Configuration</button>
                                <button @click="handleToggleSensorSilence(sensor)" class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">
                                    {{ sensor.isSilenced ? 'Unsilence Alerts' : 'Silence Alerts' }}
                                </button>
                                <button @click="handleRemoveSensor(sensor)" class="w-full text-left px-3 py-2 text-sm text-danger-text hover:bg-danger-bg transition-colors border-t border-border-default mt-1 pt-2">Remove Sensor</button>
                            </BaseMeatballMenu>
                        </div>
                        <div class="mt-3 pt-3 border-t border-border-default flex justify-between items-center">
                             <span class="px-1.5 py-0.5 rounded text-sm font-medium tracking-wider bg-bg-inset text-text-m border border-border-default/50">{{ sensorOsi(sensor) }}</span>
                             <svg v-if="sensor.isSilenced" class="w-3.5 h-3.5 text-medium shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" title="Alerts Silenced"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/></svg>
                        </div>
                    </div>
                </div>
                <div v-else class="border border-dashed border-border-default rounded-lg p-8 flex flex-col items-center justify-center text-center bg-bg-surface/50">
                    <p class="text-sm text-text-h font-medium">No sensors deployed</p>
                    <p class="text-sm text-text-m mt-1">Select a sensor from the catalog below.</p>
                </div>
            </div>
        </div>

        <div class="shrink-0 mb-6">
            <h2 class="text-base font-semibold text-text-h mb-4">Sensor Catalog</h2>
            <div v-if="isManifestLoading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                <div v-for="i in 4" :key="i" class="bg-bg-surface border border-border-default rounded-lg p-5 h-36 animate-pulse"></div>
            </div>
            <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                <div v-for="s in sensors" :key="s.id" @click="openSensor(s)" class="bg-bg-surface border border-border-default rounded-lg p-4 shadow-sm hover:border-primary-main hover:shadow-md cursor-pointer transition-all duration-normal group flex flex-col">
                    <div class="flex justify-between items-start mb-3">
                        <div class="w-10 h-10 rounded-md bg-bg-base border border-border-default/50 text-text-h flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform duration-normal">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="s.icon_svg"></path></svg>
                        </div>
                        <span class="px-2 py-0.5 rounded text-sm font-medium tracking-wider bg-bg-inset text-text-m border border-border-default/50">{{ s.osi_layer }}</span>
                    </div>
                    <h3 class="text-sm font-semibold text-text-h mb-1">{{ s.name }}</h3>
                    <p class="text-sm text-text-m leading-relaxed line-clamp-2">{{ s.description }}</p>
                </div>
            </div>
        </div>

        <!-- Manage Key Modal -->
        <BaseModal :show="showKeyModal" @close="showKeyModal = false" title="Manage Node Key">
            <div class="space-y-4">
                <p class="text-sm text-text-m">This is the unique API key for this node. It is used to authenticate the node with the hub.</p>
                <div class="bg-bg-surface border border-border-default rounded-lg p-4 font-mono text-sm break-all">
                    <div class="flex items-center justify-between mb-2">
                        <span class="text-sm text-text-h font-semibold">Node API Key</span>
                        <button @click="handleCopy('key-modal', node.apiKey)"
                                class="px-2.5 py-1 rounded-md text-sm font-medium transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 border outline-none"
                                :class="copiedStates['key-modal'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-secondary-main text-secondary-text border-secondary-border hover:bg-secondary-hover hover:text-text-h'">
                            {{ copiedStates['key-modal'] ? 'Copied!' : 'Copy' }}
                        </button>
                    </div>
                    <div class="text-text-m select-all">{{ node.apiKey || 'Unavailable' }}</div>
                </div>
                <div class="flex justify-end">
                    <BaseButton variant="secondary" @click="showKeyModal = false">Close</BaseButton>
                </div>
            </div>
        </BaseModal>

        <!-- Sync Compose Modal -->
        <Teleport to="body">
            <transition enter-active-class="transition duration-normal ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-[var(--duration-fast)] ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <div v-if="showSyncModal" class="fixed inset-0 z-[var(--z-modal)] flex justify-center items-center p-4 sm:p-6 bg-black/60 backdrop-blur-sm" @click.self="showSyncModal = false">
                    
                    <div class="bg-bg-base w-full max-w-2xl max-h-[85vh] rounded-lg shadow-2xl flex flex-col overflow-hidden border border-border-default transform transition-all">
                        
                        <div class="px-6 py-5 border-b border-border-default flex justify-between items-center bg-bg-surface shrink-0">
                            <h2 class="text-base font-semibold text-text-h">Synchronize Node</h2>
                            <button @click="showSyncModal = false" class="p-2 -mr-2 text-text-m hover:text-text-h transition-colors duration-[var(--duration-fast)] rounded-full hover:bg-secondary-hover focus:outline-none">
                                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path></svg>
                            </button>
                        </div>

                        <div class="flex-1 overflow-y-auto custom-scroll bg-bg-base p-6 md:p-8 space-y-6">
                            
                            <div>
                                <h3 class="text-sm font-semibold text-text-h mb-2">Automatic Deployment (Recommended)</h3>
                                <p class="text-sm text-text-m mb-4">
                                    Run the HoneyWire Wizard on your server to automatically reconcile this node's configuration.
                                </p>
                                
                                <div class="bg-bg-inset border border-border-default rounded-md p-4 relative group flex flex-col gap-3">
                                    <code class="text-success-text text-xs font-mono whitespace-pre-wrap break-all leading-relaxed">{{ syncCommand }}</code>
                                    
                                    <button @click="handleCopy('sync-cmd', syncCommand)"
                                            class="self-end px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 focus:outline-none border"
                                            :class="copiedStates['sync-cmd'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-bg-surface text-text-h border-border-default hover:bg-secondary-hover'">
                                        {{ copiedStates['sync-cmd'] ? 'Copied!' : 'Copy' }}
                                    </button>
                                </div>
                            </div>

                            <div class="border-t border-border-default pt-6">
                                <button @click="showManualSync = !showManualSync" class="flex items-center justify-between w-full text-left group outline-none">
                                    <span class="text-sm font-semibold text-text-h group-hover:text-primary-main transition-colors">Advanced / Manual Deployment</span>
                                    <svg class="w-4 h-4 text-text-m transition-transform duration-normal" :class="showManualSync ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                                </button>
                                
                                <div v-show="showManualSync" class="mt-4 space-y-4">
                                    <p class="text-sm text-danger-main">
                                        Docker Compose v5.0.0+ is strictly required.
                                    </p>
                                    
                                    <p class="text-sm text-text-m">
                                        Save the following configuration to <code class="px-1.5 py-0.5 bg-bg-inset border border-border-default rounded text-xs font-mono">/opt/honeywire/sensors/honeywire-compose.yml</code>.
                                    </p>
                                    
                                    <div class="relative bg-bg-surface border border-border-default rounded-lg p-4 font-mono text-sm text-text-h overflow-auto max-h-[30vh] custom-scroll">
                                        <button @click="handleCopy('sync-yaml', syncComposeYaml)"
                                                class="absolute top-3 right-3 px-2.5 py-1.5 rounded-md text-xs font-medium transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 z-10 focus:outline-none border"
                                                :class="copiedStates['sync-yaml'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-secondary-main text-secondary-text border-secondary-border hover:bg-secondary-hover hover:text-text-h'">
                                            {{ copiedStates['sync-yaml'] ? 'Copied!' : 'Copy' }}
                                        </button>
                                        <pre class="whitespace-pre-wrap break-words pr-16">{{ syncComposeYaml || 'No compose output available.' }}</pre>
                                    </div>

                                    <p class="text-sm text-text-m">
                                        Then, apply the configuration using Docker Compose:
                                    </p>

                                    <div class="bg-bg-inset border border-border-default rounded-md p-4 relative group flex flex-col gap-3">
                                        <code class="text-text-h text-xs font-mono break-all leading-relaxed">cd /opt/honeywire/sensors<br/>docker compose -f honeywire-compose.yml -p honeywire up -d --remove-orphans</code>
                                        
                                        <button @click="handleCopy('manual-cmd', 'cd /opt/honeywire/sensors\ndocker compose -p honeywire up -d --remove-orphans')"
                                                class="self-end px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 focus:outline-none border"
                                                :class="copiedStates['manual-cmd'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-bg-surface text-text-h border-border-default hover:bg-secondary-hover'">
                                            {{ copiedStates['manual-cmd'] ? 'Copied!' : 'Copy' }}
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="px-6 py-4 border-t border-border-default bg-bg-surface flex justify-end shrink-0">
                            <BaseButton variant="secondary" @click="showSyncModal = false">Done</BaseButton>
                        </div>
                    </div>
                </div>
            </transition>
        </Teleport>

        <!-- Sensor Deployment Modal -->
        <Teleport to="body">
            <transition enter-active-class="transition duration-normal ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-[var(--duration-fast)] ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <div v-if="selectedSensor" class="fixed inset-0 z-[var(--z-modal)] flex justify-center items-center p-4 sm:p-6 bg-black/60 backdrop-blur-sm" @click.self="closeSensor">
                    
                    <div class="bg-bg-base w-full max-w-3xl h-full max-h-[85vh] rounded-lg shadow-2xl flex flex-col overflow-hidden border border-border-default transform transition-all">
                        
                        <div class="px-6 py-5 border-b border-border-default flex justify-between items-start bg-bg-surface shrink-0">
                            <div class="flex items-center gap-4">
                                <div class="w-12 h-12 rounded-md bg-bg-inset border border-border-default/50 text-text-h flex items-center justify-center shrink-0 shadow-sm">
                                    <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="selectedSensor.icon_svg"></path></svg>
                                </div>
                                <div>
                                    <h2 class="text-base font-semibold text-text-h">{{ selectedSensor.name }}</h2>
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
                            <div v-show="activeTab === 'readme'" class="p-6 md:p-8 readme-container text-sm text-text-m">
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
                                                <div v-if="getEnvType(env) === 'boolean'" class="relative w-full">
                                                    <div @click="openBooleanDropdown = openBooleanDropdown === env.name ? null : env.name" class="w-full px-3 py-2 text-sm bg-input-bg border rounded-md cursor-pointer flex justify-between items-center transition-all duration-[var(--duration-fast)] shadow-sm select-none" :class="openBooleanDropdown === env.name ? 'border-primary-main ring-1 ring-focus-ring' : 'border-input-border hover:border-border-default'">
                                                        <span v-if="String(envVarValues[env.name] !== undefined && envVarValues[env.name] !== '' ? envVarValues[env.name] : getUIDefault(env.default)) === 'true'" class="text-success-main font-medium flex items-center gap-2">
                                                            <span class="w-2 h-2 rounded-full bg-success-main"></span>true
                                                        </span>
                                                        <span v-else-if="String(envVarValues[env.name] !== undefined && envVarValues[env.name] !== '' ? envVarValues[env.name] : getUIDefault(env.default)) === 'false'" class="text-danger-main font-medium flex items-center gap-2">
                                                            <span class="w-2 h-2 rounded-full bg-danger-main"></span>false
                                                        </span>
                                                        <span v-else class="text-text-m font-medium flex items-center gap-2">
                                                            <span class="w-2 h-2 rounded-full bg-border-default"></span>Select...
                                                        </span>
                                                        <svg class="w-4 h-4 text-text-m transition-transform duration-200" :class="openBooleanDropdown === env.name ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                                                    </div>
                                                    
                                                    <div v-if="openBooleanDropdown === env.name" @click="openBooleanDropdown = null" class="fixed inset-0 z-[var(--z-elevated)]"></div>

                                                    <transition enter-active-class="transition duration-100 ease-out" enter-from-class="transform scale-95 opacity-0" enter-to-class="transform scale-100 opacity-100" leave-active-class="transition duration-75 ease-in" leave-from-class="transform scale-100 opacity-100" leave-to-class="transform scale-95 opacity-0">
                                                        <ul v-if="openBooleanDropdown === env.name" class="absolute z-[var(--z-dropdown)] w-full mt-1 bg-bg-surface border border-border-default rounded-md shadow-lg py-1 overflow-hidden">
                                                            <li @click="envVarValues[env.name] = 'true'; openBooleanDropdown = null" class="px-3 py-2 cursor-pointer transition-colors text-sm font-medium duration-[var(--duration-fast)] flex items-center gap-2 text-success-main hover:bg-success-bg border border-transparent hover:border-success-border/50">
                                                                <span class="w-2 h-2 rounded-full bg-success-main"></span>true
                                                            </li>
                                                            <li @click="envVarValues[env.name] = 'false'; openBooleanDropdown = null" class="px-3 py-2 cursor-pointer transition-colors text-sm font-medium duration-[var(--duration-fast)] flex items-center gap-2 text-danger-main hover:bg-danger-bg border border-transparent hover:border-danger-border/50">
                                                                <span class="w-2 h-2 rounded-full bg-danger-main"></span>false
                                                            </li>
                                                        </ul>
                                                    </transition>
                                                </div>
                                                <div v-else-if="getEnvType(env) === 'number'" class="relative w-full flex items-center">
                                                    <input v-model="envVarValues[env.name]" type="number" :placeholder="getUIDefault(env.default)" @focus="activeEnvVar = env.name" @blur="activeEnvVar = null" class="w-full pl-3 pr-10 py-2 text-sm text-text-h bg-input-bg border border-input-border rounded-md focus:outline-none focus:border-primary-main focus:ring-1 focus:ring-focus-ring transition-all duration-[var(--duration-fast)] shadow-sm placeholder:text-text-m/50 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" />
                                                    <div class="absolute right-1 top-1 bottom-1 flex flex-col border-l border-input-border w-7">
                                                        <button tabindex="-1" @click.prevent="incrementEnvVar(env.name, env.default)" class="flex-1 flex items-center justify-center text-text-m hover:text-text-h hover:bg-secondary-hover transition-colors rounded-tr-md outline-none border-b border-input-border"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7"></path></svg></button>
                                                        <button tabindex="-1" @click.prevent="decrementEnvVar(env.name, env.default)" class="flex-1 flex items-center justify-center text-text-m hover:text-text-h hover:bg-secondary-hover transition-colors rounded-br-md outline-none"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7"></path></svg></button>
                                                    </div>
                                                </div>
                                                <input v-else v-model="envVarValues[env.name]" type="text" :placeholder="getUIDefault(env.default)" @focus="activeEnvVar = env.name" @blur="activeEnvVar = null" class="w-full px-3 py-2 text-sm text-text-h bg-input-bg border border-input-border rounded-md focus:outline-none focus:border-primary-main focus:ring-1 focus:ring-focus-ring transition-all duration-[var(--duration-fast)] shadow-sm placeholder:text-text-m/50" />
                                                <p class="text-sm text-text-m mt-1">{{ env.description }}</p>
                                            </div>
                                        </template>

                                    </div>
                                </div>
                                
                                <div class="relative flex-1 min-h-[250px] mb-6">
                                    <button @click="handleCopy('compose-yaml', rawCompose)"
                                            class="absolute top-3 right-3 px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 z-10 focus:outline-none border"
                                            :class="copiedStates['compose-yaml'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-secondary-main text-secondary-text border-secondary-border hover:bg-secondary-hover hover:text-text-h'">
                                        {{ copiedStates['compose-yaml'] ? 'Copied!' : 'Copy' }}
                                    </button>
                                    <pre ref="composePre" v-html="highlightedCompose" class="absolute inset-0 w-full h-full bg-bg-surface text-text-m p-4 rounded-md text-sm font-mono custom-scroll border border-border-default leading-relaxed overflow-auto focus:outline-none scroll-smooth shadow-inner"></pre>
                                </div>

                                <div class="mt-auto border-t border-border-default pt-4 flex justify-end">
                                    <BaseButton variant="primary" @click="handleApplySensor" class="px-6">{{ isEditingSensor ? 'Apply Settings' : 'Add to Node' }}</BaseButton>
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
.readme-container :deep(h3) { font-size: var(--text-sm); font-weight: var(--font-weight-medium); color: var(--text-h); margin-top: 1.5rem; margin-bottom: 0.75rem; }
.readme-container :deep(p) { line-height: var(--text-leading-normal); margin-bottom: 1rem; }
.readme-container :deep(code) { font-family: var(--font-mono); background-color: var(--input-bg); color: var(--text-h); padding: 0.1rem 0.3rem; border-radius: var(--radius-sm); font-size: var(--text-sm); border: 1px solid var(--input-border); }
</style>