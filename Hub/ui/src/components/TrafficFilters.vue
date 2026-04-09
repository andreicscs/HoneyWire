<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick, computed } from 'vue'

const props = defineProps({
    fleet: { type: Array, required: true },
    selectedSensor: { type: String, default: null }
})

const emit = defineEmits(['select-sensor', 'forget-sensor', 'toggle-silence'])

const scrollArea = ref(null)
const showOfflineWarning = ref(false)
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

// REMOVED the setTimeout and isSilencing delay
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
    const offlineBtns = container.querySelectorAll('.is-offline')
    if (offlineBtns.length === 0) { showOfflineWarning.value = false; return }
    const lastOffline = offlineBtns[offlineBtns.length - 1]
    showOfflineWarning.value = lastOffline.getBoundingClientRect().right > (container.getBoundingClientRect().right + 5)
}

watch(() => props.selectedSensor, (newVal) => {
    nextTick(() => {
        const el = document.getElementById(newVal ? `pill-${newVal}` : 'pill-all')
        if (el) el.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
    })
})
watch(() => props.fleet, () => nextTick(checkScroll), { deep: true })
onMounted(() => { nextTick(checkScroll); window.addEventListener('click', closeMenu); window.addEventListener('scroll', closeOnScroll, true) })
onUnmounted(() => { window.removeEventListener('click', closeMenu); window.removeEventListener('scroll', closeOnScroll, true) })
</script>

<template>
    <div class="flex justify-between items-center gap-4">
        <div class="flex items-center w-full gap-2">
            <div class="flex-1 relative overflow-hidden">
                <div ref="scrollArea" @scroll.passive="checkScroll" class="flex overflow-x-auto whitespace-nowrap gap-2 items-center custom-scroll pb-3 pr-2 relative">
                    <div class="sticky left-0 z-20 pr-2 bg-slate-100 dark:bg-[#0a0a0c] flex items-center border-r border-slate-200 dark:border-zinc-800 transition-all duration-200">
                        <button id="pill-all" @click="$emit('select-sensor', null)" 
                                class="shrink-0 px-3 py-1.5 rounded-md border text-sm font-medium transition-colors"
                                :class="!selectedSensor ? 'bg-slate-800 text-slate-50 dark:bg-zinc-200 dark:text-black border-transparent' : 'bg-slate-50 dark:bg-zinc-900 border-slate-300 dark:border-zinc-700 text-slate-600 dark:text-zinc-300 hover:bg-slate-100 dark:hover:bg-zinc-800'">All Traffic</button>
                    </div>

                    <div v-for="s in fleet" :key="s.sensor_id" class="shrink-0 relative">
                        <button :id="'pill-' + s.sensor_id" @click="$emit('select-sensor', s.sensor_id)" 
                                class="flex items-center gap-2 pl-3 pr-2 py-1.5 rounded-md border text-sm font-medium transition-colors group"
                                :class="[
                                    selectedSensor === s.sensor_id ? 'bg-slate-800 text-slate-50 dark:bg-zinc-200 dark:text-black border-transparent' : 'bg-slate-50 dark:bg-zinc-900 border-slate-300 dark:border-zinc-700 text-slate-600 dark:text-zinc-300 hover:bg-slate-100 dark:hover:bg-zinc-800',
                                    s.status !== 'online' ? 'is-offline' : ''
                                ]"
                                :title="s.status === 'online' ? 'Active' : 'Last seen: ' + s.last_seen">
                            <span class="w-1.5 h-1.5 rounded-full" :class="s.status === 'online' ? 'bg-emerald-500' : 'bg-rose-500'"></span>
                            <span class="mono text-xs pointer-events-none">{{ s.sensor_id }}</span>
                            
                            <div @click.stop="toggleMenu($event, s.sensor_id)" 
                                 class="meatball-toggle w-5 h-5 ml-1 rounded flex items-center justify-center transition-all text-current opacity-40 group-hover:opacity-100 hover:bg-current/10 pointer-events-auto"
                                 :class="{'opacity-100 bg-current/10': activeMenu === s.sensor_id}">
                                <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24"><path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/></svg>
                            </div>
                        </button>
                    </div>
                </div>
                <div class="absolute right-0 top-0 bottom-3 w-16 bg-gradient-to-l from-slate-100 dark:from-[#0a0a0c] to-transparent pointer-events-none"></div>
            </div>
            
            <div class="w-6 h-8 shrink-0 flex items-center justify-center">
                <transition enter-active-class="transition-opacity duration-300 ease-out" leave-active-class="transition-opacity duration-300 ease-in" enter-from-class="opacity-0" leave-to-class="opacity-0">
                    <div v-show="showOfflineWarning" class="flex items-center justify-center">
                        <span class="w-1.5 h-1.5 rounded-full bg-rose-500" :class="{ 'animate-pulse': showOfflineWarning }"></span>
                    </div>
                </transition>
            </div>
        </div>

        <Teleport to="body">
            <transition enter-active-class="transition ease-out duration-100" enter-from-class="transform opacity-0 scale-95" enter-to-class="transform opacity-100 scale-100" leave-active-class="transition ease-in duration-75" leave-from-class="transform opacity-100 scale-100" leave-to-class="transform opacity-0 scale-95">
                <div v-if="activeMenu && activeSensorData" 
                     :style="{ top: menuPos.top, left: menuPos.left }"
                     class="global-sensor-dropdown fixed w-36 rounded-md shadow-xl bg-white dark:bg-zinc-800 border border-slate-200 dark:border-zinc-700 z-[100] py-1 overflow-hidden">
                    
                    <button @click.stop="handleSilence(activeSensorData.sensor_id)" 
                            class="w-full text-left px-3 py-2 text-xs font-semibold flex items-center gap-2 hover:bg-slate-100 dark:hover:bg-zinc-700 transition-colors group"
                            :class="activeSensorData.is_silenced ? 'text-amber-600 dark:text-amber-500' : 'text-slate-600 dark:text-zinc-300'">
                        <svg class="w-3.5 h-3.5 transition-transform duration-200 group-hover:rotate-12 group-active:-rotate-12 origin-top" 
                             fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path v-if="!activeSensorData.is_silenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                            <path v-if="activeSensorData.is_silenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                        </svg>
                        {{ activeSensorData.is_silenced ? 'Unsilence' : 'Silence Alert' }}
                    </button>

                    <button v-if="activeSensorData.status !== 'online'" 
                            @click="handleForget(activeSensorData.sensor_id)" 
                            class="w-full text-left px-3 py-2 text-xs font-semibold text-rose-600 dark:text-rose-400 flex items-center gap-2 hover:bg-rose-50 dark:hover:bg-rose-900/20 transition-colors group border-t border-slate-100 dark:border-zinc-700/50 mt-1 pt-2">
                        <svg class="w-3.5 h-3.5 transition-transform duration-200 group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" />
                            <path class="origin-bottom-right transition-transform duration-300 group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                        </svg>
                        Forget Node
                    </button>
                </div>
            </transition>
        </Teleport>
    </div>
</template>