<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useFleetStore } from '../stores/Fleet/fleet.ts'
import type { FleetNode } from '../stores/Fleet/fleet.ts'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import BaseButton from '../components/ui/forms/BaseButton.vue'

import FleetSkeleton from '../components/fleetmanagement/FleetSkeleton.vue'
import FleetDeployModal from '../components/fleetmanagement/FleetDeployModal.vue'
import FleetNodeWidget from '../components/fleetmanagement/FleetNodeWidget.vue'

const fleetStore = useFleetStore()
const router = useRouter()

// --- MANIFEST CATALOG ---
const isManifestLoading = ref(true)
const manifestData = ref<any[]>([])

// --- PARALLEL LOAD ---
const isInitialLoading = ref(true)
const showSkeleton = ref(false)

onMounted(async () => {
    const skeletonTimer = setTimeout(() => {
        if (isInitialLoading.value) showSkeleton.value = true
    }, 350)

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
        clearTimeout(skeletonTimer)
        isManifestLoading.value = false
        isInitialLoading.value = false
        showSkeleton.value = false
    }
})

// --- DEPLOY MODAL ---
const showDeployModal = ref(false)

// TODO: REMOVE DEBUG OVERRIDE BEFORE PRODUCTION
// You can test the "First Startup" UI state at any time by running:
// localStorage.setItem('DEBUG_FIRST_STARTUP', 'true') in your browser console.
const isFirstStartup = computed(() => {
    return !isInitialLoading.value && (fleetStore.nodes.length === 0 || localStorage.getItem('DEBUG_FIRST_STARTUP') === 'true')
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
    router.push({ name: 'node-detail', params: { id: nodeId } })
}

</script>

<template>
    <div class="min-h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6 pb-4 sm:pb-6">

        <div class="flex items-center justify-between shrink-0">
             <PageHeader 
                title="Fleet Overview" 
                description="Monitor the health and status of your deployed HoneyWire nodes across all environments. Click on a node for detailed insights and management options."
            />
            
            <BaseButton 
                variant="primary" 
                class="gap-2 text-sm transition-all relative" 
                :class="{ 'animate-bounce-subtle ring-4 ring-primary-main/30 ring-offset-2 ring-offset-bg shadow-lg': isFirstStartup }"
                @click="showDeployModal = true"
            >
                <span v-if="isFirstStartup" class="absolute -top-2 -right-2 flex h-4 w-4"><span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary-main opacity-75"></span><span class="relative inline-flex rounded-full h-4 w-4 bg-primary-main"></span></span>
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                Deploy New Node
            </BaseButton>
        </div>

        <FleetSkeleton v-if="showSkeleton" />

        <div v-else-if="!isInitialLoading && fleetStore.enrichedNodes.length === 0" class="flex-1 flex items-center justify-center text-center text-base text-text-m py-20 z-20">
            No nodes deployed.
        </div>

        <div v-else-if="!isInitialLoading" class="grid grid-cols-1 lg:grid-cols-3 2xl:grid-cols-4 gap-5 auto-rows-max">
            
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