<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { useAppStore } from '../stores/System/app'
import { useFleetStore } from '../stores/Fleet/fleet'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import BaseWidget from '../components/ui/layout/BaseWidget.vue'
import BaseStatusDot from '../components/ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../components/ui/navigation/BaseMeatballMenu.vue'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import BaseModal from '../components/ui/feedback/BaseModal.vue'
import BaseInput from '../components/ui/forms/BaseInput.vue'
import { useClipboard } from '../utils/useClipboard'

const appStore = useAppStore()
const fleetStore = useFleetStore()

// --- MANIFEST CATALOG ---
const isManifestLoading = ref(true)
const manifestData = ref([])

const manifestMap = computed(() => {
    const map = new Map()
    for (const s of manifestData.value) {
        map.set(s.id, s)
        map.set(s.sensorId, s)
        map.set(s.name, s)
    }
    return map
})

const getOsiForSensor = (installedSensor) => {
    const manifest = manifestMap.value.get(installedSensor.id)
        || manifestMap.value.get(installedSensor.name)
        || manifestMap.value.get(installedSensor.sensorId)
    return manifest?.osi_layer || installedSensor.osi || 'Other'
}

// --- PARALLEL LOAD ---
const isInitialLoading = ref(true)

onMounted(async () => {
    try {
        const [, manifests] = await Promise.all([
            fleetStore.fetchFleet(),
            fleetStore.fetchManifests().catch(err => {
                console.error('Failed to load manifests', err)
                return []
            })
        ])
        manifestData.value = manifests
    } finally {
        isManifestLoading.value = false
        isInitialLoading.value = false
    }
})

// --- DEPLOY MODAL STATE (ephemeral UI) ---
const showDeployModal = ref(false)
const deployStep = ref(1)
const newNodeForm = ref({ alias: '' })
const generatedNodeKey = ref('')

// --- INLINE RENAME STATE (ephemeral UI) ---
const editingAliasNodeId = ref(null)
const rawAliasValue = ref('')
const aliasInputRefs = ref({})

const enableAliasEdit = async (node) => {
    editingAliasNodeId.value = node.id
    rawAliasValue.value = node.alias
    await nextTick()
    if (aliasInputRefs.value[node.id]) {
        aliasInputRefs.value[node.id].focus()
        aliasInputRefs.value[node.id].select()
    }
}

const cancelAliasEdit = () => {
    editingAliasNodeId.value = null
    rawAliasValue.value = ''
}

const saveAlias = async (node) => {
    if (editingAliasNodeId.value !== node.id) return
    const val = rawAliasValue.value.trim()
    if (val && val !== node.alias) {
        try {
            await fleetStore.updateNode(node.id, {
                alias: val,
                tags: node.tags,
                publicIp: node.publicIp,
                privateIp: node.privateIp,
            })
        } catch (err) {
            // Store handles rollback
        }
    }
    editingAliasNodeId.value = null
    rawAliasValue.value = ''
}

// --- TAGS LOGIC (ephemeral UI + store delegation) ---
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

const saveTag = async (node) => {
    const val = newTagValue.value.trim()
    if (val && !node.tags.includes(val)) {
        try {
            await fleetStore.updateNode(node.id, {
                alias: node.alias,
                tags: [...node.tags, val],
                publicIp: node.publicIp,
                privateIp: node.privateIp,
            })
        } catch (err) {
            // Store handles rollback
        }
    }
    cancelTag()
}

const removeTag = async (node, index) => {
    const newTags = [...node.tags]
    newTags.splice(index, 1)
    try {
        await fleetStore.updateNode(node.id, {
            alias: node.alias,
            tags: newTags,
            publicIp: node.publicIp,
            privateIp: node.privateIp,
        })
    } catch (err) {
        // Store handles rollback
    }
}

// --- OSI LAYER SORT ORDER ---
const osiOrder = ['Physical', 'Data Link', 'Network', 'Transport', 'Session', 'Presentation', 'Application', 'Other']

const sortOsi = (a, b) => {
    const aIdx = osiOrder.indexOf(a.type)
    const bIdx = osiOrder.indexOf(b.type)
    if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx
    if (aIdx !== -1) return -1
    if (bIdx !== -1) return 1
    return a.type.localeCompare(b.type)
}

// --- DATA MAPPING (presentation transform — belongs in view) ---
const displayNodes = computed(() => {
    if (isManifestLoading.value) {
        return fleetStore.nodes
            .filter(node => node.id && !node.id.startsWith('__pending_'))
            .map(node => {
                const sensorsList = node.installedSensors || []
                const totalSensors = sensorsList.length
                const onlineSensors = sensorsList.filter(s => s.status === 'up').length
                const isSilenced = totalSensors > 0 && sensorsList.every(s => s.isSilenced)
                return {
                    ...node,
                    totalSensors,
                    onlineSensors,
                    isSilenced,
                    sensorSummary: [],
                    hasUpdate: false,
                    isAwaitingCheckIn: node.status === 'pending' || (!node.lastHeartbeat && totalSensors === 0)
                }
            })
    }

    return fleetStore.nodes
        .filter(node => node.id && !node.id.startsWith('__pending_'))
        .map(node => {
            const sensorsList = node.installedSensors || []
            const onlineSensors = sensorsList.filter(s => s.status === 'up').length
            const totalSensors = sensorsList.length
            const isSilenced = totalSensors > 0 && sensorsList.every(s => s.isSilenced)

            const osiGroups = new Map()
            for (const sensor of sensorsList) {
                const osi = getOsiForSensor(sensor)
                if (!osiGroups.has(osi)) {
                    osiGroups.set(osi, [])
                }
                osiGroups.get(osi).push({
                    name: sensor.display || sensor.name,
                    status: sensor.status
                })
            }

            const sensorSummary = totalSensors > 0
                ? Array.from(osiGroups.entries())
                    .map(([type, sensors]) => ({ type, count: sensors.length, sensors }))
                    .sort(sortOsi)
                : []

            return {
                ...node,
                totalSensors,
                onlineSensors,
                isSilenced,
                sensorSummary,
                hasUpdate: false,
                isAwaitingCheckIn: node.status === 'pending' || (!node.lastHeartbeat && totalSensors === 0)
            }
        })
})

// --- ACTIONS (all delegated to store) ---

const handleDeploySubmit = async () => {
    if (!newNodeForm.value.alias) return
    try {
        const result = await fleetStore.createNode(newNodeForm.value.alias)
        generatedNodeKey.value = result.apiKey
        deployStep.value = 2
    } catch (err) {
        alert('Could not create node. Please try again.')
    }
}

const closeDeployModal = () => {
    showDeployModal.value = false
    setTimeout(() => {
        deployStep.value = 1
        newNodeForm.value.alias = ''
        generatedNodeKey.value = ''
    }, 300)
}

const handleSilenceNode = (nodeId) => fleetStore.silenceNode(nodeId)
const handleForgetNode = (nodeId) => fleetStore.deleteNode(nodeId)

// --- Copy Animation (ephemeral UI) ---
const { copiedStates, handleCopy } = useClipboard()

const handleOpenNodeDetail = (nodeId) => {
    fleetStore.selectTarget(nodeId, null, false)
    appStore.currentView = 'node-detail'
}

</script>

<template>
    <div class="h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6">
        <div class="flex items-center justify-between shrink-0">
             <PageHeader 
                title="Fleet Overview" 
                description="Monitor the health and status of your deployed HoneyWire nodes across all environments. Click on a node for detailed insights and management options."
            />
            
            <BaseButton variant="primary" class="gap-2 text-sm" @click="showDeployModal = true">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                Deploy New Node
            </BaseButton>
        </div>

        <!-- Detailed skeleton loading — matches BaseWidget structure -->
        <div v-if="isInitialLoading" class="grid grid-cols-1 lg:grid-cols-3 2xl:grid-cols-4 gap-5 auto-rows-max">
            <div v-for="i in 4" :key="i" class="bg-bg-surface border border-border-default rounded-lg overflow-hidden shadow-sm flex flex-col min-h-[280px]">
                <!-- Header: matches BaseWidget #header slot -->
                <div class="px-4 py-3">
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2.5">
                            <div class="w-2.5 h-2.5 rounded-full bg-bg-inset animate-pulse"></div>
                            <div class="h-4 w-28 bg-bg-inset rounded animate-pulse"></div>
                        </div>
                        <div class="h-6 w-6 bg-bg-inset rounded animate-pulse"></div>
                    </div>
                </div>
                <!-- Body: matches BaseWidget default slot -->
                <div class="flex-1 px-4 py-3 flex flex-col gap-4">
                    <!-- IPs row -->
                    <div class="grid grid-cols-2 gap-y-2 gap-x-4 text-sm">
                        <div class="flex items-center gap-1.5">
                            <div class="w-3.5 h-3.5 bg-bg-inset rounded animate-pulse"></div>
                            <div class="h-3 w-20 bg-bg-inset rounded animate-pulse"></div>
                        </div>
                        <div class="flex items-center gap-1.5">
                            <div class="w-3.5 h-3.5 bg-bg-inset rounded animate-pulse"></div>
                            <div class="h-3 w-20 bg-bg-inset rounded animate-pulse"></div>
                        </div>
                    </div>
                    <!-- Tags row -->
                    <div class="flex gap-1.5">
                        <div class="h-5 w-12 bg-bg-inset rounded-md animate-pulse"></div>
                        <div class="h-5 w-16 bg-bg-inset rounded-md animate-pulse"></div>
                        <div class="h-5 w-8 bg-bg-inset rounded-md animate-pulse"></div>
                    </div>
                    <!-- Sensor area: matches deployed sensors section -->
                    <div class="mt-auto bg-bg-inset/30 border border-border-default/50 rounded-lg p-3">
                        <div class="flex items-center justify-between mb-2">
                            <div class="h-3 w-24 bg-bg-inset rounded animate-pulse"></div>
                            <div class="h-3 w-16 bg-bg-inset rounded animate-pulse"></div>
                        </div>
                        <div class="flex gap-1.5 mt-1">
                            <div class="h-6 w-20 bg-bg-inset rounded-md animate-pulse"></div>
                            <div class="h-6 w-16 bg-bg-inset rounded-md animate-pulse"></div>
                        </div>
                    </div>
                </div>
                <!-- Footer: matches BaseWidget #footer slot -->
                <div class="border-t border-border-default/50 px-4 py-3">
                    <div class="h-3 w-32 mx-auto bg-bg-inset rounded animate-pulse"></div>
                </div>
            </div>
        </div>

        <div v-else class="grid grid-cols-1 lg:grid-cols-3 2xl:grid-cols-4 gap-5 auto-rows-max">
            
            <BaseWidget v-for="node in displayNodes" :key="node.id" class="flex flex-col h-full min-h-[280px] transition-all duration-[var(--duration-fast)]" :class="{ 'opacity-50 pointer-events-none': fleetStore.isNodeActionPending(node.id, 'deleting') }">
                
                <template #header>
                    <div class="flex items-start justify-between w-full">
                        <div class="flex items-center gap-2.5 min-w-0">
                            <BaseStatusDot :status="node.status" />
                            <div>
                                <div class="flex items-center gap-2">
                                    <span v-if="editingAliasNodeId !== node.id"
                                        @click="enableAliasEdit(node)"
                                        class="text-base font-medium text-text-h truncate max-w-[180px] cursor-edit hover:text-primary-main border-b border-dashed border-transparent hover:border-primary-main transition-colors select-none" 
                                        :title="`Click to rename ${node.alias}`">
                                        {{ node.alias }}
                                    </span>
                                    <input v-else
                                        :ref="el => { if (el) aliasInputRefs[node.id] = el }"
                                        v-model="rawAliasValue"
                                        @keyup.enter="saveAlias(node)"
                                        @keyup.esc="cancelAliasEdit"
                                        @blur="saveAlias(node)"
                                        class="text-base font-medium text-text-h bg-input-bg border border-primary-main rounded px-1.5 py-0 focus:outline-none ring-1 ring-focus-ring max-w-[180px] truncate"
                                    />
                                    <span v-if="node.hasPendingConfig" 
                                        class="shrink-0 text-high" 
                                        title="Pending sync — open node details to apply changes">
                                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                                    </span>
                                    <span v-if="node.hasUpdate" class="px-1.5 py-0.5 rounded text-sm bg-low/20 border border-low/40 text-highlight-text" title="Sensor Updates Available">
                                        Update
                                    </span>
                                    <svg v-if="node.isSilenced" class="w-4 h-4 text-medium shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" title="Node Silenced">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                                    </svg>
                                </div>
                            </div>
                        </div>

                        <BaseMeatballMenu v-if="node.id !== 'unassigned'" :id="`node-menu-${node.id}`">
                            <button @click="handleSilenceNode(node.id)" class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">
                                {{ node.isSilenced ? 'Unsilence Node' : 'Silence Node' }}
                            </button>
                            <button @click="handleForgetNode(node.id)" class="w-full text-left px-3 py-2 text-sm text-danger-text hover:bg-danger-bg transition-colors border-t border-border-default mt-1 pt-2">
                                <span v-if="fleetStore.isNodeActionPending(node.id, 'deleting')">Deleting...</span><span v-else>Delete Node</span>
                            </button>
                        </BaseMeatballMenu>
                    </div>
                </template>

                <div class="flex-1 mt-3 flex flex-col">
                    
                    <div v-if="node.id !== 'unassigned'" class="grid grid-cols-2 gap-y-2 gap-x-4 text-sm mb-4">
                        
                        <div @click="handleCopy(node.id + '-pub', node.publicIp)" 
                             class="flex items-center gap-1.5 cursor-pointer transition-colors duration-[var(--duration-fast)] group/pub w-max rounded px-1 -ml-1 py-0.5 border border-transparent"
                             :class="copiedStates[node.id + '-pub'] ? 'bg-success-bg text-success-text border-success-border' : 'text-text-m hover:text-text-h hover:bg-secondary-hover'">
                            <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                            <span class="font-mono truncate">{{ copiedStates[node.id + '-pub'] ? 'Copied!' : (node.publicIp || 'Unknown') }}</span>
                        </div>

                        <div @click="handleCopy(node.id + '-priv', node.privateIp)" 
                             class="flex items-center gap-1.5 cursor-pointer transition-colors duration-[var(--duration-fast)] group/priv w-max rounded px-1 -ml-1 py-0.5 border border-transparent"
                             :class="copiedStates[node.id + '-priv'] ? 'bg-success-bg text-success-text border-success-border' : 'text-text-m hover:text-text-h hover:bg-secondary-hover'">
                            <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <rect x="2" y="14" width="8" height="6" rx="2" ry="2"/>
                                <rect x="14" y="14" width="8" height="6" rx="2" ry="2"/>
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 14v-2a2 2 0 012-2h8a2 2 0 012 2v2M12 2v8"/>
                                <rect x="8" y="2" width="8" height="6" rx="2" ry="2"/>
                            </svg>
                            <span class="font-mono truncate">{{ copiedStates[node.id + '-priv'] ? 'Copied!' : (node.privateIp || 'Unknown') }}</span>
                        </div>
                    </div>

                    <div class="flex flex-wrap gap-1.5 mb-4">
                        <span v-for="(tag, index) in node.tags" :key="tag" 
                            class="px-2 py-0.5 bg-bg-inset border border-border-default text-text-m text-sm font-medium rounded-md tracking-wider flex items-center gap-1.5 group/tag transition-colors hover:border-text-m">
                            {{ tag }}
                            <button @click.stop="removeTag(node, index)" class="opacity-0 group-hover/tag:opacity-100 text-text-m hover:text-danger-main transition-all outline-none focus:opacity-100">
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
                                class="pl-5 pr-2 py-0.5 bg-input-bg border border-primary-main text-text-h text-sm rounded-md focus:outline-none ring-1 ring-focus-ring w-32 shadow-sm transition-all placeholder:text-text-m/50"
                                placeholder="tag name..."
                            />
                        </div>
                        
                        <button v-else @click.stop="enableTagEdit(node.id)" 
                                class="px-1.5 py-0.5 border border-dashed border-border-default text-text-m text-sm rounded-md hover:text-text-h hover:border-text-m transition-colors outline-none focus:ring-1 focus:ring-focus-ring">
                            + Tag
                        </button>
                    </div>

                    <div v-if="!node.isAwaitingCheckIn" class="mt-auto bg-bg-surface border border-border-default rounded-lg p-3">
                        <div class="flex items-center justify-between mb-2">
                            <span class="text-sm font-normal text-text-h">Deployed Sensors</span>
                            <span class="text-sm text-text-m">{{ node.onlineSensors }} / {{ node.totalSensors }} Online</span>
                        </div>
                        
                        <div class="flex flex-wrap gap-1.5 mt-1">
                            <template v-if="isManifestLoading">
                                <div v-for="i in Math.min(node.totalSensors, 3)" :key="i" class="px-2 py-1 rounded-md text-sm bg-secondary-main border border-border-default animate-pulse w-16 h-6"></div>
                            </template>
                            <template v-else>
                                <div v-for="summary in node.sensorSummary" :key="summary.type" class="relative group/tooltip">
                                    <span class="px-2 py-1 rounded-md text-sm font-medium flex items-center gap-1.5 border border-border-default bg-secondary-main text-text-m cursor-help hover:border-text-m transition-colors">
                                        <span class="text-text-h">{{ summary.count }}</span> {{ summary.type }}
                                    </span>

                                    <div class="absolute bottom-full left-0 mb-2 w-max min-w-[180px] opacity-0 invisible group-hover/tooltip:opacity-100 group-hover/tooltip:visible transition-all duration-fast z-dropdown bg-bg-surface border border-border-default shadow-lg rounded-md p-2 pointer-events-none">
                                        <div class="text-sm font-medium text-text-h mb-2">{{ summary.type }}</div>
                                        <div class="flex flex-col gap-2">
                                            <div v-for="s in summary.sensors" :key="s.name" class="flex items-center gap-2 text-sm text-text-h">
                                                <BaseStatusDot :status="s.status" />
                                                <span class="font-mono truncate">{{ s.name }}</span>
                                            </div>
                                        </div>
                                        
                                        <div class="absolute top-full left-6 -mt-px border-4 border-transparent border-t-border-default"></div>
                                        <div class="absolute top-full left-6 -mt-[2px] border-4 border-transparent border-t-bg-surface"></div>
                                    </div>
                                </div>
                            </template>
                            
                            <span v-if="node.sensorSummary.length === 0 && !isManifestLoading" class="text-sm text-text-m italic">
                                No sensors deployed.
                            </span>
                        </div>
                    </div>
                    <div v-else class="mt-auto bg-disabled-bg border border-dashed border-disabled-border rounded-lg p-4 flex flex-col items-center justify-center text-center opacity-70">
                        <svg class="w-6 h-6 text-text-h mb-2 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                        </svg>
                        <span class="text-sm font-medium text-text-h">Awaiting Initial Check-in</span>
                        <span class="text-sm text-text-m mt-1 max-w-[200px]">Deploy your first Sensor!</span>
                    </div>

                </div>

                <template #footer>
                    <button @click="handleOpenNodeDetail(node.id)" class="w-full py-3 text-sm font-medium text-text-h hover:text-text-h bg-bg-secondary hover:bg-secondary-hover border-t border-border-default rounded-b-xl transition-colors flex items-center justify-center gap-2 group/btn outline-none">
                        <span>{{ node.totalSensors === 0 ? 'Install First Sensor' : 'Manage Node & Sensors' }}</span>
                        <svg class="w-4 h-4 transition-transform duration-normal group-hover/btn:translate-x-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
                    </button>
                </template>

            </BaseWidget>

        </div>
    </div>
    
    <!-- Deploy Modal -->
    <BaseModal :show="showDeployModal" @close="closeDeployModal" title="Deploy New Node">
        <div v-if="deployStep === 1" class="space-y-5">
            <p class="text-sm text-text-m leading-normal">
                Create a logical node in the hub before installing the agent on your server.
            </p>
            
            <div>
                <label class="block text-sm font-medium text-text-h mb-1.5">Node Alias</label>
                <BaseInput v-model="newNodeForm.alias" placeholder="e.g., AWS-East-Gateway" autofocus />
            </div>

            <div class="flex justify-end gap-3 pt-5 border-t border-border-default mt-6">
                <BaseButton variant="ghost" @click="closeDeployModal">Cancel</BaseButton>
                <BaseButton variant="primary" @click="handleDeploySubmit" :disabled="!newNodeForm.alias">Create Node</BaseButton>
            </div>
        </div>

        <div v-else class="space-y-6">
            <div class="flex items-center gap-3 text-success-text bg-success-bg border border-success-border p-3.5 rounded-md">
                <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                <span class="text-sm font-medium">Node created successfully.</span>
            </div>

            <div>
                <div class="flex items-center justify-between mb-1.5">
                    <label class="block text-sm font-medium text-text-h">Node API Key</label>
                    <BaseButton 
                        variant="ghost" 
                        class="!py-1 !px-2 !text-xs transition-colors" 
                        :class="copiedStates['modal-key'] ? '!text-success-main hover:!text-success-main' : ''"
                        @click="handleCopy('modal-key', generatedNodeKey)"
                    >
                        <span class="flex items-center gap-1.5">
                            <svg v-if="!copiedStates['modal-key']" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                            <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                            {{ copiedStates['modal-key'] ? 'Copied!' : 'Copy' }}
                        </span>
                    </BaseButton>
                </div>
                <div class="flex items-center gap-2">
                    <code class="flex-1 block px-3 py-2.5 bg-bg-inset border border-border-default rounded-md text-sm font-mono text-text-h truncate select-all">
                        {{ generatedNodeKey }}
                    </code>
                </div>
                <p class="text-sm text-text-m font-medium mt-2">Save this key — it won't be shown again.</p>
            </div>

            <div>
                <div class="flex items-center justify-between mb-1.5">
                    <label class="block text-sm font-medium text-text-h">Wizard Installation Command</label>
                    <BaseButton 
                        variant="ghost" 
                        class="!py-1 !px-2 !text-xs transition-colors" 
                        :class="copiedStates['modal-cmd'] ? '!text-success-main hover:!text-success-main' : ''"
                        @click="handleCopy('modal-cmd', `curl -sL https://hub.honeywire.local/wizard.sh | bash -s -- --key ${generatedNodeKey}`)"
                    >
                        <span class="flex items-center gap-1.5">
                            <svg v-if="!copiedStates['modal-cmd']" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                            <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                            {{ copiedStates['modal-cmd'] ? 'Copied!' : 'Copy' }}
                        </span>
                    </BaseButton>
                </div>
                <code class="block w-full p-4 bg-bg-inset border border-border-default rounded-md text-sm font-mono text-text-m whitespace-pre-wrap break-all leading-normal select-all">curl -sL https://hub.honeywire.local/wizard.sh | bash -s -- --key {{ generatedNodeKey }}</code>
            </div>

            <div class="flex justify-end pt-5 border-t border-border-default mt-6">
                <BaseButton variant="secondary" @click="closeDeployModal">Done</BaseButton>
            </div>
        </div>
    </BaseModal>

</template>