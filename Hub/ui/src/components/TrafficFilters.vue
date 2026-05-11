<script setup>
import { ref, computed, nextTick, watch, onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useFleetStore } from '../stores/fleet'

const fleetStore = useFleetStore()
const { sensors, selectedNode, selectedSensor } = storeToRefs(fleetStore)

const scrollArea = ref(null)
const showOfflineWarning = ref(false)

// Meatball Menu State
const activeMenu = ref(null)
const menuPos = ref({ top: '0px', left: '0px' })

const activeNodeData = computed(() => activeNodes.value.find(n => n.node_id === activeMenu.value))

const isNodeSilenced = computed(() => {
    if (!activeNodeData.value) return false;
    return activeNodeData.value.sensors.every(s => s.is_silenced);
})

// FIXED: Removed highlightedNodeId completely to prevent duplicate ID bugs.

const activeNodes = computed(() => {
    const nodesMap = {};
    
    sensors.value.forEach(s => {
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
    fleetStore.silenceNode(nodeId)
    activeMenu.value = null
}

const handleForgetNode = (nodeId) => {
    fleetStore.forgetNode(nodeId)
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

watch(() => selectedNode.value, (newVal) => {
    nextTick(() => {
        const el = document.getElementById(newVal ? `pill-${newVal}` : 'pill-all')
        if (el) el.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
    })
})

watch(() => selectedSensor.value, (newVal) => {
    nextTick(() => {
        // FIXED: Relying on selectedNode.value instead of highlightedNodeId
        const elId = newVal && selectedNode.value ? `pill-${selectedNode.value}` : 'pill-all'
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
                        
                        <div class="sticky left-0 z-20 pr-2 bg-bg-bg flex items-center border-r border-border-default transition-all duration-200">
                            
                            <button id="pill-all" @click="fleetStore.selectTarget(null, null)" 
                                    class="shrink-0 px-3.5 py-1.5 rounded-md border text-sm font-bold transition-all duration-300 shadow-sm"
                                    :class="!selectedNode && !selectedSensor ? 'bg-select-solid-bg text-select-solid-text border-select-solid-bg' : 'bg-bg-surface border-border-default text-text-muted hover:bg-button-hover hover:text-text-main'">
                                All Traffic
                            </button>
                        </div>

                        <div v-for="n in activeNodes" :key="n.node_id" :id="'pill-' + n.node_id"
                            class="shrink-0 relative flex items-center rounded-md border transition-all duration-300 shadow-sm group/pill"
                            :class="[
                                /* Solid Selected Style */
                                (selectedNode === n.node_id && !selectedSensor) ? 'bg-select-solid-bg text-select-solid-text border-select-solid-bg' : 
                                
                                /* Blue Highlight Style (Sensor selected) */
                                (selectedNode === n.node_id && selectedSensor) ? 'bg-highlight-bg border-highlight-border text-highlight-text ring-1 ring-highlight-ring' : 
                                
                                /* Default */
                                'bg-bg-surface border-border-default text-text-muted hover:bg-button-hover hover:text-text-main',
                                
                                n.status === 'offline' ? 'is-offline' : '',
                                n.status === 'degraded' ? 'is-degraded' : '',
                                
                                (selectedNode && selectedNode !== n.node_id) ? 'opacity-70 grayscale-[50%]' : ''
                            ]">
                             
                            <div @click="fleetStore.selectTarget(n.node_id, null)" class="flex items-center gap-2 pl-3 pr-1 py-1.5 cursor-pointer flex-1" :title="`${n.online}/${n.total} Sensors Online`">
                                <span class="w-1.5 h-1.5 rounded-full" 
                                      :class="{
                                          'bg-success-main': n.status === 'online',
                                          'bg-high': n.status === 'degraded',
                                          'bg-critical': n.status === 'offline'
                                      }"></span>
                                
                                <span class="mono text-xs font-semibold pointer-events-none">{{ n.alias }}</span>
                                <span class="text-[9px] font-semibold opacity-60 ml-0.5">[{{n.total}}]</span>
                            </div>

                            <div @click.stop="toggleMenu($event, n.node_id)" 
                                 class="meatball-toggle w-5 h-5 mr-1 rounded flex items-center justify-center transition-all cursor-pointer shrink-0 opacity-40 group-hover/pill:opacity-100"
                                 :class="[
                                     activeMenu === n.node_id ? 'opacity-100 text-text-main bg-button-selected' :
                                     selectedNode === n.node_id ? 'hover:bg-bg-surface/20' :
                                     'hover:text-text-main hover:bg-button-hover'
                                 ]">
                                <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/></svg>
                            </div>
                        </div>
                    </div>
                    
                    <div class="w-6 h-8 shrink-0 flex items-center justify-center z-1">
                        <transition enter-active-class="transition-opacity duration-300 ease-out" leave-active-class="transition-opacity duration-300 ease-in" enter-from-class="opacity-0" leave-to-class="opacity-0">
                            <div v-show="showOfflineWarning" class="flex items-center justify-center">
                                <span class="w-1.5 h-1.5 rounded-full bg-high animate-pulse"></span>
                            </div>
                        </transition>
                    </div>
                </div>
                <div class="absolute right-5 top-0 bottom-7 w-10 h-8 bg-gradient-to-l from-bg-bg to-transparent pointer-events-none"></div>
            </div>
        </div>

        <Teleport to="body">
            <transition enter-active-class="transition ease-out duration-100" enter-from-class="transform opacity-0 scale-95" enter-to-class="transform opacity-100 scale-100" leave-active-class="transition ease-in duration-75" leave-from-class="transform opacity-100 scale-100" leave-to-class="transform opacity-0 scale-95">
                <div v-if="activeMenu && activeNodeData" 
                     :style="{ top: menuPos.top, left: menuPos.left }"
                     class="node-dropdown fixed w-40 rounded-md shadow-xl bg-bg-surface border border-border-default z-[100] py-1 overflow-hidden">
                    
                    <button @click.stop="handleSilenceNode(activeNodeData.node_id)" 
                            class="w-full text-left px-3 py-2 text-xs font-semibold flex items-center gap-2 hover:bg-button-hover transition-colors group"
                            :class="isNodeSilenced ? 'text-archive-text' : 'text-text-muted hover:text-text-main'">
                        <svg class="w-3.5 h-3.5 transition-transform duration-200 group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path v-if="!isNodeSilenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                            <path v-if="isNodeSilenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                        </svg>
                        {{ isNodeSilenced ? 'Unsilence Node' : 'Silence Node' }}
                    </button>
                    
                    <button @click="handleForgetNode(activeNodeData.node_id)" 
                            class="w-full text-left px-3 py-2 text-xs font-semibold text-danger-text flex items-center gap-2 hover:bg-danger-bg-subtle transition-colors group border-t border-border-default mt-1 pt-2">
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