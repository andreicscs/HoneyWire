<script setup>
import { ref, computed, nextTick, watch, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useFleetStore } from '../../stores/fleet'
import BaseMeatballMenu from '../ui/navigation/BaseMeatballMenu.vue'
import BaseStatusDot from '../ui/feedback/BaseStatusDot.vue'

const fleetStore = useFleetStore()
const { sensors, selectedNode, selectedSensor } = storeToRefs(fleetStore)

const scrollArea = ref(null)
const showOfflineWarning = ref(false)

const isNodeSilenced = (node) => {
    if (!node || !node.sensors.length) return false
    return node.sensors.every(s => s.is_silenced)
}

const activeNodes = computed(() => {
    const nodesMap = {};
    sensors.value.forEach(s => {
        if (!s.node_id) return;
        if (!nodesMap[s.node_id]) {
            nodesMap[s.node_id] = { node_id: s.node_id, alias: s.node_id, sensors: [], total: 0, online: 0 };
        }
        nodesMap[s.node_id].sensors.push(s);
        nodesMap[s.node_id].total++;
        if (s.status === 'online') nodesMap[s.node_id].online++;
    });

    return Object.values(nodesMap).map(n => {
        let status = 'offline';
        if (n.online === n.total) status = 'up'; 
        else if (n.online > 0) status = 'degraded';
        else status = 'down';
        return { ...n, status };
    });
});

const handleSilenceNode = (nodeId) => fleetStore.silenceNode(nodeId)
const handleForgetNode = (nodeId) => fleetStore.forgetNode(nodeId)

const checkScroll = () => {
    if (!scrollArea.value) return
    const container = scrollArea.value
    const warningNodes = container.querySelectorAll('.has-warnings')
    if (warningNodes.length === 0) { 
        showOfflineWarning.value = false; 
        return 
    }
    const lastWarning = warningNodes[warningNodes.length - 1]
    showOfflineWarning.value = lastWarning.getBoundingClientRect().right > (container.getBoundingClientRect().right + 5)
}

watch(() => selectedNode.value, (newVal) => {
    nextTick(() => {
        const el = document.getElementById(newVal ? `pill-${newVal}` : 'pill-all')
        if (el) el.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
    })
})

watch(() => selectedSensor.value, (newVal) => {
    nextTick(() => {
        const elId = newVal && selectedNode.value ? `pill-${selectedNode.value}` : 'pill-all'
        const el = document.getElementById(elId)
        if (el) el.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
    })
})

watch(activeNodes, () => nextTick(checkScroll), { deep: true })

onMounted(() => nextTick(checkScroll))
</script>

<template>
    <div class="flex justify-between items-center gap-4">
        <div class="flex items-center w-full gap-2">
            <div class="flex-1 relative overflow-hidden">
                <div class="relative flex overflow-x-auto gap-1">
                    <div ref="scrollArea" @scroll.passive="checkScroll" class="flex overflow-x-auto whitespace-nowrap gap-2 items-center custom-scroll pb-3 pr-2 relative">
                        
                        <div class="sticky left-0 z-20 pr-2 bg-bg flex items-center border-r border-border-default transition-colors duration-normal">
                            <button id="pill-all" @click="fleetStore.selectTarget(null, null)" 
                                    class="shrink-0 px-3.5 py-1.5 rounded-md border text-sm font-medium transition-colors duration-normal shadow-sm outline-none"
                                    :class="!selectedNode && !selectedSensor 
                                        ? 'bg-primary-selected text-primary-text border-primary-selected' 
                                        : 'bg-secondary-main border-secondary-border text-text-m hover:bg-secondary-hover hover:text-text-h'">
                                All Traffic
                            </button>
                        </div>

                        <div v-for="n in activeNodes" :key="n.node_id" :id="'pill-' + n.node_id"
                            class="shrink-0 relative flex items-center rounded-md border transition-all duration-normal shadow-sm group/pill"
                            :class="[
                                (selectedNode === n.node_id && !selectedSensor) ? 'bg-primary-selected text-primary-text border-primary-selected' : 
                                (selectedNode === n.node_id && selectedSensor) ? 'bg-highlight-bg border-highlight-border text-highlight-text ring-1 ring-highlight-ring' : 
                                'bg-secondary-main border-secondary-border text-text-m hover:bg-secondary-hover hover:text-text-h',
                                
                                ['down', 'degraded'].includes(n.status) ? 'has-warnings' : '',
                                
                                (selectedNode && selectedNode !== n.node_id) ? 'opacity-50' : ''
                            ]">
                             
                            <div @click="fleetStore.selectTarget(n.node_id, null)" class="flex items-center gap-2 pl-2.5 py-1 cursor-pointer flex-1" :title="`${n.online}/${n.total} Sensors Online`">
                                <BaseStatusDot :status="n.status" />
                                <span class="font-mono text-sm pointer-events-none">{{ n.alias }}</span>
                                <span class="text-sm opacity-60 ml-0.5">[{{n.total}}]</span>
                            </div>

                            <BaseMeatballMenu 
                                :id="`node-${n.node_id}`" 
                                :inverted="selectedNode === n.node_id && !selectedSensor"
                                class="mx-2 opacity-50 group-hover/pill:opacity-100 transition-opacity"
                            >
                                <button @click="handleSilenceNode(n.node_id)" 
                                        class="w-full text-left px-3 py-2 text-sm font-medium flex items-center gap-2 text-text-m hover:bg-secondary-hover transition-colors group"
                                        :class="isNodeSilenced(n) ? 'text-archive-text hover:bg-archive-bg' : ' hover:text-text-h'">
                                    <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <path v-if="!isNodeSilenced(n)" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                                        <path v-if="isNodeSilenced(n)" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                                    </svg>
                                    {{ isNodeSilenced(n) ? 'Unsilence Node' : 'Silence Node' }}
                                </button>
                                
                                <button @click="handleForgetNode(n.node_id)" 
                                        class="w-full text-left px-3 py-2 text-sm font-medium text-danger-text flex items-center gap-2 hover:bg-danger-bg transition-colors group border-t border-border-default mt-1 pt-2">
                                    <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" />
                                        <path class="origin-bottom-right transition-transform duration-normal group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                                    </svg>
                                    Forget Node
                                </button>
                            </BaseMeatballMenu>
                        </div>
                    </div>
                    
                    <div class="w-6 h-8 shrink-0 flex items-center justify-center z-10 pointer-events-none">
                        <transition enter-active-class="transition-opacity duration-normal ease-out" leave-active-class="transition-opacity duration-normal ease-in" enter-from-class="opacity-0" leave-to-class="opacity-0">
                            <div v-show="showOfflineWarning" class="flex items-center justify-center">
                                <span class="w-1.5 h-1.5 rounded-full bg-high animate-pulse shadow-sm"></span>
                            </div>
                        </transition>
                    </div>
                </div>
                
                <div class="absolute right-5 top-0 bottom-7 w-10 h-8 bg-gradient-to-l from-bg to-transparent pointer-events-none z-10"></div>
            </div>
        </div>
    </div>
</template>>