<script setup>
import { ref, computed, nextTick, watch, onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useFleetStore } from '../stores/fleet'
import BaseTimeFilter from './ui/BaseTimeFilter.vue'
import BaseWidget from './ui/BaseWidget.vue'
import BaseLegend from './ui/BaseLegend.vue'

const fleetStore = useFleetStore()
const { sensors: fleet, uptimeData, selectedNode, selectedSensor, activeTimeframe, overallUptime } = storeToRefs(fleetStore)

const scrollArea = ref(null)
const canScrollDown = ref(false)

const activeMenu = ref(null)
const menuPos = ref({ top: '0px', left: '0px' })

const activeSensorData = computed(() => {
    if (!activeMenu.value) return null
    const [nId, sId] = activeMenu.value.split('|')
    return fleet.value.find(s => s.node_id === nId && s.sensor_id === sId)
})

const toggleMenu = (e, nodeId, sensorId) => {
    const compositeId = `${nodeId}|${sensorId}`
    if (activeMenu.value === compositeId) {
        activeMenu.value = null
        return
    }
    const rect = e.currentTarget.getBoundingClientRect()
    menuPos.value = { top: rect.bottom + 6 + 'px', left: rect.left + 'px' }
    activeMenu.value = compositeId
}

const handleSilence = (nodeId, sensorId) => {
    fleetStore.toggleSilence(nodeId, sensorId)
    activeMenu.value = null
}

const handleForget = (nodeId, sensorId) => {
    fleetStore.forgetSensor(nodeId, sensorId)
    activeMenu.value = null
}

const closeMenu = (e) => { if (!e.target.closest('.global-sensor-dropdown') && !e.target.closest('.meatball-toggle')) activeMenu.value = null }
const closeOnScroll = () => { if (activeMenu.value) activeMenu.value = null }

const checkScroll = () => {
    if (!scrollArea.value) return
    const container = scrollArea.value
    const currentBottom = Math.ceil(container.scrollTop + container.clientHeight)
    canScrollDown.value = currentBottom < (container.scrollHeight - 15)
}

const scrollToBottom = () => {
    if (scrollArea.value) scrollArea.value.scrollTo({ top: scrollArea.value.scrollHeight, behavior: 'smooth' })
}

const isSilenced = (nodeId, sensorId) => {
    const sensor = fleet.value.find(f => f.node_id === nodeId && f.sensor_id === sensorId)
    return sensor ? sensor.is_silenced : false
}

const groupedUptime = computed(() => {
    const groupsMap = new Map();
    
    uptimeData.value.forEach(sensor => {
        const nId = sensor.node_id || 'unassigned';
        if (!groupsMap.has(nId)) {
            groupsMap.set(nId, { nodeId: nId, sensors: [] });
        }
        groupsMap.get(nId).sensors.push(sensor);
    });
    
    const groups = Array.from(groupsMap.values());
    
    groups.sort((a, b) => {
        if (a.nodeId === 'unassigned') return 1;
        if (b.nodeId === 'unassigned') return -1;
        return a.nodeId.localeCompare(b.nodeId);
    });
    
    return groups;
});

watch(selectedSensor, (newVal) => {
    if (newVal && selectedNode.value) {
        nextTick(() => {
            const el = document.getElementById(`row-${selectedNode.value}-${newVal}`)
            if (el) el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        })
    }
})

watch(selectedNode, (newVal) => {
    if (newVal) {
        nextTick(() => {
            const el = document.getElementById(`group-${newVal}`)
            if (el) el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        })
    }
})

watch(uptimeData, () => nextTick(checkScroll), { deep: true })

onMounted(() => { 
    nextTick(checkScroll)
    window.addEventListener('click', closeMenu)
    window.addEventListener('scroll', closeOnScroll, true) 
})

onUnmounted(() => { 
    window.removeEventListener('click', closeMenu)
    window.removeEventListener('scroll', closeOnScroll, true) 
})

const legendItems = [
    { label: 'Up', colorClass: 'bg-success-main' },
    { label: 'Degraded', colorClass: 'bg-high' },
    { label: 'Down', colorClass: 'bg-critical' },
    { label: 'N/A', colorClass: 'bg-bg-inset' }
]
</script>

<template>
    <BaseWidget>
        <template #header>
            <div class="flex justify-between items-start h-14 relative z-10 shrink-0 w-full">
                <div>
                    <h3 class="text-base  text-text-h">Fleet Uptime</h3>
                    <div class="flex items-center gap-4 mt-1">
                        <p class="text-sm text-text-m">
                            Fleet Overall Uptime: 
                            <span class=" transition-colors" 
                                  :class="parseFloat(overallUptime) >= 95 ? 'text-success-main' : (parseFloat(overallUptime) >= 85 ? 'text-high' : 'text-critical')">
                                {{ overallUptime }}
                            </span>
                        </p>
                    </div>
                </div>
                
                <BaseTimeFilter v-model="fleetStore.activeTimeframe" />
            </div>
        </template>

        <div class="flex-1 relative mt-2 min-h-0 w-full">
            <div ref="scrollArea" @scroll.passive="checkScroll" class="absolute top-0 left-0 right-0 bottom-0 overflow-y-auto custom-scroll pr-3 pb-10">
                <div v-show="uptimeData.length === 0" class="text-sm text-text-m py-4 text-center">No fleet data available.</div>
                
                <div v-for="group in groupedUptime" :key="group.nodeId" :id="'group-' + group.nodeId"
                    class="transition-all duration-300 rounded-lg p-1 mb-1.5 border"
                    :class="{
                        'border-select-group-border bg-select-group-bg': selectedNode === group.nodeId && !selectedSensor,
                        'border-transparent': selectedNode !== group.nodeId || selectedSensor,
                        'opacity-50 grayscale-[40%]': (selectedNode || selectedSensor) && selectedNode !== group.nodeId
                    }">
                     
                    <div class="px-1 mb-1 flex items-center gap-2 group/header"
                        :class="group.nodeId !== 'unassigned' ? 'cursor-pointer' : ''"
                        @click="group.nodeId !== 'unassigned' ? fleetStore.selectTarget(group.nodeId) : null">
                                            
                        <span class="text-sm tracking-wider transition-colors"
                              :class="group.nodeId !== 'unassigned' ? 'text-text-m group-hover/header:text-text-h' : 'text-text-m'">
                            {{ group.nodeId !== 'unassigned' ? group.nodeId : 'Unassigned Sensors' }}
                        </span>
                        
                        <div class="h-px flex-1 transition-colors"
                             :class="group.nodeId !== 'unassigned' ? 'bg-border-default group-hover/header:bg-text-muted/50' : 'bg-border-default'"></div>
                    </div>
                     
                    <div v-for="sensor in group.sensors" :key="sensor.node_id + '-' + sensor.id" :id="'row-' + sensor.node_id + '-' + sensor.id" 
                        class="flex items-center w-full transition-all duration-300 px-2 py-0.5 mt-px rounded-md border"
                        :class="{
                            'opacity-50 grayscale-[40%]': selectedSensor && (selectedSensor !== sensor.id || selectedNode !== sensor.node_id),
                            'bg-select-row-bg border-select-row-border shadow-sm': selectedSensor === sensor.id && selectedNode === sensor.node_id,
                            'border-transparent': !selectedSensor || (selectedSensor !== sensor.id || selectedNode !== sensor.node_id)
                        }">
                         
                        <div class="w-[180px] flex items-center gap-1.5 shrink-0 pr-2">
                            
                            <div @click.stop="toggleMenu($event, sensor.node_id, sensor.id)" 
                                 class="meatball-toggle w-5 h-5 rounded flex items-center justify-center transition-all cursor-pointer shrink-0"
                                 :class="[
                                     activeMenu === sensor.node_id + '|' + sensor.id ? 'text-text-h bg-button-selected' :
                                     selectedSensor === sensor.id ? 'text-text-m hover:text-text-h hover:bg-button-hover' :
                                     'text-text-m/70 hover:text-text-h hover:bg-button-hover'
                                 ]">
                                <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/></svg>
                            </div>

                            <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="sensor.isOnline ? 'bg-success-main' : 'bg-critical'"></span>
                            
                            <button @click="fleetStore.selectTarget(sensor.node_id, sensor.id)"
                                class="text-sm mono text-left transition-colors cursor-pointer px-1 py-0.5 rounded-md flex items-center gap-1.5 max-w-[calc(100%-28px)]"
                                :class="selectedSensor === sensor.id && selectedNode === sensor.node_id ? 'text-text-h ' : 'text-text-m  hover:text-text-h'"
                                :title="`Node: ${sensor.node_id || 'Unassigned'}`">
                                <span class="truncate">{{ sensor.name }}</span>
                                
                                <svg v-show="isSilenced(sensor.node_id, sensor.id)" class="w-3 h-3 shrink-0 text-medium" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                    <path d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                                </svg>
                            </button>
                        </div>

                        <div class="flex-1 flex justify-end gap-[2px] overflow-hidden flex-nowrap pl-2">
                            <div v-for="(block, i) in sensor.blocks" :key="i"
                                 class="flex-1 max-w-[8px] min-h-5 min-w-[2px] h-4 rounded-[2px] transition-opacity hover:opacity-70 cursor-pointer"
                                 :class="{
                                     'bg-success-main': block.status === 'up', 
                                     'bg-critical': block.status === 'down', 
                                     'bg-high': block.status === 'degraded', 
                                     'bg-bg-inset': block.status === 'nodata'
                                 }"
                                 :title="`${block.timeLabel} - ${block.label}`">
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="absolute bottom-1 left-0 right-0 flex items-end justify-center pointer-events-none">
                <transition enter-active-class="transition-all duration-300 ease-out" enter-from-class="opacity-0 translate-y-4 scale-95" enter-to-class="opacity-100 translate-y-0 scale-100" leave-active-class="transition-all duration-200 ease-in" leave-from-class="opacity-100 translate-y-0 scale-100" leave-to-class="opacity-0 translate-y-4 scale-95">
                    <div v-show="canScrollDown && uptimeData.some(s => s.blocks.some(b => b.status === 'down' || b.status === 'degraded'))" 
                        @click="scrollToBottom"
                        class="pointer-events-auto relative cursor-pointer group/notify active:scale-95 transition-transform duration-150 drop-shadow-md">
                        <div class="animate-bounce-subtle relative bg-bg-surface border border-border-default py-1.5 px-2 rounded-full flex justify-center items-center transition-colors duration-200 group-hover/notify:bg-button-hover z-10">
                            <div class="w-1.5 z-1 h-2.5 rounded-[1px]" :class="[(uptimeData.some(s => s.blocks.some(b => b.status === 'down')) ? 'bg-critical' : 'bg-high'), { 'animate-pulse': canScrollDown }]"></div>
                            <div class="absolute z-0 -bottom-[3px] left-1/2 transform -translate-x-1/2 w-2.5 h-2.5 bg-bg-surface border-r border-b border-border-default rotate-45 rounded-[1px] transition-colors duration-200 group-hover/notify:bg-button-hover"></div>
                        </div>
                    </div>
                </transition>
            </div>
        </div>
        
        <template #footer>
            <div class="hidden sm:block"><BaseLegend :items="legendItems" /></div>
        </template>

        <Teleport to="body">
            <transition enter-active-class="transition ease-out duration-100" enter-from-class="transform opacity-0 scale-95" enter-to-class="transform opacity-100 scale-100" leave-active-class="transition ease-in duration-75" leave-from-class="transform opacity-100 scale-100" leave-to-class="transform opacity-0 scale-95">
                <div v-if="activeMenu && activeSensorData" 
                     :style="{ top: menuPos.top, left: menuPos.left }"
                     class="global-sensor-dropdown fixed w-36 rounded-md shadow-xl bg-bg-surface border border-border-default z-[100] py-1 overflow-hidden">
                    
                    <button @click.stop="handleSilence(activeSensorData.node_id, activeSensorData.sensor_id)" 
                            class="w-full text-left px-3 py-2 text-sm flex items-center gap-2 hover:bg-button-hover transition-colors group"
                            :class="activeSensorData.is_silenced ? 'text-archive-text' : 'text-text-m hover:text-text-h'">
                        <svg class="w-3.5 h-3.5 transition-transform duration-200 group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path v-if="!activeSensorData.is_silenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                            <path v-if="activeSensorData.is_silenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                        </svg>
                        {{ activeSensorData.is_silenced ? 'Unsilence' : 'Silence Alert' }}
                    </button>
                    
                    <button @click="handleForget(activeSensorData.node_id, activeSensorData.sensor_id)" 
                            class="w-full text-left px-3 py-2 text-sm text-danger-text flex items-center gap-2 hover:bg-danger-bg transition-colors group border-t border-border-default mt-1 pt-2">
                        <svg class="w-3.5 h-3.5 transition-transform duration-200 group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" />
                            <path class="origin-bottom-right transition-transform duration-300 group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                        </svg>
                        Forget Sensor
                    </button>
                </div>
            </transition>
        </Teleport>

    </BaseWidget>
</template>