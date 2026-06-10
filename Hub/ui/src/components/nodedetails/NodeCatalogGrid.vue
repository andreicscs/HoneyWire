<script setup lang="ts">
const props = defineProps<{ manifests: any[], isLoading: boolean, fetchError: boolean, installedSensors?: any[] }>()
const emit = defineEmits<{ (e: 'open', manifest: any): void, (e: 'edit', sensor: any): void }>()

const isInstalled = (manifest: any) => {
    return props.installedSensors?.some((sensor: any) => sensor.id === manifest.id || sensor.sensorId === manifest.id || sensor.name === manifest.id)
}

const getInstalledSensor = (manifest: any) => {
    return props.installedSensors?.find((sensor: any) => sensor.id === manifest.id || sensor.sensorId === manifest.id || sensor.name === manifest.id)
}

const handleSensorClick = (s: any) => {
    const installed = getInstalledSensor(s)
    if (installed) {
        emit('edit', installed)
    } else {
        emit('open', s)
    }
}
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
            <div v-for="s in manifests" :key="s.id" @click="handleSensorClick(s)" class="bg-bg-surface border rounded-lg p-4 shadow-sm hover:shadow-md cursor-pointer transition-all duration-normal group flex flex-col relative overflow-hidden" :class="isInstalled(s) ? 'border-primary-main/50' : 'border-border-default hover:border-primary-main'">
                <div class="flex justify-between items-start mb-3">
                    <div class="w-10 h-10 rounded-md bg-bg-base border border-border-default/50 text-text-h flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform duration-normal"><svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="s.icon_svg"></path></svg></div>
                    <span class="px-2 py-0.5 rounded text-sm font-medium tracking-wider bg-bg-inset text-text-m border border-border-default/50">{{ s.osi_layer }}</span>
                </div>
                <h3 class="text-sm font-semibold text-text-h mb-1">{{ s.name }}</h3>
                <p class="text-sm text-text-m leading-relaxed line-clamp-2 mb-2 flex-grow">{{ s.description }}</p>
                <div class="flex justify-end items-center mt-auto">
                    <svg v-if="isInstalled(s)" class="w-5 h-5 text-primary-main" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" title="Installed"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                    <svg v-else class="w-5 h-5 text-text-m opacity-0 group-hover:opacity-100 transition-opacity" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" title="Install"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15"></path></svg>
                </div>
            </div>
        </div>
    </div>
</template>
