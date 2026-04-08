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

// Logic to show/hide the "↓ Offline Events Below" floating button
const checkScroll = () => {
    if (!scrollArea.value) return
    canScrollDown.value = scrollArea.value.scrollTop + scrollArea.value.clientHeight < scrollArea.value.scrollHeight - 5
}

// Watch for external selections (e.g., from the top filters) and scroll this view to match
watch(() => props.selectedSensor, (newVal) => {
    if (newVal) {
        nextTick(() => {
            const el = document.getElementById(`row-${newVal}`)
            if (el) {
                el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
            }
        })
    }
})

const scrollToBottom = () => {
    if (scrollArea.value) {
        scrollArea.value.scrollTo({ top: scrollArea.value.scrollHeight, behavior: 'smooth' })
    }
}

// Re-check scrollbar whenever the data changes
watch(() => props.uptimeData, () => nextTick(checkScroll), { deep: true })
onMounted(() => nextTick(checkScroll))

// Helper to check if a sensor in the heatmap is currently silenced
const isSilenced = (sensorId) => {
    const sensor = props.fleet.find(f => f.sensor_id === sensorId)
    return sensor ? sensor.is_silenced : false
}
</script>

<template>
    <div class="bg-slate-50 dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 rounded-lg p-5 flex flex-col backdrop-blur-sm h-full w-full relative">
        
        <div class="flex justify-between items-start mb-1 shrink-0">
            <div>
                <h3 class="text-sm font-semibold text-slate-800 dark:text-zinc-200">Fleet Uptime</h3>
                <p class="text-xs text-slate-500 dark:text-zinc-400 mt-1">
                    Fleet Overall Uptime: 
                    <span class="font-semibold transition-colors" 
                          :class="parseFloat(overallUptime) >= 95 ? 'text-emerald-600 dark:text-emerald-400' : (parseFloat(overallUptime) >= 85 ? 'text-amber-600 dark:text-amber-400' : 'text-rose-600 dark:text-rose-400')">
                        {{ overallUptime }}
                    </span>
                </p>
            </div>
            
            <div class="flex bg-slate-200 dark:bg-zinc-800 p-0.5 rounded-md text-[11px] font-medium text-slate-600 dark:text-zinc-400">
                <button v-for="time in ['1H', '24H', '7D', '30D']" :key="time"
                        @click="$emit('update:timeframe', time)"
                        class="px-2.5 py-1 rounded transition-colors"
                        :class="activeTimeframe === time ? 'bg-slate-50 dark:bg-zinc-700 text-slate-900 dark:text-zinc-100 shadow-sm' : 'hover:text-slate-800 dark:hover:text-zinc-200'">
                    {{ time }}
                </button>
            </div>
        </div>

        <div class="relative mt-4 flex-1">
            <div ref="scrollArea" @scroll.passive="checkScroll" class="absolute inset-0 overflow-y-auto custom-scroll pr-2 space-y-3">
                
                <div v-show="uptimeData.length === 0" class="text-xs text-slate-400 dark:text-zinc-500 py-4 text-center">
                    No fleet data available.
                </div>
                
                <div v-for="sensor in uptimeData" :key="sensor.id" class="flex items-center w-full">
                    
                    <div class="w-50 flex items-center gap-1.5 shrink-0">
                        <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="sensor.isOnline ? 'bg-emerald-500' : 'bg-rose-500'"></span>
                        <button @click="$emit('select-sensor', sensor.id)"
                                class="text-[11px] mono text-left truncate transition-colors cursor-pointer px-2 py-0.5 rounded-md flex items-center gap-1 flex-1 min-w-0"
                                :class="selectedSensor === sensor.id ? 'bg-slate-200 text-slate-900 dark:bg-zinc-700 dark:text-white font-bold' : 'text-slate-600 dark:text-zinc-400 font-medium hover:text-slate-900 dark:hover:text-zinc-200'"
                                :title="`Filter by ${sensor.name}${isSilenced(sensor.id) ? ' (Silenced)' : ''}`">
                            <span class="truncate flex-1">{{ sensor.name }}</span>
                            
                            <svg v-show="isSilenced(sensor.id)" class="w-3 h-3 shrink-0 text-slate-400 dark:text-zinc-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"></path></svg>
                        </button>
                    </div>
                                                                                                    
                    <div class="flex-1 flex justify-end gap-[2px] overflow-hidden flex-nowrap pl-2">
                        <div v-for="(block, i) in sensor.blocks" :key="i"
                             class="flex-1 max-w-[8px] min-w-[2px] h-5 rounded-[1px] transition-opacity hover:opacity-70 cursor-pointer"
                             :class="{
                                 'bg-emerald-500': block.status === 'up',
                                 'bg-rose-500': block.status === 'down',
                                 'bg-amber-500': block.status === 'degraded',
                                 'bg-slate-200 dark:bg-zinc-700': block.status === 'nodata'
                             }"
                             :title="`${block.timeLabel} - ${block.label}`">
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <transition enter-active-class="transition ease-out duration-200" enter-from-class="opacity-0 translate-y-2" enter-to-class="opacity-100 translate-y-0" leave-active-class="transition ease-in duration-150" leave-from-class="opacity-100 translate-y-0" leave-to-class="opacity-0 translate-y-2">
            <div v-show="canScrollDown && uptimeData.some(s => s.blocks.some(b => b.status === 'down'))" 
                 @click="scrollToBottom"
                 class="absolute bottom-1 left-1/2 -translate-x-1/2 bg-white/80 dark:bg-zinc-800/80 border border-slate-200 dark:border-zinc-700 text-slate-500 dark:text-zinc-400 text-[10px] font-medium px-2.5 py-1 rounded-md shadow-sm backdrop-blur-md flex items-center gap-2 cursor-pointer transition-colors hover:bg-slate-50 dark:hover:bg-zinc-700 z-10">
                <div class="w-1.5 sm:w-2 h-4 rounded-[1px] bg-rose-500 animate-pulse"></div>
                <span>↓ Offline Events Below</span>
            </div>
        </transition>
    </div>
</template>