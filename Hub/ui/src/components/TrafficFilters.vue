<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'

const props = defineProps({
    fleet: { type: Array, required: true },
    selectedSensor: { type: String, default: null }
})

defineEmits(['select-sensor'])

const scrollArea = ref(null)
const showOfflineWarning = ref(false)

// Advanced Check: Only show the red dot if an OFFLINE sensor is physically hidden off-screen
const checkScroll = () => {
    if (!scrollArea.value) return
    const container = scrollArea.value
    
    // Find all buttons that currently have the offline class
    const offlineBtns = container.querySelectorAll('.is-offline')
    
    let isHidden = false
    if (offlineBtns.length > 0) {
        const lastOffline = offlineBtns[offlineBtns.length - 1]
        // Calculate if the right edge of the last offline pill is past the visible scroll area
        if (lastOffline.offsetLeft + lastOffline.offsetWidth > container.scrollLeft + container.clientWidth + 10) {
            isHidden = true
        }
    }
    showOfflineWarning.value = isHidden
}

// Watch for external selections (e.g., from the Heatmap) and scroll this view to match
watch(() => props.selectedSensor, (newVal) => {
    nextTick(() => {
        const targetId = newVal ? `pill-${newVal}` : 'pill-all'
        const el = document.getElementById(targetId)
        if (el) {
            el.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
        }
    })
})

// Re-check scrollbar whenever the fleet changes
watch(() => props.fleet, () => nextTick(checkScroll), { deep: true })
onMounted(() => nextTick(checkScroll))
</script>

<template>
    <div class="flex justify-between items-center gap-4">
        <div class="flex items-center w-full gap-2">
            <div class="flex-1 relative overflow-hidden">
                
                <div ref="scrollArea" @scroll.passive="checkScroll" class="flex overflow-x-auto whitespace-nowrap gap-2 items-center custom-scroll pb-3 pr-2 relative">
                    
                    <div class="sticky left-0 z-20 pr-2 bg-slate-100 dark:bg-[#0a0a0c] flex items-center border-r border-slate-200 dark:border-zinc-800 transition-all duration-200">
                        <button id="pill-all" @click="$emit('select-sensor', null)" 
                                class="shrink-0 px-3 py-1.5 rounded-md border text-sm font-medium transition-colors"
                                :class="!selectedSensor ? 'bg-slate-800 text-slate-50 dark:bg-zinc-200 dark:text-black border-transparent' : 'bg-slate-50 dark:bg-zinc-900 border-slate-300 dark:border-zinc-700 text-slate-600 dark:text-zinc-300 hover:bg-slate-100 dark:hover:bg-zinc-800'">
                            All Traffic
                        </button>
                    </div>

                    <button v-for="s in fleet" :key="s.sensor_id" 
                            :id="'pill-' + s.sensor_id"
                            @click="$emit('select-sensor', s.sensor_id)" 
                            class="shrink-0 flex items-center gap-2 px-3 py-1.5 rounded-md border text-sm font-medium transition-colors group"
                            :class="[
                                selectedSensor === s.sensor_id ? 'bg-slate-800 text-slate-50 dark:bg-zinc-200 dark:text-black border-transparent' : 'bg-slate-50 dark:bg-zinc-900 border-slate-300 dark:border-zinc-700 text-slate-600 dark:text-zinc-300 hover:bg-slate-100 dark:hover:bg-zinc-800',
                                s.status !== 'online' ? 'is-offline' : ''
                            ]"
                            :title="s.status === 'online' ? 'Active' : 'Last seen: ' + s.last_seen">
                        
                        <span class="w-1.5 h-1.5 rounded-full" :class="s.status === 'online' ? 'bg-emerald-500' : 'bg-rose-500'"></span>
                        <span class="mono text-xs pointer-events-none">{{ s.sensor_id }}</span>
                    </button>
                </div>
                
                <div class="absolute right-0 top-0 bottom-3 w-16 bg-gradient-to-l from-slate-100 dark:from-[#0a0a0c] to-transparent pointer-events-none"></div>
            </div>
            
            <div class="w-6 h-8 shrink-0 flex items-center justify-center">
                <transition enter-active-class="transition-opacity duration-300" leave-active-class="transition-opacity duration-300" enter-from-class="opacity-0" leave-to-class="opacity-0">
                    <span v-show="showOfflineWarning" class="w-1.5 h-1.5 rounded-full bg-rose-500 animate-pulse"></span>
                </transition>
            </div>
        </div>
    </div>
</template>