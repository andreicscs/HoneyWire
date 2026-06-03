<script setup lang="ts">
defineProps<{ manifests: any[], isLoading: boolean, fetchError: boolean }>()
defineEmits<{ (e: 'open', manifest: any): void }>()
</script>

<template>
    <div class="shrink-0">
        <h2 class="text-base font-semibold text-text-h mb-4">Sensor Catalog</h2>
        <div v-if="isLoading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            <div v-for="i in 4" :key="i" class="bg-bg-surface border border-border-default rounded-lg p-5 h-36 animate-pulse"></div>
        </div>
        <div v-else-if="fetchError" class="flex flex-col items-center justify-center py-20 text-center">
            <svg class="w-12 h-12 text-danger-text mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
            <h3 class="text-base font-medium text-text-h">Unable to reach Sensor Registry</h3>
            <p class="text-base text-text-m mt-2 max-w-md">Please ensure this Hub has connectivity access to pull the latest sensor manifests.</p>
        </div>
        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            <div v-for="s in manifests" :key="s.id" @click="$emit('open', s)" class="bg-bg-surface border border-border-default rounded-lg p-4 shadow-sm hover:border-primary-main hover:shadow-md cursor-pointer transition-all duration-normal group flex flex-col">
                <div class="flex justify-between items-start mb-3">
                    <div class="w-10 h-10 rounded-md bg-bg-base border border-border-default/50 text-text-h flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform duration-normal"><svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="s.icon_svg"></path></svg></div>
                    <span class="px-2 py-0.5 rounded text-sm font-medium tracking-wider bg-bg-inset text-text-m border border-border-default/50">{{ s.osi_layer }}</span>
                </div>
                <h3 class="text-sm font-semibold text-text-h mb-1">{{ s.name }}</h3>
                <p class="text-sm text-text-m leading-relaxed line-clamp-2">{{ s.description }}</p>
            </div>
        </div>
    </div>
</template>