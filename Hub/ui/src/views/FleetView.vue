<script setup>
import { ref } from 'vue'
import { useAppStore } from '../stores/app'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import BaseWidget from '../components/ui/layout/BaseWidget.vue'
import BaseStatusDot from '../components/ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../components/ui/navigation/BaseMeatballMenu.vue'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import BaseModal from '../components/ui/feedback/BaseModal.vue'
import BaseInput from '../components/ui/forms/BaseInput.vue'

const appStore = useAppStore()

// --- MODAL STATE ---
const showDeployModal = ref(false)
const deployStep = ref(1) 
const newNodeForm = ref({ alias: '' })
const tagInput = ref('')
const tagsList = ref([])
const generatedNodeKey = ref('')

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

const handleDeploySubmit = () => {
    generatedNodeKey.value = 'hw_key_' + Math.random().toString(36).substring(2, 15)
    deployStep.value = 2
}

const closeDeployModal = () => {
    showDeployModal.value = false
    setTimeout(() => {
        deployStep.value = 1
        newNodeForm.value.alias = ''
        tagsList.value = []
        tagInput.value = ''
    }, 300)
}
// -------------------

const ghostNode = {
    id: 'node-789',
    alias: 'DMZ-Honeypot-Alpha',
    status: 'pending', // You may need to add 'pending' to your BaseStatusDot (gray/pulse)
    publicIp: null,
    privateIp: null,
    hasUpdate: false,
    hasPendingConfig: false,
    tags: ['DMZ', 'New'],
    sensorSummary: [],
    totalSensors: 0,
    onlineSensors: 0,
    isSilenced: false,
    isAwaitingCheckIn: true // <-- Flag to trigger the ghost UI
}

// Mock Data
const mockNodes = ref([
    {
        id: 'node-123',
        alias: 'AWS-East-Gateway',
        status: 'up',
        publicIp: '203.0.113.42',
        privateIp: '10.0.1.4',
        hasUpdate: false,
        hasPendingConfig: true, 
        tags: ['DMZ', 'Production'],
        sensorSummary: [
            { type: 'Network', count: 3, sensors: [{ name: 'hw-tcp-tarpit', status: 'up' }, { name: 'hw-udp-reflector', status: 'up' }, { name: 'hw-port-scan', status: 'down' }] },
            { type: 'Web', count: 1, sensors: [{ name: 'hw-web-router', status: 'up' }] },
            { type: 'File', count: 1, sensors: [{ name: 'hw-file-canary', status: 'up' }] }
        ],
        totalSensors: 5,
        onlineSensors: 5,
        isSilenced: false
    },
    {
        id: 'node-456',
        alias: 'Internal-DB-Honeypot',
        status: 'degraded',
        publicIp: '198.51.100.12',
        privateIp: '192.168.1.100',
        hasUpdate: true, 
        hasPendingConfig: false,
        tags: ['Internal', 'VLAN-40'],
        sensorSummary: [
            { type: 'Database', count: 2, sensors: [{ name: 'hw-sql-tarpit', status: 'up' }, { name: 'hw-redis-canary', status: 'down' }] },
            { type: 'Network', count: 1, sensors: [{ name: 'hw-tcp-tarpit', status: 'up' }] }
        ],
        totalSensors: 3,
        onlineSensors: 2,
        isSilenced: true
    },
    {
        id: 'unassigned',
        alias: 'Unassigned Sensors',
        status: 'down',
        publicIp: null,
        privateIp: null,
        hasUpdate: false,
        hasPendingConfig: false,
        tags: [],
        sensorSummary: [
            { type: 'File', count: 2, sensors: [{ name: 'hw-file-canary', status: 'down' }, { name: 'hw-file-canary-2', status: 'down' }] }
        ],
        totalSensors: 2,
        onlineSensors: 0,
        isSilenced: false
    },
    ghostNode
])

// --- Copy Animation Logic ---
// We use an object to track multiple copy states (e.g., 'node-123-pub', 'node-123-priv')
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
            
            <BaseWidget v-for="node in mockNodes" :key="node.id" class="flex flex-col h-full min-h-[280px]">
                
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
                            <button class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">
                                {{ node.isSilenced ? 'Unsilence Node' : 'Silence Node' }}
                            </button>
                            <button class="w-full text-left px-3 py-2 text-sm text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors">
                                Rename Node
                            </button>
                            <button class="w-full text-left px-3 py-2 text-sm text-danger-text hover:bg-danger-bg transition-colors border-t border-border-default mt-1 pt-2">
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

                    <div v-if="node.tags && node.tags.length > 0" class="flex flex-wrap gap-1.5 mb-4">
                        <span v-for="tag in node.tags" :key="tag" class="px-2 py-0.5 bg-bg-inset border border-border-default text-text-m text-sm font-medium rounded-md tracking-wider">
                            {{ tag }}
                        </span>
                        <button class="px-1.5 py-0.5 border border-dashed border-border-default text-text-m text-sm rounded-md hover:text-text-h transition-colors">
                            + Tag
                        </button>
                    </div>

                    <div v-if="node.hasPendingConfig" class="mb-4 flex items-center justify-between bg-high/10 border border-high/20 rounded-md px-3 py-2">
                        <div class="flex items-center gap-2 text-high text-sm font-medium">
                            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                            </svg>
                            Pending Deployment
                        </div>
                        <BaseButton variant="secondary" class="gap-2 text-sm text-text-h">
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
                            <span class="text-xs text-text-m mt-1 max-w-[200px]">Deploy your first Sensor!.</span>
                        </div>

                </div>

                <template #footer>
                    <button @click="appStore.currentView = 'node-detail'" class="w-full py-3 text-sm font-medium text-text-h hover:text-text-h bg-bg-secondary hover:bg-secondary-hover border-t border-border-default rounded-b-xl transition-colors flex items-center justify-center gap-2 group/btn outline-none">
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
                <p class="text-xs text-success-main font-medium mt-2">This key can be viewed later in Node Management.</p>
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