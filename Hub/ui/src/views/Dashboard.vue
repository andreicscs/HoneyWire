<script setup>
import TrafficFilters from '../components/dashboard/TrafficFilters.vue'
import SeverityChart from '../components/dashboard/SeverityChart.vue'
import UptimeHeatmap from '../components/dashboard/UptimeHeatmap.vue'
import EventTable from '../components/dashboard/EventTable.vue'
import ThreatVelocity from '../components/dashboard/ThreatVelocity.vue'
import { ref } from 'vue'
import { useFleetStore } from '../stores/fleet'

const fleetStore = useFleetStore()
const showProvisionModal = ref(false)
const provisionToken = ref('')
const provisionNodeId = ref('')
const isGeneratingToken = ref(false)
const currentHost = ref(window.location.host)

const generateToken = async () => {
    const alias = window.prompt('Enter a friendly alias for the new node', `New Node ${Date.now()}`)
    if (!alias) return

    isGeneratingToken.value = true
    try {
        const result = await fleetStore.createNode(alias)
        provisionToken.value = result.apiKey
        provisionNodeId.value = result.nodeId
        showProvisionModal.value = true
    } catch (err) {
        alert('Failed to create node. Please try again.')
    } finally {
        isGeneratingToken.value = false
    }
}

const copyCommand = () => {
    const cmd = `./wizard --link http://${currentHost.value} ${provisionToken.value}`
    navigator.clipboard.writeText(cmd)

    const btn = document.getElementById('copy-cmd-btn')
    const originalText = btn.innerHTML
    
    btn.innerHTML = 'Copied!'
    btn.classList.add('bg-success-bg', 'text-success-text', 'border-success-border')
    btn.classList.remove('bg-bg-surface', 'text-text-h', 'border-border-default', 'hover:bg-button-hover')
    
    setTimeout(() => { 
        btn.innerHTML = originalText 
        btn.classList.remove('bg-success-bg', 'text-success-text', 'border-success-border')
        btn.classList.add('bg-bg-surface', 'text-text-h', 'border-border-default', 'hover:bg-button-hover')
    }, 2000)
}
</script>

<template>
    <div class="flex flex-col gap-4 sm:gap-6 h-full max-w-[1600px] mx-auto w-full px-2 sm:px-4 lg:px-6 mb-16">

        <TrafficFilters />

        <div class="flex flex-wrap gap-4 sm:gap-6 shrink-0">
            <div class="flex-[1_1_350px] min-w-[100%] sm:min-w-[350px] h-[320px] lg:h-[340px] shrink-0">
                <ThreatVelocity />
            </div>

            <div class="flex-[1_1_280px] min-w-[100%] sm:min-w-[280px] max-w-[450px] mx-auto h-[320px] lg:h-[340px] shrink-0">
                <SeverityChart />
            </div>
            
            <div class="flex-[1.5_1_450px] min-w-[100%] lg:min-w-[450px] h-[320px] lg:h-[340px] shrink-0">
                <UptimeHeatmap />
            </div>
        </div>

        <div class="flex-1 min-h-0 pb-6 mt-2">
            <EventTable />
        </div>
    </div>

    <Teleport to="body">
        <transition enter-active-class="transition duration-200 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-150 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
            <div v-if="showProvisionModal" class="fixed inset-0 z-[200] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
                <div class="bg-bg-surface border border-border-default rounded-lg shadow-lg max-w-lg w-full overflow-hidden flex flex-col transform transition-all">
                    <div class="p-5 border-b border-border-default flex justify-between items-center">
                        <h3 class=" text-text-h">Provision New Node</h3>
                        <button @click="showProvisionModal = false" class="text-text-m hover:text-text-h transition-colors hover:bg-button-hover rounded-full p-1 -mr-1">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                        </button>
                    </div>
                    
                    <div class="p-5 space-y-4">
                        <p class="text-sm text-text-m">
                            Run the following command on the server you want to monitor. This token is single-use and expires in 15 minutes.
                        </p>
                        
                        <div class="space-y-3">
                            <div class="text-xs text-text-m font-medium">Node ID: <span class="font-mono text-text-h">{{ provisionNodeId }}</span></div>
                            <div class="bg-bg-inset border border-border-default rounded-md p-4 relative group flex flex-col gap-3">
                                <code class="text-success-main text-xs font-mono break-all leading-relaxed">
                                    ./wizard --link http://{{ currentHost }} {{ provisionToken }}
                                </code>
                                <button id="copy-cmd-btn" @click="copyCommand"
                                        class="self-end px-3 py-1.5 rounded-md bg-bg-surface border border-border-default text-text-h text-sm   tracking-wider hover:bg-button-hover transition-colors shadow-sm active:scale-95">
                                    Copy
                                </button>
                            </div>
                        </div>
                    </div>

                    <div class="p-4 bg-bg-base border-t border-border-default flex justify-end">
                        <button @click="showProvisionModal = false" class="px-4 py-2 bg-bg-surface hover:bg-button-hover text-text-h border border-border-default text-base  rounded-md transition-colors shadow-sm">
                            Close
                        </button>
                    </div>
                </div>
            </div>
        </transition>
    </Teleport>
</template>