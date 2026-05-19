<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAppStore } from '../stores/app'
import { useFleetStore } from '../stores/fleet'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import BaseWidget from '../components/ui/layout/BaseWidget.vue'
import BaseStatusDot from '../components/ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../components/ui/navigation/BaseMeatballMenu.vue'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import BaseModal from '../components/ui/feedback/BaseModal.vue'
import BaseInput from '../components/ui/forms/BaseInput.vue'

const appStore = useAppStore()
const fleetStore = useFleetStore()

onMounted(() => {
    fleetStore.fetchFleet()
})

// --- DEPLOY MODAL STATE ---
const showDeployModal = ref(false)
const deployStep = ref(1) 
const newNodeForm = ref({ alias: '' })
const tagInput = ref('')
const tagsList = ref([])
const generatedNodeKey = ref('')

// --- EDIT MODAL STATE ---
const showEditModal = ref(false)
const editNodeForm = ref({ id: '', alias: '', publicIp: '', privateIp: '', apiKey: '' })

// --- TAGS LOGIC ---
const addTag = () => {
    const val = tagInput.value.trim()
    if (val && !tagsList.value.includes(val)) {
        tagsList.value.push(val)
    }
    tagInput.value = ''
}

const removeTag = (index) => {
    tagsList.value.splice(index, 1)
}

// --- DATA MAPPING ---
const displayNodes = computed(() => {
    return fleetStore.nodes.map(node => {
        const sensorsList = node.installedSensors || []
        const onlineSensors = sensorsList.filter(s => s.status === 'up').length
        const totalSensors = sensorsList.length
        const isSilenced = totalSensors > 0 && sensorsList.every(s => s.isSilenced)

        // Grouping for the UI tooltip
        const sensorSummary = totalSensors > 0
            ? [{ type: 'Deployed', count: totalSensors, sensors: sensorsList.map(s => ({ name: s.display || s.name, status: s.status })) }]
            : []

        return {
            ...node,
            totalSensors,
            onlineSensors,
            isSilenced,
            sensorSummary,
            hasUpdate: false,
            // A node is pending if it has no sensors or has explicitly just been created
            isAwaitingCheckIn: node.status === 'pending' || (!node.lastHeartbeat && totalSensors === 0)
        }
    })
})

// --- API ACTIONS ---
const handleDeploySubmit = async () => {
    if (!newNodeForm.value.alias) return

    try {
        const response = await fetch('/api/v1/nodes', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ alias: newNodeForm.value.alias, tags: tagsList.value }),
        })

        if (!response.ok) throw new Error('Failed to create node')

        const data = await response.json()
        generatedNodeKey.value = data.apiKey
        deployStep.value = 2
        await fleetStore.fetchFleet()
    } catch (err) {
        console.error('Create node failed:', err)
        alert('Could not create node. Please try again.')
    }
}

const handleOpenEditNode = (node) => {
    editNodeForm.value = { 
        id: node.id, 
        alias: node.alias, 
        publicIp: node.publicIp || '', 
        privateIp: node.privateIp || '',
        apiKey: node.apiKey || ''
    }
    tagsList.value = [...(node.tags || [])] 
    showEditModal.value = true
}

const triggerManualSync = async (node) => {
    if (!node) return
    const apiKey = node.apiKey
    if (!apiKey) {
        alert('Unable to sync this node because the node API key is missing.')
        return
    }

    try {
        const response = await fetch('/api/v1/nodes/compose', {
            headers: { Authorization: `Bearer ${apiKey}` },
        })
        if (!response.ok) {
            throw new Error(`Sync failed: ${response.status}`)
        }
        await fleetStore.fetchFleet()
        alert('Pending sync request sent successfully. The node compose was generated and pending config has been updated.')
    } catch (err) {
        console.error('Failed to sync node:', err)
        alert('Unable to sync this node. Please try again.')
    }
}

const handleEditSubmit = async () => {
    try {
        const response = await fetch(`/api/v1/nodes/${editNodeForm.value.id}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                alias: editNodeForm.value.alias, 
                tags: tagsList.value,
                publicIp: editNodeForm.value.publicIp,
                privateIp: editNodeForm.value.privateIp
            }),
        })

        if (!response.ok) throw new Error('Failed to update node')
        
        showEditModal.value = false
        await fleetStore.fetchFleet()
    } catch (err) {
        console.error('Update node failed:', err)
    }
}

const closeDeployModal = () => {
    showDeployModal.value = false
    setTimeout(() => {
        deployStep.value = 1
        newNodeForm.value.alias = ''
        tagsList.value = []
        tagInput.value = ''
        generatedNodeKey.value = ''
    }, 300)
}

const handleSilenceNode = (nodeId) => fleetStore.silenceNode(nodeId)
const handleForgetNode = (nodeId) => fleetStore.deleteNode(nodeId)

const handleOpenNodeDetail = (nodeId) => {
    fleetStore.selectTarget(nodeId)
    appStore.currentView = 'node-detail'
}

// --- Copy Animation Logic ---
const copiedStates = ref({})

const handleCopy = async (id, text) => {
    if (!text) return
    try {
        await navigator.clipboard.writeText(text)
        copiedStates.value[id] = true
        setTimeout(() => {
            copiedStates.value[id] = false
        }, 2000)
    } catch (err) {
        console.error('Failed to copy text', err)
    }
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

        <div class="grid grid-cols-1 lg:grid-cols-2 2xl:grid-cols-3 gap-5 auto-rows-max">
            
            <BaseWidget v-for="node in displayNodes" :key="node.id" class="flex flex-col h-full min-h-[280px]">
                
                <template #header>
                    <div class="flex items-start justify-between w-full">
                        <div class="flex items-center gap-2.5 min-w-0">
                            <BaseStatusDot :status="node.status" />
                            <div>
                                <div class="flex items-center gap-2">
                                    <h3 class="text-base font-medium text-text-h truncate max-w-[200px]" :title="node.alias">
                                        {{ node.alias }}
                                    </h3>
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
                            <button @click="handleOpenEditNode(node)" class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">
                                Node Settings
                            </button>
                            <button @click="handleForgetNode(node.id)" class="w-full text-left px-3 py-2 text-sm text-danger-text hover:bg-danger-bg transition-colors border-t border-border-default mt-1 pt-2">
                                Delete Node
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
                        <span v-for="tag in node.tags" :key="tag" class="px-2 py-0.5 bg-bg-inset border border-border-default text-text-m text-sm font-medium rounded-md tracking-wider">
                            {{ tag }}
                        </span>
                        <button @click.stop="openEditModal(node)" class="px-1.5 py-0.5 border border-dashed border-border-default text-text-m text-sm rounded-md hover:text-text-h transition-colors">
                            + Tag
                        </button>
                    </div>

                    <div v-if="node.hasPendingConfig" class="mb-4 flex items-center justify-between bg-high/10 border border-high/20 rounded-md px-3 py-2">
                        <div class="flex items-center gap-2 text-high text-sm font-medium">
                            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                            </svg>
                            Pending Sync
                        </div>
                        <BaseButton variant="secondary" class="gap-2 text-sm text-text-h" @click="triggerManualSync(node)">
                            Sync Now
                        </BaseButton>
                    </div>

                    <div v-if="!node.isAwaitingCheckIn" class="mt-auto bg-bg-surface border border-border-default rounded-lg p-3">
                        <div class="flex items-center justify-between mb-2">
                            <span class="text-sm font-normal text-text-h">Deployed Sensors</span>
                            <span class="text-sm text-text-m">{{ node.onlineSensors }} / {{ node.totalSensors }} Online</span>
                        </div>
                        
                        <div class="flex flex-wrap gap-1.5 mt-1">
                            <div v-for="summary in node.sensorSummary" :key="summary.type" class="relative group/tooltip">
                                <span class="px-2 py-1 rounded-md text-sm font-medium flex items-center gap-1.5 border border-border-default bg-secondary-main text-text-m cursor-help hover:border-text-m transition-colors">
                                    <span class="text-text-h">{{ summary.count }}</span> {{ summary.type }}
                                </span>

                                <div class="absolute bottom-full left-0 mb-2 w-max min-w-[180px] opacity-0 invisible group-hover/tooltip:opacity-100 group-hover/tooltip:visible transition-all duration-fast z-dropdown bg-bg-surface border border-border-default shadow-lg rounded-md p-2 pointer-events-none">
                                    <div class="text-xs font-medium text-text-h mb-2">{{ summary.type }} Sensors</div>
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
                            
                            <span v-if="node.sensorSummary.length === 0" class="text-sm text-text-m italic">
                                No sensors deployed.
                            </span>
                        </div>
                    </div>
                    <div v-else class="mt-auto bg-disabled-bg border border-dashed border-disabled-border rounded-lg p-4 flex flex-col items-center justify-center text-center opacity-70">
                        <svg class="w-6 h-6 text-text-h mb-2 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                        </svg>
                        <span class="text-sm font-medium text-text-h">Awaiting Initial Check-in</span>
                        <span class="text-xs text-text-m mt-1 max-w-[200px]">Deploy your first Sensor!</span>
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
    
    <BaseModal :show="showDeployModal" @close="closeDeployModal" title="Deploy New Node">
        <div v-if="deployStep === 1" class="space-y-5">
            <p class="text-sm text-text-m leading-normal">
                Create a logical node in the hub before installing the agent on your server.
            </p>
            
            <div>
                <label class="block text-sm font-medium text-text-h mb-1.5">Node Alias</label>
                <BaseInput v-model="newNodeForm.alias" placeholder="e.g., AWS-East-Gateway" autofocus />
            </div>
            
            <div>
                <label class="block text-sm font-medium text-text-h mb-1.5">Tags (Optional)</label>
                <div class="flex flex-col gap-2.5">
                    <div v-if="tagsList.length > 0" class="flex flex-wrap gap-1.5">
                        <span v-for="(tag, index) in tagsList" :key="tag" 
                            class="flex items-center gap-1.5 px-2 py-1 bg-bg-inset border border-border-default text-text-m text-xs font-medium rounded-md tracking-wider">
                            {{ tag }}
                            <button @click="removeTag(index)" class="text-text-m hover:text-text-h hover:text-danger-text transition-colors outline-none">
                                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                            </button>
                        </span>
                    </div>
                    
                    <BaseInput 
                        v-model="tagInput" 
                        @keydown.enter.prevent="addTag" 
                        placeholder="Type a tag and press Enter..." 
                    />
                </div>
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
                <p class="text-xs text-text-m font-medium mt-2">This key can also be viewed later in Node Settings.</p>
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

    <BaseModal :show="showEditModal" @close="showEditModal = false" title="Node Settings">
        <div class="space-y-4">
            <div class="mb-2">
                <div class="flex items-center justify-between mb-1.5">
                    <label class="block text-sm font-medium text-text-h">API Key</label>
                    <BaseButton variant="ghost" class="!py-1 !px-2 !text-xs transition-colors" :class="copiedStates['edit-key'] ? '!text-success-main hover:!text-success-main' : ''" @click="handleCopy('edit-key', editNodeForm.apiKey)">
                        <span class="flex items-center gap-1.5">
                            <svg v-if="!copiedStates['edit-key']" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                            <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                            {{ copiedStates['edit-key'] ? 'Copied!' : 'Copy Key' }}
                        </span>
                    </BaseButton>
                </div>
                <code class="block w-full px-3 py-2 bg-bg-inset border border-border-default rounded-md text-sm font-mono text-text-m truncate select-all">{{ editNodeForm.apiKey || 'Unavailable' }}</code>
            </div>

            <div>
                <label class="block text-sm font-medium text-text-h mb-1.5">Node Alias</label>
                <BaseInput v-model="editNodeForm.alias" />
            </div>
            
            <div class="grid grid-cols-2 gap-4">
                <div>
                    <label class="block text-sm font-medium text-text-h mb-1.5">Public IP</label>
                    <BaseInput v-model="editNodeForm.publicIp" placeholder="e.g. 198.51.100.14" />
                </div>
                <div>
                    <label class="block text-sm font-medium text-text-h mb-1.5">Private IP</label>
                    <BaseInput v-model="editNodeForm.privateIp" placeholder="e.g. 10.0.0.5" />
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

            <div class="flex justify-end gap-3 pt-5 border-t border-border-default mt-6">
                <BaseButton variant="ghost" @click="showEditModal = false">Cancel</BaseButton>
                <BaseButton variant="primary" @click="handleEditSubmit" :disabled="!editNodeForm.alias">Save Changes</BaseButton>
            </div>
        </div>
    </BaseModal>

</template>