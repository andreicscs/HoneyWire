<script setup>
    import { useSentinel } from '../api/useSentinel'
    import TrafficFilters from '../components/TrafficFilters.vue'
    import SeverityChart from '../components/SeverityChart.vue'
    import UptimeHeatmap from '../components/UptimeHeatmap.vue'
    import EventTable from '../components/EventTable.vue'
    import ThreatVelocity from '../components/ThreatVelocity.vue'

import { ref } from 'vue' // (Make sure ref is imported if it isn't already)
    const showProvisionModal = ref(false)
const provisionToken = ref('')
const isGeneratingToken = ref(false)
const currentHost = ref(window.location.host) // <-- ADD THIS LINE

const generateToken = async () => {
    isGeneratingToken.value = true
    try {
        const response = await fetch('/api/v1/tokens/generate', { method: 'POST' })
        if (!response.ok) throw new Error('Failed to generate token')
        
        const data = await response.json()
        provisionToken.value = data.token
        showProvisionModal.value = true
    } catch (err) {
        console.error("Token error:", err)
        alert("Failed to generate token.")
    } finally {
        isGeneratingToken.value = false
    }
}

    //---------------------------------------//
    const { 
        fleet, selectedNode, selectedSensor, events, uptimeData, activeTimeframe, velocityTimeframe, 
        overallUptime, viewingArchive, archiveAll,
        activeEvent, isActiveSensorSilenced, archiveEvent, toggleSilence, forgetSensor, markEventRead,
        silenceNode, forgetNode
    } = useSentinel()

    const handleNodeSelect = (nodeId) => { 
        // State toggle: if clicking the active node, clear everything
        if (selectedNode.value === nodeId && !selectedSensor.value) {
            selectedNode.value = null
            selectedSensor.value = null
        } else {
            selectedNode.value = nodeId
            selectedSensor.value = null // Clear sensor focus when clicking a node pill
        }
    }

    const handleSensorSelect = (sensorId, nodeId) => { 
        // State toggle: if clicking the active sensor, clear everything
        if (selectedSensor.value === sensorId && selectedNode.value === nodeId) {
            selectedSensor.value = null
            selectedNode.value = null
        } else {
            selectedSensor.value = sensorId
            selectedNode.value = nodeId 
        }
    }
</script>

<template>
    <div class="flex flex-col gap-4 sm:gap-6 h-full max-w-[1600px] mx-auto w-full px-2 sm:px-4 lg:px-6">
        

        <div class="flex justify-end w-full mb-4 px-2 sm:px-4 lg:px-6">
            <button @click="generateToken" :disabled="isGeneratingToken"
                    class="bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold py-2 px-4 rounded-md shadow-sm transition-colors flex items-center gap-2 disabled:opacity-50">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path></svg>
                {{ isGeneratingToken ? 'Generating...' : 'Add Node' }}
            </button>
        </div>

        <TrafficFilters 
            :fleet="fleet" 
            :selectedNode="selectedNode" 
            :selectedSensor="selectedSensor"
            @select-node="handleNodeSelect" 
            @silence-node="silenceNode"
            @forget-node="forgetNode"
        />

        <div class="flex flex-wrap gap-4 sm:gap-6 shrink-0">
            <div class="flex-[1_1_350px] min-w-[100%] sm:min-w-[350px] h-[320px] lg:h-[340px] shrink-0">
                <ThreatVelocity 
                    :events="events"
                    :activeTimeframe="velocityTimeframe"
                    @update:timeframe="t => velocityTimeframe = t"
                />
            </div>

            <div class="flex-[1_1_280px] min-w-[100%] sm:min-w-[280px] max-w-[450px] mx-auto h-[320px] lg:h-[340px] shrink-0">
                <SeverityChart :events="events" />
            </div>
            
            <div class="flex-[1.5_1_450px] min-w-[100%] lg:min-w-[450px] h-[320px] lg:h-[340px] shrink-0">
                <UptimeHeatmap 
                    :uptimeData="uptimeData"
                    :overallUptime="overallUptime"
                    :activeTimeframe="activeTimeframe"
                    :fleet="fleet"
                    :selectedNode="selectedNode"
                    :selectedSensor="selectedSensor"
                    @update:timeframe="t => activeTimeframe = t"
                    @select-sensor="(sId, nId) => { selectedSensor = sId; selectedNode = nId }" 
                    @select-node="(nId) => { selectedNode = nId; selectedSensor = null }"
                    @toggle-silence="toggleSilence"
                    @forget-sensor="forgetSensor"
                />
            </div>
        </div>

        <div class="flex-1 min-h-0 pb-6 mt-2">
            <EventTable 
                :events="events" 
                :viewingArchive="viewingArchive"
                @archive-all="archiveAll"
                @archive-event="archiveEvent"
                @mark-read="markEventRead"
            />
        </div>
    </div>


    <Teleport to="body">
    <div v-if="showProvisionModal" class="fixed inset-0 z-[200] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
        <div class="bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-700 rounded-lg shadow-2xl max-w-lg w-full overflow-hidden flex flex-col">
            <div class="p-5 border-b border-slate-100 dark:border-zinc-800 flex justify-between items-center">
                <h3 class="font-bold text-slate-800 dark:text-zinc-100">Provision New Node</h3>
                <button @click="showProvisionModal = false" class="text-slate-400 hover:text-slate-600 dark:hover:text-zinc-300">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                </button>
            </div>
            
            <div class="p-5 space-y-4">
                <p class="text-sm text-slate-600 dark:text-zinc-400">
                    Run the following command on the server you want to monitor. This token is single-use and expires in 15 minutes.
                </p>
                
                <div class="bg-slate-900 rounded-md p-3 relative group">
                    <code class="text-emerald-400 text-xs font-mono break-all">
                        ./wizard --link http://{{ currentHost }} {{ provisionToken }}
                    </code>
                </div>
            </div>

            <div class="p-4 bg-slate-50 dark:bg-zinc-800/50 flex justify-end">
                <button @click="showProvisionModal = false" class="px-4 py-2 bg-slate-200 dark:bg-zinc-700 hover:bg-slate-300 dark:hover:bg-zinc-600 text-slate-800 dark:text-zinc-200 text-sm font-semibold rounded transition-colors">
                    Close
                </button>
            </div>
        </div>
    </div>
</Teleport>
</template>