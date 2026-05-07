<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick, computed } from 'vue'

const props = defineProps({
    fleet: { type: Array, required: true },
    selectedNode: { type: String, default: null },
    selectedSensor: { type: String, default: null } 
})

const emit = defineEmits(['select-node', 'silence-node', 'forget-node'])

const scrollArea = ref(null)
const showOfflineWarning = ref(false)

// Meatball Menu State
const activeMenu = ref(null)
const menuPos = ref({ top: '0px', left: '0px' })

const activeNodeData = computed(() => activeNodes.value.find(n => n.node_id === activeMenu.value))

const isNodeSilenced = computed(() => {
    if (!activeNodeData.value) return false;
    // We consider the node "silenced" if ALL its sensors are currently silenced.
    return activeNodeData.value.sensors.every(s => s.is_silenced);
})

const highlightedNodeId = computed(() => {
    if (!props.selectedSensor) return null;
    const sensor = props.fleet.find(s => s.sensor_id === props.selectedSensor);
    return sensor ? sensor.node_id : null;
});

const activeNodes = computed(() => {
    const nodesMap = {};
    
    props.fleet.forEach(s => {
        if (!s.node_id) return;
        
        if (!nodesMap[s.node_id]) {
            nodesMap[s.node_id] = { 
                node_id: s.node_id, 
                alias: s.node_id,
                sensors: [], 
                total: 0, 
                online: 0 
            };
        }
        
        nodesMap[s.node_id].sensors.push(s);
        nodesMap[s.node_id].total++;
        if (s.status === 'online') nodesMap[s.node_id].online++;
    });

    return Object.values(nodesMap).map(n => {
        let status = 'offline';
        if (n.online === n.total) status = 'online';
        else if (n.online > 0) status = 'degraded';
        
        return { ...n, status };
    });
});

const toggleMenu = (e, id) => {
    if (activeMenu.value === id) {
        activeMenu.value = null
        return
    }
    const rect = e.currentTarget.getBoundingClientRect()
    menuPos.value = { top: rect.bottom + 6 + 'px', left: rect.left + 'px' }
    activeMenu.value = id
}

const handleSilenceNode = (nodeId) => {
    emit('silence-node', nodeId)
    activeMenu.value = null
}

const handleForgetNode = (nodeId) => {
    emit('forget-node', nodeId)
    activeMenu.value = null
}

const closeMenu = (e) => { if (!e.target.closest('.node-dropdown') && !e.target.closest('.meatball-toggle')) activeMenu.value = null }
const closeOnScroll = () => { if (activeMenu.value) activeMenu.value = null }

const checkScroll = () => {
    if (!scrollArea.value) return
    const container = scrollArea.value
    const offlineBtns = container.querySelectorAll('.is-offline, .is-degraded')
    if (offlineBtns.length === 0) { showOfflineWarning.value = false; return }
    const lastOffline = offlineBtns[offlineBtns.length - 1]
    showOfflineWarning.value = lastOffline.getBoundingClientRect().right > (container.getBoundingClientRect().right + 5)
}

watch(() => props.selectedNode, (newVal) => {
    nextTick(() => {
        const el = document.getElementById(newVal ? `pill-${newVal}` : 'pill-all')
        if (el) el.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
    })
})

watch(() => props.selectedSensor, (newVal) => {
    nextTick(() => {
        const elId = newVal && props.selectedNode ? `pill-${props.selectedNode}` : 'pill-all'
        const el = document.getElementById(elId)
        if (el) el.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
    })
})

watch(activeNodes, () => nextTick(checkScroll), { deep: true })

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
    <div class="flex justify-between items-center gap-4">
        <div class="flex items-center w-full gap-2">
            <div class="flex-1 relative overflow-hidden">
                <div class="relative flex overflow-x-auto gap-1">
                    <div ref="scrollArea" @scroll.passive="checkScroll" class="flex overflow-x-auto whitespace-nowrap gap-2 items-center custom-scroll pb-3 pr-2 relative">
                        
                        <div class="sticky left-0 z-20 pr-2 bg-slate-200 dark:bg-[#0a0a0c] flex items-center border-r border-slate-300 dark:border-zinc-800 transition-all duration-200">
                            <button id="pill-all" @click="$emit('select-node', null)" 
                                    class="shrink-0 px-3.5 py-1.5 rounded-md border text-sm font-bold transition-all duration-300 shadow-sm"
                                    :class="!selectedNode && !selectedSensor ? 'bg-slate-800 text-white dark:bg-zinc-200 dark:text-black border-slate-800 dark:border-zinc-200' : 'bg-white dark:bg-zinc-900 border-slate-300 dark:border-zinc-700 text-slate-600 dark:text-zinc-300 hover:bg-slate-50 dark:hover:bg-zinc-800'">
                                All Traffic
                            </button>
                        </div>

                        <!-- Pill Wrapper is now a div to safely contain the menu button -->
                        <div v-for="n in activeNodes" :key="n.node_id" :id="'pill-' + n.node_id"
                             class="shrink-0 relative flex items-center rounded-md border transition-all duration-300 shadow-sm group/pill"
                             :class="[
                                 /* Dark Selected Style: Node is selected, NO sensor is selected */
                                 (selectedNode === n.node_id && !selectedSensor) ? 'bg-slate-700 text-white dark:bg-zinc-300 dark:text-zinc-900 border-slate-700 dark:border-zinc-300' : 
                                 
                                 /* Blue Highlight Style: Node is selected, AND a specific sensor is selected */
                                 (selectedNode === n.node_id && selectedSensor) ? 'bg-blue-50 dark:bg-transparent border-blue-400 dark:border-zinc-400 text-blue-900 dark:text-zinc-200 ring-1 ring-blue-500/30 dark:ring-zinc-400/50' : 
                                 
                                 /* Default Unselected Style */
                                 'bg-white dark:bg-zinc-900 border-slate-300 dark:border-zinc-800 text-slate-600 dark:text-zinc-400 hover:bg-slate-50 dark:hover:bg-zinc-800',
                                 
                                 n.status === 'offline' ? 'is-offline' : '',
                                 n.status === 'degraded' ? 'is-degraded' : '',
                                 
                                 /* Dimming logic: Dim if ANY node is selected, and it's NOT this specific node */
                                 (selectedNode && selectedNode !== n.node_id) ? 'opacity-40 grayscale-[50%]' : ''
                             ]">
                             
                            <!-- Clickable Filter Area -->
                            <div @click="$emit('select-node', n.node_id)" class="flex items-center gap-2 pl-3 pr-1 py-1.5 cursor-pointer flex-1" :title="`${n.online}/${n.total} Sensors Online`">
                                <span class="w-1.5 h-1.5 rounded-full" 
                                      :class="{
                                          'bg-emerald-500': n.status === 'online',
                                          'bg-amber-500': n.status === 'degraded',
                                          'bg-rose-500': n.status === 'offline'
                                      }"></span>
                                
                                <span class="mono text-xs font-semibold pointer-events-none">{{ n.alias }}</span>
                                <span class="text-[9px] font-semibold opacity-60 ml-0.5">[{{n.total}}]</span>
                            </div>

                            <!-- Meatball Toggle -->
                            <div @click.stop="toggleMenu($event, n.node_id)" 
                                 class="meatball-toggle w-5 h-5 mr-1 rounded flex items-center justify-center transition-all cursor-pointer shrink-0 opacity-40 group-hover/pill:opacity-100"
                                 :class="[
                                     activeMenu === n.node_id ? 'opacity-100 text-slate-800 dark:text-white bg-black/10 dark:bg-white/20' :
                                     selectedNode === n.node_id ? 'hover:bg-white/20 dark:hover:bg-black/10' :
                                     'hover:text-slate-800 dark:hover:text-white hover:bg-black/5 dark:hover:bg-white/10'
                                 ]">
                                <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/></svg>
                            </div>
                        </div>
                    </div>
                    
                    <div class="w-6 h-8 shrink-0 flex items-center justify-center z-1">
                        <transition enter-active-class="transition-opacity duration-300 ease-out" leave-active-class="transition-opacity duration-300 ease-in" enter-from-class="opacity-0" leave-to-class="opacity-0">
                            <div v-show="showOfflineWarning" class="flex items-center justify-center">
                                <span class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></span>
                            </div>
                        </transition>
                    </div>
                </div>
                <div class="absolute right-5 top-0 bottom-7 w-10 bg-gradient-to-l from-slate-200/60 dark:from-[#0a0a0c] to-transparent pointer-events-none"></div>
            </div>
        </div>

        <Teleport to="body">
            <transition enter-active-class="transition ease-out duration-100" enter-from-class="transform opacity-0 scale-95" enter-to-class="transform opacity-100 scale-100" leave-active-class="transition ease-in duration-75" leave-from-class="transform opacity-100 scale-100" leave-to-class="transform opacity-0 scale-95">
                <div v-if="activeMenu && activeNodeData" 
                     :style="{ top: menuPos.top, left: menuPos.left }"
                     class="node-dropdown fixed w-40 rounded-md shadow-xl bg-white dark:bg-zinc-800 border border-slate-200 dark:border-zinc-700 z-[100] py-1 overflow-hidden">
                    
                    <button @click.stop="handleSilenceNode(activeNodeData.node_id)" 
                            class="w-full text-left px-3 py-2 text-xs font-semibold flex items-center gap-2 hover:bg-slate-100 dark:hover:bg-zinc-700 transition-colors group"
                            :class="isNodeSilenced ? 'text-amber-600 dark:text-amber-500' : 'text-slate-600 dark:text-zinc-300'">
                        <svg class="w-3.5 h-3.5 transition-transform duration-200 group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path v-if="!isNodeSilenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                            <path v-if="isNodeSilenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                        </svg>
                        {{ isNodeSilenced ? 'Unsilence Node' : 'Silence Node' }}
                    </button>
                    
                    <button @click="handleForgetNode(activeNodeData.node_id)" 
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