<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useFleetStore } from '../../stores/Fleet/fleet'
import type { FleetNode, InstalledSensor } from '../../stores/Fleet/fleet'
import BaseMeatballMenu from '../ui/navigation/BaseMeatballMenu.vue'
import BaseStatusDot from '../ui/feedback/BaseStatusDot.vue'
import { nodeMenuStyles } from '../nodedetails/nodedetailsStyles.ts'
import { filterStyles } from './dashboardWidgetsStyles.ts'

const fleetStore = useFleetStore()
const { nodes, selectedNode, selectedSensor } = storeToRefs(fleetStore)

const scrollArea = ref<HTMLElement | null>(null)
const showOfflineWarning = ref(false)

interface ActiveNode {
    nodeId: string
    alias: string
    sensors: InstalledSensor[]
    total: number
    online: number
    status: string
}

const activeNodes = computed<ActiveNode[]>(() => {
    return nodes.value.map((node: FleetNode) => {
        const sensorsList = node.installedSensors || []
        const total = sensorsList.length
        const online = sensorsList.filter(s => s.status === 'up').length

        return {
            nodeId: node.id,
            alias: node.alias,
            sensors: sensorsList,
            total,
            online,
            status: node.status // Backend already derives exact status (up, down, degraded)
        }
    })
})

const handleSilenceNode = (nodeId: string) => fleetStore.silenceNode(nodeId)
const handleForgetNode = async (nodeId: string) => {
    const node = fleetStore.nodes.find(n => n.id === nodeId)
    const alias = node ? node.alias : nodeId
    if (!confirm(`Delete Node "${alias}" aka "${nodeId}", ALL of its underlying sensors and events?`)) return
    const res = await fleetStore.deleteNode(nodeId)
    if (!res.success) alert(res.error)
}

const checkScroll = () => {
    if (!scrollArea.value) return
    const container = scrollArea.value
    const warningNodes = container.querySelectorAll('.has-warnings') as NodeListOf<HTMLElement>
    if (warningNodes.length === 0) { 
        showOfflineWarning.value = false; 
        return 
    }
    const lastWarning = warningNodes[warningNodes.length - 1]
    showOfflineWarning.value = lastWarning.getBoundingClientRect().right > (container.getBoundingClientRect().right + 5)
}

const scrollToSelectedPill = () => {
    nextTick(() => {
        const node = selectedNode.value
        const el = document.getElementById(node?.id ? `pill-${node.id}` : 'pill-all')
        if (el && scrollArea.value) {
            const container = scrollArea.value
            const scrollLeft = el.offsetLeft - container.clientWidth / 2 + el.clientWidth / 2
            container.scrollTo({ left: Math.max(0, scrollLeft), behavior: 'smooth' })
        }
    })
}

watch(() => selectedNode.value?.id, () => {
    scrollToSelectedPill()
})

watch(() => selectedSensor.value?.sensorId, () => {
    scrollToSelectedPill()
})

watch(activeNodes, () => {
    nextTick(() => {
        checkScroll()
        if (selectedNode.value?.id) {
            scrollToSelectedPill()
        }
    })
}, { deep: true })

onMounted(() => {
    nextTick(() => {
        checkScroll()
        if (selectedNode.value?.id) {
            scrollToSelectedPill()
        }
    })
})
</script>

<template>
    <div :class="filterStyles.bar">
        <div :class="filterStyles.scrollOuter">
            <div :class="filterStyles.viewport">
                <div :class="filterStyles.track">
                    <div ref="scrollArea" @scroll.passive="checkScroll" :class="filterStyles.scroll">
                        
                        <div :class="filterStyles.allSticky">
                            <button id="pill-all" @click="fleetStore.selectTarget(null, null)" 
                                    :class="[filterStyles.allPill, !selectedNode && !selectedSensor ? filterStyles.allActive : filterStyles.allInactive]">
                                All Traffic
                            </button>
                        </div>

                        <div v-for="n in activeNodes" :key="n.nodeId" :id="'pill-' + n.nodeId"
                            :class="[filterStyles.nodePill,
                                (selectedNode?.id === n.nodeId && !selectedSensor) ? filterStyles.nodeSelected :
                                (selectedNode?.id === n.nodeId && selectedSensor) ? filterStyles.nodeSelectedSensor :
                                filterStyles.nodeIdle,
                                ['down', 'degraded'].includes(n.status) ? 'has-warnings' : '',
                                (selectedNode && selectedNode?.id !== n.nodeId) ? filterStyles.nodeDimmed : ''
                            ]">
                             
                            <div @click="fleetStore.selectTarget(n.nodeId, null)" data-testid="node-pill-select" :class="filterStyles.selectZone" :title="`${n.online}/${n.total} Sensors Online`">
                                <BaseStatusDot :status="n.status" />
                                <span class="font-mono text-sm pointer-events-none">{{ n.alias }}</span>
                                <span class="text-sm opacity-60 ml-0.5">[{{n.total}}]</span>
                            </div>

                            <BaseMeatballMenu 
                                :id="`node-${n.nodeId}`" 
                                :inverted="selectedNode?.id === n.nodeId && !selectedSensor"
                                class="mx-2 opacity-50 group-hover/pill:opacity-100 transition-opacity"
                            >
                                <button @click="handleSilenceNode(n.nodeId)" 
                                        data-testid="filter-menu-silence"
                                        :class="[nodeMenuStyles.item, fleetStore.isNodeSilenced(n.nodeId) ? 'text-archive-text hover:bg-archive-bg' : ' hover:text-text-h']">
                                    <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <path v-if="!fleetStore.isNodeSilenced(n.nodeId)" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                                        <path v-if="fleetStore.isNodeSilenced(n.nodeId)" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                                    </svg>
                                    {{ fleetStore.isNodeSilenced(n.nodeId) ? 'Unsilence Node' : 'Silence Node' }}
                                </button>
                                
                                <button @click="handleForgetNode(n.nodeId)" 
                                        data-testid="filter-menu-forget"
                                        :class="nodeMenuStyles.itemDanger">
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
                
                <div :class="filterStyles.fadeEdge"></div>
            </div>
        </div>
    </div>
</template>