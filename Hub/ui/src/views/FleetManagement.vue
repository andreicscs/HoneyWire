<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAppStore } from '../stores/System/app.ts'
import { useFleetStore } from '../stores/Fleet/fleet.ts'
import type { FleetNode } from '../stores/Fleet/fleet.ts'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import BaseButton from '../components/ui/forms/BaseButton.vue'

import FleetSkeleton from '../components/fleetmanagement/FleetSkeleton.vue'
import FleetDeployModal from '../components/fleetmanagement/FleetDeployModal.vue'
import FleetNodeWidget from '../components/fleetmanagement/FleetNodeWidget.vue'

const appStore = useAppStore()
const fleetStore = useFleetStore()

// --- MANIFEST CATALOG ---
const isManifestLoading = ref(true)
const manifestData = ref<any[]>([])

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
                v-for="node in fleetStore.enrichedNodes" 
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