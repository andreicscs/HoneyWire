<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick, computed } from 'vue'

const props = defineProps({
    uptimeData: { type: Array, required: true },
    overallUptime: { type: String, required: true },
    activeTimeframe: { type: String, required: true },
    fleet: { type: Array, required: true },
    selectedNode: { type: String, default: null },
    selectedSensor: { type: String, default: null }
})

const emit = defineEmits(['update:timeframe', 'select-sensor', 'select-node', 'toggle-silence', 'forget-sensor'])

const scrollArea = ref(null)
const canScrollDown = ref(false)

const activeMenu = ref(null)
const menuPos = ref({ top: '0px', left: '0px' })

const activeSensorData = computed(() => props.fleet.find(s => s.sensor_id === activeMenu.value))

const toggleMenu = (e, id) => {
    if (activeMenu.value === id) {
        activeMenu.value = null
        return
    }
    const rect = e.currentTarget.getBoundingClientRect()
    menuPos.value = { top: rect.bottom + 6 + 'px', left: rect.left + 'px' }
    activeMenu.value = id
}

const handleSilence = (sensorId) => {
    emit('toggle-silence', sensorId)
    activeMenu.value = null
}

const handleForget = (sensorId) => {
    emit('forget-sensor', sensorId)
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

const isSilenced = (sensorId) => {
    const sensor = props.fleet.find(f => f.sensor_id === sensorId)
    return sensor ? sensor.is_silenced : false
}

const getNodeForSensor = (sensorId) => {
    return props.fleet.find(f => f.sensor_id === sensorId)?.node_id || null
}

const groupedUptime = computed(() => {
    const groupsMap = new Map();
    
    props.uptimeData.forEach(sensor => {
        const nId = getNodeForSensor(sensor.id) || 'unassigned';
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

watch(() => props.selectedSensor, (newVal) => {
    if (newVal) {
        nextTick(() => {
            const el = document.getElementById(`row-${newVal}`)
            if (el) el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        })
    }
})

watch(() => props.selectedNode, (newVal) => {
    if (newVal) {
        nextTick(() => {
            const el = document.getElementById(`group-${newVal}`)
            if (el) el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        })
    }
})

watch(() => props.uptimeData, () => nextTick(checkScroll), { deep: true })

onMounted(() => { 
    nextTick(checkScroll)
    window.addEventListener('click', closeMenu)
    window.addEventListener('scroll', closeOnScroll, true) 
})

onUnmounted(() => { 
    window.removeEventListener('click', closeMenu)
    window.removeEventListener('scroll', closeOnScroll, true) 
})
</script>

<template>
    <div class="bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 rounded-lg p-4 sm:p-5 flex flex-col shadow-sm h-full w-full overflow-hidden relative group">
        
        <div class="flex justify-between items-start h-14 relative z-10 shrink-0 w-full">
            <div>
                <h3 class="text-sm font-semibold text-slate-800 dark:text-zinc-200">Fleet Uptime</h3>
                <div class="flex items-center gap-4 mt-1">
                    <p class="text-xs text-slate-500 dark:text-zinc-400">
                        Fleet Overall Uptime: 
                        <span class="font-semibold transition-colors" 
                              :class="parseFloat(overallUptime) >= 95 ? 'text-emerald-600 dark:text-emerald-400' : (parseFloat(overallUptime) >= 85 ? 'text-amber-600 dark:text-amber-400' : 'text-rose-600 dark:text-rose-400')">
                            {{ overallUptime }}
                        </span>
                    </p>
                </div>
            </div>
            
            <div class="flex bg-slate-50 border border-slate-100 dark:border-transparent dark:bg-zinc-800 p-0.5 rounded-md text-[11px] font-medium text-slate-500 dark:text-zinc-400">
                <button v-for="time in ['1H', '24H', '7D', '30D']" :key="time"
                        @click="$emit('update:timeframe', time)"
                        class="px-2.5 py-1 rounded transition-colors"
                        :class="activeTimeframe === time ? 'bg-white dark:bg-zinc-700 text-slate-800 dark:text-zinc-100 shadow-sm border border-slate-200 dark:border-transparent' : 'hover:text-slate-700 dark:hover:text-zinc-200'">
                    {{ time }}
                </button>
            </div>
        </div>

        <div class="flex-1 relative mt-2 min-h-0 w-full">
            <div ref="scrollArea" @scroll.passive="checkScroll" class="absolute top-0 left-0 right-0 bottom-0 overflow-y-auto custom-scroll pr-3 pb-10">
                <div v-show="uptimeData.length === 0" class="text-xs text-slate-400 dark:text-zinc-500 py-4 text-center">No fleet data available.</div>
                
                <!-- Tighter spacing: mb-1.5, p-1 -->
                <div v-for="group in groupedUptime" :key="group.nodeId" :id="'group-' + group.nodeId"
                     class="transition-all duration-300 rounded-lg p-1 mb-1.5 border"
                     :class="{
                         'border-slate-300 dark:border-zinc-600 bg-slate-50/50 dark:bg-white/5': selectedNode && selectedNode === group.nodeId,
                         'border-transparent': !selectedNode || selectedNode !== group.nodeId,
                         'opacity-30 grayscale-[40%]': (selectedNode && selectedNode !== group.nodeId) || (selectedSensor && (getNodeForSensor(selectedSensor) || 'unassigned') !== group.nodeId)
                     }">
                     
                    <!-- Tighter Header, fully clickable -->
                    <div class="px-1 mb-1 flex items-center gap-2 group/header"
                         :class="group.nodeId !== 'unassigned' ? 'cursor-pointer' : ''"
                         @click="group.nodeId !== 'unassigned' ? $emit('select-node', group.nodeId) : null">
                        
                        <span class="text-[8.5px] uppercase tracking-wider font-bold transition-colors"
                              :class="group.nodeId !== 'unassigned' ? 'text-slate-400 dark:text-zinc-500 group-hover/header:text-slate-700 dark:group-hover/header:text-zinc-300' : 'text-slate-400 dark:text-zinc-500'">
                            {{ group.nodeId !== 'unassigned' ? group.nodeId : 'Unassigned Sensors' }}
                        </span>
                        
                        <div class="h-px flex-1 transition-colors"
                             :class="group.nodeId !== 'unassigned' ? 'bg-slate-200 dark:bg-zinc-800 group-hover/header:bg-slate-300 dark:group-hover/header:bg-zinc-600' : 'bg-slate-200 dark:bg-zinc-800'"></div>
                    </div>
                     
                    <!-- Tighter sensor rows: py-0.5 mt-px -->
                    <div v-for="sensor in group.sensors" :key="sensor.id" :id="'row-' + sensor.id" 
                         class="flex items-center w-full transition-all duration-300 px-2 py-0.5 mt-px rounded-md border"
                         :class="{
                             'opacity-30 grayscale-[40%]': selectedSensor && selectedSensor !== sensor.id,
                             'bg-slate-100 dark:bg-zinc-800 border-slate-300 dark:border-zinc-500 shadow-sm': selectedSensor === sensor.id,
                             'border-transparent': selectedSensor !== sensor.id
                         }">
                         
                        <div class="w-[180px] flex items-center gap-1.5 shrink-0 pr-2">
                            
                            <div @click.stop="toggleMenu($event, sensor.id)" 
                                 class="meatball-toggle w-5 h-5 rounded flex items-center justify-center transition-all cursor-pointer shrink-0"
                                 :class="[
                                     activeMenu === sensor.id ? 'text-slate-700 dark:text-white bg-slate-200 dark:bg-zinc-700' :
                                     selectedSensor === sensor.id ? 'text-slate-500 dark:text-zinc-300 hover:text-slate-700 dark:hover:text-white hover:bg-slate-200 dark:hover:bg-zinc-600' :
                                     'text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-200 dark:hover:bg-zinc-700'
                                 ]">
                                <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/></svg>
                            </div>

                            <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="sensor.isOnline ? 'bg-emerald-500' : 'bg-rose-500'"></span>
                            
                            <button @click="$emit('select-sensor', sensor.id, getNodeForSensor(sensor.id))"
                                    class="text-[11px] mono text-left transition-colors cursor-pointer px-1 py-0.5 rounded-md flex items-center gap-1.5 max-w-[calc(100%-28px)]"
                                    :class="selectedSensor === sensor.id ? 'text-slate-900 dark:text-white font-bold' : 'text-slate-600 dark:text-zinc-400 font-medium hover:text-slate-900 dark:hover:text-zinc-200'"
                                    :title="`Node: ${getNodeForSensor(sensor.id) || 'Unassigned'}`">
                                <span class="truncate">{{ sensor.name }}</span>
                                <svg v-show="isSilenced(sensor.id)" class="w-3 h-3 shrink-0" :class="selectedSensor === sensor.id ? 'text-amber-500 dark:text-amber-400' : 'text-amber-500'" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                    <path d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                                </svg>
                            </button>
                        </div>

                        <div class="flex-1 flex justify-end gap-[2px] overflow-hidden flex-nowrap pl-2">
                            <div v-for="(block, i) in sensor.blocks" :key="i"
                                 class="flex-1 max-w-[8px] min-h-5 min-w-[2px] h-4 rounded-[2px] transition-opacity hover:opacity-70 cursor-pointer"
                                 :class="{'bg-emerald-500': block.status === 'up', 'bg-rose-500': block.status === 'down', 'bg-amber-500': block.status === 'degraded', 'bg-slate-200 dark:bg-zinc-800': block.status === 'nodata'}"
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
                        class="pointer-events-auto relative cursor-pointer group active:scale-95 transition-transform duration-150 drop-shadow-[0_4px_12px_rgba(0,0,0,0.09)] dark:drop-shadow-[0_4px_12px_rgba(0,0,0,0.3)]">
                        <div class="animate-bounce-subtle relative bg-white dark:bg-zinc-800 border border-slate-300 dark:border-zinc-700 py-1.5 px-2 rounded-full flex justify-center items-center transition-colors duration-200 group-hover:bg-slate-50 dark:group-hover:bg-zinc-700/90 z-10">
                            <div class="w-1.5 z-1 h-2.5 rounded-[1px]" :class="[(uptimeData.some(s => s.blocks.some(b => b.status === 'down')) ? 'bg-rose-500' : 'bg-amber-500'), { 'animate-pulse': canScrollDown }]"></div>
                            <div class="absolute z-0 -bottom-[3px] left-1/2 transform -translate-x-1/2 w-2.5 h-2.5 bg-white dark:bg-zinc-800 border-r border-b border-slate-300 dark:border-zinc-700 rotate-45 rounded-[1px] transition-colors duration-200 group-hover:bg-slate-50 dark:group-hover:bg-zinc-700/90"></div>
                        </div>
                    </div>
                </transition>
            </div>
        </div>
        
        <div class="hidden sm:flex mt-auto h-4 pt-5 items-center justify-center gap-3 sm:gap-4 text-[8px] font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wider shrink-0 border-t border-transparent">
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-emerald-500"></span>Up</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-amber-500"></span>Degraded</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-rose-500"></span>Down</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-slate-200 dark:bg-zinc-800"></span>N/A</div>
        </div>

        <Teleport to="body">
            <transition enter-active-class="transition ease-out duration-100" enter-from-class="transform opacity-0 scale-95" enter-to-class="transform opacity-100 scale-100" leave-active-class="transition ease-in duration-75" leave-from-class="transform opacity-100 scale-100" leave-to-class="transform opacity-0 scale-95">
                <div v-if="activeMenu && activeSensorData" 
                     :style="{ top: menuPos.top, left: menuPos.left }"
                     class="global-sensor-dropdown fixed w-36 rounded-md shadow-xl bg-white dark:bg-zinc-800 border border-slate-200 dark:border-zinc-700 z-[100] py-1 overflow-hidden">
                    <button @click.stop="handleSilence(activeSensorData.sensor_id)" 
                            class="w-full text-left px-3 py-2 text-xs font-semibold flex items-center gap-2 hover:bg-slate-100 dark:hover:bg-zinc-700 transition-colors group"
                            :class="activeSensorData.is_silenced ? 'text-amber-600 dark:text-amber-500' : 'text-slate-600 dark:text-zinc-300'">
                        <svg class="w-3.5 h-3.5 transition-transform duration-200 group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path v-if="!activeSensorData.is_silenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                            <path v-if="activeSensorData.is_silenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                        </svg>
                        {{ activeSensorData.is_silenced ? 'Unsilence' : 'Silence Alert' }}
                    </button>
                    <button @click="handleForget(activeSensorData.sensor_id)" 
                            class="w-full text-left px-3 py-2 text-xs font-semibold text-rose-600 dark:text-rose-400 flex items-center gap-2 hover:bg-rose-50 dark:hover:bg-rose-900/20 transition-colors group border-t border-slate-100 dark:border-zinc-700/50 mt-1 pt-2">
                        <svg class="w-3.5 h-3.5 transition-transform duration-200 group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" />
                            <path class="origin-bottom-right transition-transform duration-300 group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                        </svg>
                        Forget Sensor
                    </button>
                </div>
            </transition>
        </Teleport>

    </div>
</template>