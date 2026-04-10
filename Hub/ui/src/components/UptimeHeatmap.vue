<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'

const props = defineProps({
    uptimeData: { type: Array, required: true },
    overallUptime: { type: String, required: true },
    activeTimeframe: { type: String, required: true },
    fleet: { type: Array, required: true },
    selectedSensor: { type: String, default: null }
})

const emit = defineEmits(['update:timeframe', 'select-sensor'])

const scrollArea = ref(null)
const canScrollDown = ref(false)

const checkScroll = () => {
    if (!scrollArea.value) return
    const container = scrollArea.value
    const currentBottom = Math.ceil(container.scrollTop + container.clientHeight)
    canScrollDown.value = currentBottom < (container.scrollHeight - 15)
}

watch(() => props.selectedSensor, (newVal) => {
    if (newVal) {
        nextTick(() => {
            const el = document.getElementById(`row-${newVal}`)
            if (el) el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        })
    }
})

const scrollToBottom = () => {
    if (scrollArea.value) scrollArea.value.scrollTo({ top: scrollArea.value.scrollHeight, behavior: 'smooth' })
}

watch(() => props.uptimeData, () => nextTick(checkScroll), { deep: true })
onMounted(() => nextTick(checkScroll))

const isSilenced = (sensorId) => {
    const sensor = props.fleet.find(f => f.sensor_id === sensorId)
    return sensor ? sensor.is_silenced : false
}
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
            <div ref="scrollArea" @scroll.passive="checkScroll" class="absolute top-0 left-0 right-0 bottom-0 overflow-y-auto custom-scroll pr-3 space-y-2 pb-10">
                <div v-show="uptimeData.length === 0" class="text-xs text-slate-400 dark:text-zinc-500 py-4 text-center">No fleet data available.</div>
                
                <div v-for="sensor in uptimeData" :key="sensor.id" :id="'row-' + sensor.id" class="flex items-center w-full">
                    <div class="w-50 flex items-center gap-1.5 shrink-0 pr-2">
                        <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="sensor.isOnline ? 'bg-emerald-500' : 'bg-rose-500'"></span>
                        
                        <button @click="$emit('select-sensor', sensor.id)"
                                class="text-[11px] mono text-left transition-colors cursor-pointer px-2 py-0.5 rounded-md flex items-center gap-1.5 max-w-[calc(100%-12px)]"
                                :class="selectedSensor === sensor.id ? 'bg-slate-200 text-slate-900 dark:bg-zinc-700 dark:text-white font-bold' : 'text-slate-600 dark:text-zinc-400 font-medium hover:text-slate-900 dark:hover:text-zinc-200'"
                                :title="`Filter by ${sensor.name}${isSilenced(sensor.id) ? ' (Silenced)' : ''}`">
                            <span class="truncate">{{ sensor.name }}</span>
                            <svg v-show="isSilenced(sensor.id)" class="w-3 h-3 shrink-0 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
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
        
        <div class="hidden sm:flex mt-auto h-4 pt-5 items-center justify-end gap-3 sm:gap-4 text-[8px] font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wider shrink-0 border-t border-transparent">
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-emerald-500"></span>Up</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-amber-500"></span>Degraded</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-rose-500"></span>Down</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-slate-200 dark:bg-zinc-800"></span>N/A</div>
        </div>

    </div>
</template>