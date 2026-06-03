<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAppStore } from '../stores/System/app'
import { useFleetStore } from '../stores/Fleet/fleet'
import type { FleetNode } from '../stores/Fleet/fleet'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import BaseButton from '../components/ui/forms/BaseButton.vue'

import FleetSkeleton from '../components/ui/feedback/FleetSkeleton.vue'
import FleetDeployModal from '../components/ui/feedback/FleetDeployModal.vue'
import FleetNodeWidget from '../components/ui/layout/FleetNodeWidget.vue'

const appStore = useAppStore()
const fleetStore = useFleetStore()

// --- MANIFEST CATALOG ---
const isManifestLoading = ref(true)
const manifestData = ref<any[]>([])

const manifestMap = computed(() => {
    const map = new Map()
    for (const s of manifestData.value) {
        map.set(s.id, s)
        map.set(s.sensorId, s)
        map.set(s.name, s)
    }
    return map
})

const getOsiForSensor = (installedSensor: any): string => {
    const manifest = manifestMap.value.get(installedSensor.id)
        || manifestMap.value.get(installedSensor.name)
        || manifestMap.value.get(installedSensor.sensorId)
    return manifest?.osi_layer || installedSensor.osi || 'Other'
}

// --- PARALLEL LOAD ---
const isInitialLoading = ref(true)

onMounted(async () => {
    try {
        const [, manifests] = await Promise.all([
            fleetStore.fetchFleet(),
            fleetStore.fetchManifests().catch(err => {
                console.error('Failed to load manifests', err)
                return []
            })
        ])
        manifestData.value = manifests
    } finally {
        isManifestLoading.value = false
        isInitialLoading.value = false
    }
})

// --- DEPLOY MODAL ---
const showDeployModal = ref(false)

// --- OSI LAYER SORT ORDER ---
const osiOrder = ['Physical', 'Data Link', 'Network', 'Transport', 'Session', 'Presentation', 'Application', 'Other']

const sortOsi = (a: any, b: any) => {
    const aIdx = osiOrder.indexOf(a.type)
    const bIdx = osiOrder.indexOf(b.type)
    if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx
    if (aIdx !== -1) return -1
    if (bIdx !== -1) return 1
    return a.type.localeCompare(b.type)
}

// --- VIEW SPECIFIC TYPES ---
export interface DisplayNode extends FleetNode {
    totalSensors: number;
    onlineSensors: number;
    isSilenced: boolean;
    sensorSummary: { type: string; count: number; sensors: any[] }[];
    hasUpdate: boolean;
    isAwaitingCheckIn: boolean;
}

// --- DATA MAPPING ---
const displayNodes = computed<DisplayNode[]>(() => {
    if (isManifestLoading.value) {
        return fleetStore.nodes
            .filter(node => node.id && !node.id.startsWith('__pending_'))
            .map(node => {
                const sensorsList = node.installedSensors || []
                const totalSensors = sensorsList.length
                const onlineSensors = sensorsList.filter(s => s.status === 'up').length
                const isSilenced = totalSensors > 0 && sensorsList.every(s => s.isSilenced)
                return {
                    ...node,
                    totalSensors,
                    onlineSensors,
                    isSilenced,
                    sensorSummary: [],
                    hasUpdate: false,
                    isAwaitingCheckIn: node.status === 'pending' || (!node.lastHeartbeat && totalSensors === 0)
                }
            })
    }

    return fleetStore.nodes
        .filter(node => node.id && !node.id.startsWith('__pending_'))
        .map(node => {
            const sensorsList = node.installedSensors || []
            const onlineSensors = sensorsList.filter(s => s.status === 'up').length
            const totalSensors = sensorsList.length
            const isSilenced = totalSensors > 0 && sensorsList.every(s => s.isSilenced)

            const osiGroups = new Map()
            for (const sensor of sensorsList) {
                const osi = getOsiForSensor(sensor)
                if (!osiGroups.has(osi)) {
                    osiGroups.set(osi, [])
                }
                osiGroups.get(osi).push({
                    name: sensor.display || sensor.name,
                    status: sensor.status
                })
            }

            const sensorSummary = totalSensors > 0
                ? Array.from(osiGroups.entries())
                    .map(([type, sensors]) => ({ type, count: sensors.length, sensors }))
                    .sort(sortOsi)
                : []

            return {
                ...node,
                totalSensors,
                onlineSensors,
                isSilenced,
                sensorSummary,
                hasUpdate: false,
                isAwaitingCheckIn: node.status === 'pending' || (!node.lastHeartbeat && totalSensors === 0)
            }
        })
})

// --- ACTIONS (all delegated to store) ---
const handleUpdateNode = async (nodeId: string, updates: Partial<FleetNode>) => {
    const node = fleetStore.getNode(nodeId)
    if (!node) return
    try {
        await fleetStore.updateNode(nodeId, {
            alias: node.alias,
            tags: node.tags,
            publicIp: node.publicIp || '',
            privateIp: node.privateIp || '',
            ...updates
        })
    } catch (err) {
        // Store handles optimistic update rollback
    }
}
 
const handleSilenceNode = (nodeId: string) => fleetStore.silenceNode(nodeId)
const handleForgetNode = async (nodeId: string) => {
    if (!confirm(`Delete Node "${nodeId}" and ALL of its underlying sensors?`)) return
    const res = await fleetStore.deleteNode(nodeId)
    if (!res.success) alert(res.error)
}

const handleOpenNodeDetail = (nodeId: string) => {
    fleetStore.selectTarget(nodeId, null, false)
    appStore.setView('node-detail')
}

</script>

<template>
    <div class="min-h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6 pb-4 sm:pb-6">
        <div class="flex items-center justify-between shrink-0">
             <PageHeader 
                title="Fleet Overview" 
                description="Monitor the health and status of your deployed HoneyWire nodes across all environments. Click on a node for detailed insights and management options."
            />
            
            <BaseButton variant="primary" class="gap-2 text-sm" @click="showDeployModal = true">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                Deploy New Node
            </BaseButton>
        </div>

        <FleetSkeleton v-if="isInitialLoading" />

        <div v-else class="grid grid-cols-1 lg:grid-cols-3 2xl:grid-cols-4 gap-5 auto-rows-max">
            
            <FleetNodeWidget 
                v-for="node in displayNodes" 
                :key="node.id" 
                :node="node" 
                :isManifestLoading="isManifestLoading"
                :isDeleting="fleetStore.isNodeActionPending(node.id, 'deleting')"
                @update="handleUpdateNode(node.id, $event)"
                @silence="handleSilenceNode(node.id)"
                @delete="handleForgetNode(node.id)"
                @openDetail="handleOpenNodeDetail(node.id)"
            />
        </div>
    </div>
    
    <FleetDeployModal :show="showDeployModal" @close="showDeployModal = false" />
</template>