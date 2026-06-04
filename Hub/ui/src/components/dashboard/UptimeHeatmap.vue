<script setup lang="ts">
import { ref, nextTick, watch, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useFleetStore } from '../../stores/Fleet/fleet'
import BaseTimeFilter from '../ui/forms/BaseTimeFilter.vue'
import BaseWidget from '../ui/layout/BaseWidget.vue'
import BaseLegend from '../ui/feedback/BaseLegend.vue'
import BaseStatusDot from '../ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../ui/navigation/BaseMeatballMenu.vue'
import { formatSensorId } from '../../utils/formatSensorId'


const fleetStore = useFleetStore()
const { uptimeData, selectedNode, selectedSensor, activeTimeframe, hydratedUptimeGroups } = storeToRefs(fleetStore)

const scrollArea = ref<HTMLElement | null>(null)
const canScrollDown = ref(false)
const worstWarningBelow = ref<string | null>(null)

const handleSilence = (nodeId: string, sensorId: string, currentlySilenced: boolean) => fleetStore.toggleSilence(nodeId, sensorId, !currentlySilenced)
const handleForget = async (nodeId: string, sensorId: string) => {
    if (!confirm('Remove this sensor? The node will be marked for deployment sync.')) return
    const res = await fleetStore.removeSensor(nodeId, sensorId)
    if (res.success) {
        await fleetStore.fetchUptime()
    } else {
        alert(res.error || 'Failed to remove sensor.')
    }
}

const checkScroll = () => {
    if (!scrollArea.value) {
        return
    }
    const container = scrollArea.value
    
    const currentBottom = Math.ceil(container.scrollTop + container.clientHeight)
    canScrollDown.value = currentBottom < (container.scrollHeight - 15)

    let worstStatus = null
    const warningNodes = container.querySelectorAll('.has-warnings')

    if (warningNodes.length > 0) {
        const containerRect = container.getBoundingClientRect()
        
        for (let i = 0; i < warningNodes.length; i++) {
            const nodeRect = (warningNodes[i] as HTMLElement).getBoundingClientRect();
            const status = warningNodes[i].getAttribute('data-worst-status')
            
            // Add a 1px buffer to account for borders or subpixel rendering
            const isBelow = (nodeRect.bottom - containerRect.bottom) > 1 
            
            if (isBelow) { 
                if (status === 'down') {
                    worstStatus = 'down'
                    break
                } else if (status === 'degraded') {
                    worstStatus = 'degraded'
                }
            }
        }
    }
    
    worstWarningBelow.value = worstStatus
}

const scrollToBottom = () => {
    if (scrollArea.value) scrollArea.value.scrollTo({ top: scrollArea.value.scrollHeight, behavior: 'smooth' })
}

watch(() => selectedSensor.value?.sensorId, (newSensorId) => {
    const node = selectedNode.value
    if (newSensorId && node) {
        nextTick(() => {
            const el = document.getElementById(`row-${node.id}-${newSensorId}`)
            if (el && scrollArea.value) {
                const container = scrollArea.value
                const scrollTop = el.offsetTop - container.clientHeight / 2 + el.clientHeight / 2
                container.scrollTo({ top: Math.max(0, scrollTop), behavior: 'smooth' })
            }
        })
    }
})

watch(() => selectedNode.value?.id, (newNodeId) => {
    if (newNodeId) {
        nextTick(() => {
            const el = document.getElementById(`group-${newNodeId}`)
            if (el && scrollArea.value) {
                const container = scrollArea.value
                const scrollTop = el.offsetTop - 10 // Provide a slight padding above the group header
                container.scrollTo({ top: Math.max(0, scrollTop), behavior: 'smooth' })
            }
        })
    }
})

watch(hydratedUptimeGroups, () => nextTick(checkScroll), { deep: true })

onMounted(() => { 
    nextTick(checkScroll)
})

const legendItems = [
    { label: 'Up', colorClass: 'bg-success-main' },
    { label: 'Degraded', colorClass: 'bg-high' },
    { label: 'Down', colorClass: 'bg-critical' },
    { label: 'Pending / N/A', colorClass: 'bg-bg-inset' }
]
</script>

<template>
    <BaseWidget>
        <template #header>
            <div class="flex justify-between items-start h-14 relative z-10 shrink-0 w-full">
                <div>
                    <h3 class="text-base font-medium text-text-h">Fleet Uptime</h3>
                    <div class="flex items-center gap-2 mt-1 leading-none">
                        <span class="text-sm text-text-m">Fleet Overall:</span>
                        <span class="text-sm transition-colors duration-normal" 
                              :class="(uptimeData?.summary?.overallUptime || 0) >= 95 ? 'text-success-main' : ((uptimeData?.summary?.overallUptime || 0) >= 85 ? 'text-high' : 'text-critical')">
                            {{ (uptimeData?.summary?.overallUptime || 0).toFixed(2) }}%
                        </span>
                    </div>
                </div>
                
                <BaseTimeFilter 
                    :model-value="activeTimeframe"
                    @update:model-value="fleetStore.setActiveTimeframe($event)"
                />
            </div>
        </template>

        <div class="flex-1 relative mt-2 min-h-0 w-full">
            <div ref="scrollArea" @scroll.passive="checkScroll" class="absolute top-0 left-0 right-0 bottom-0 overflow-y-auto custom-scroll pr-3 pb-10">
                
                <div v-show="!uptimeData?.groups || uptimeData.groups.length === 0" class="absolute inset-0 flex items-center justify-center text-center text-base text-text-m z-20">No fleet data available.</div>
                
                <div v-for="group in hydratedUptimeGroups" :key="group.nodeId" :id="'group-' + group.nodeId"
                    class="transition-all duration-normal rounded-lg p-0.5 mb-0.5 border"
                    :class="{
                        'border-select-group-border bg-select-group-bg': selectedNode?.id === group.nodeId && !selectedSensor,
                        'border-transparent': selectedNode?.id !== group.nodeId || selectedSensor,
                        'opacity-50': (selectedNode || selectedSensor) && selectedNode?.id !== group.nodeId
                    }">
                     
                    <div class="px-1.5 mb-1 flex items-center gap-2 group/header"
                        :class="group.nodeId !== 'unassigned' ? 'cursor-pointer' : ''"
                        @click="group.nodeId !== 'unassigned' ? fleetStore.selectTarget(group.nodeId) : null">
                                            
                        <span class="text-sm font-semibold text-text-l transition-colors duration-[var(--duration-fast)]"
                              :class="group.nodeId !== 'unassigned' ? 'group-hover/header:text-text-h' : ''">
                            {{ group.nodeAlias || group.nodeId }}
                        </span>
                        
                        <div class="h-px flex-1 bg-border-default transition-colors duration-[var(--duration-fast)] group-hover/header:bg-text-m"></div>
                    </div>
                     
                    <div v-for="sensor in group.sensors" :key="sensor.nodeId + '-' + sensor.sensorId" :id="'row-' + sensor.nodeId + '-' + sensor.sensorId" 
                        class="flex items-center w-full transition-all duration-normal px-1.5 h-7 rounded-md border"
                        :class="{
                            'opacity-50': selectedSensor && (selectedSensor?.sensorId !== sensor.sensorId || selectedNode?.id !== sensor.nodeId),
                            'bg-select-row-bg border-select-row-border shadow-sm': selectedSensor?.sensorId === sensor.sensorId && selectedNode?.id === sensor.nodeId,
                            'border-transparent': !selectedSensor || (selectedSensor?.sensorId !== sensor.sensorId || selectedNode?.id !== sensor.nodeId),
                            'has-warnings': sensor.worstStatus !== null
                        }"
                        :data-worst-status="sensor.worstStatus"
                        >
                         
                        <div class="w-[180px] flex items-center gap-2 shrink-0 pr-2">
                            
                            <BaseMeatballMenu :id="`${sensor.nodeId}|${sensor.sensorId}`">
                                <button @click="handleSilence(sensor.nodeId, sensor.sensorId, sensor.isSilenced)" 
                                        class="w-full text-left px-3 py-2 text-sm text-text-m font-medium flex items-center gap-2 hover:bg-secondary-hover transition-colors group"
                                        :class="sensor.isSilenced ? 'text-archive-text' : 'text-text-l hover:text-text-h'">
                                    <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <path v-if="!sensor.isSilenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                                        <path v-if="sensor.isSilenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                                    </svg>
                                    {{ sensor.isSilenced ? 'Unsilence' : 'Silence Alert' }}
                                </button>
                                
                                <button @click="handleForget(sensor.nodeId, sensor.sensorId)" 
                                        class="w-full text-left px-3 py-2 text-sm font-medium text-danger-text flex items-center gap-2 hover:bg-danger-bg transition-colors group border-t border-border-default mt-1 pt-2">
                                    <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" />
                                        <path class="origin-bottom-right transition-transform duration-normal group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                                    </svg>
                                    Forget Sensor
                                </button>
                            </BaseMeatballMenu>

                            <BaseStatusDot :status="sensor.liveStatus" />
                            
                            <button @click="fleetStore.selectTarget(sensor.nodeId, sensor.sensorId)"
                                class="font-mono text-left transition-colors cursor-pointer rounded flex items-center gap-1.5 max-w-[calc(100%-28px)] text-sm"
                                :class="selectedSensor?.sensorId === sensor.sensorId && selectedNode?.id === sensor.nodeId ? 'text-text-h font-bold' : 'text-text-m font-medium hover:text-text-h'"
                                :title="`Node: ${group.nodeAlias || group.nodeId}`">
                                <span class="truncate">{{ formatSensorId(sensor.sensorId) }}</span>
                                
                                <svg v-show="sensor.isSilenced" class="w-3 h-3 shrink-0 text-medium" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                    <path d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                                </svg>
                            </button>
                        </div>

                        <div class="flex-1 flex justify-end items-center gap-[2px] overflow-hidden flex-nowrap pl-2">
                            <div v-for="(block, i) in sensor.blocks" :key="i"
                                 class="flex-1 max-w-[8px] min-h-[20px] min-w-[2px] h-5 rounded-[2px] transition-opacity duration-[var(--duration-fast)] hover:opacity-70 cursor-default"
                                 :class="{
                                     'bg-success-main': block.status === 'up', 
                                     'bg-critical': block.status === 'down', 
                                     'bg-high': block.status === 'degraded', 
                                     'bg-bg-inset': block.status === 'nodata' || block.status === 'pending'
                                 }"
                                 :title="`${block.timeLabel} - ${block.label}`">
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="absolute bottom-0 left-0 right-0 flex items-end justify-center pointer-events-none pb-2 bg-gradient-to-t from-bg-surface to-transparent h-12">
                <transition enter-active-class="transition-all duration-normal ease-out" enter-from-class="opacity-0 translate-y-4 scale-95" enter-to-class="opacity-100 translate-y-0 scale-100" leave-active-class="transition-all duration-[var(--duration-fast)] ease-in" leave-from-class="opacity-100 translate-y-0 scale-100" leave-to-class="opacity-0 translate-y-4 scale-95">
                    <div v-show="canScrollDown && worstWarningBelow !== null" 
                        @click="scrollToBottom"
                        class="pointer-events-auto relative cursor-pointer group/notify active:scale-95 transition-transform duration-[var(--duration-fast)] drop-shadow-md">
                        <div class="animate-bounce-subtle relative bg-bg-surface border border-border-default py-1.5 px-2 rounded-full flex justify-center items-center transition-colors duration-normal group-hover/notify:bg-bg-inset z-10">
                            
                            <div class="w-1.5 z-1 h-2.5 rounded-[1px]" :class="[(worstWarningBelow === 'down' ? 'bg-critical' : 'bg-high'), { 'animate-pulse': canScrollDown }]"></div>
                            
                            <div class="absolute z-0 -bottom-[3px] left-1/2 transform -translate-x-1/2 w-2 h-2 bg-bg-surface border-r border-b border-border-default rotate-45 rounded-[1px] transition-colors duration-normal group-hover/notify:bg-bg-inset"></div>
                        </div>
                    </div>
                </transition>
            </div>
        </div>
        
        <template #footer>
            <div class="hidden sm:block"><BaseLegend :items="legendItems" /></div>
        </template>
    </BaseWidget>
</template>