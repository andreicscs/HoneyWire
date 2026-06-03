<script setup lang="ts">
import BaseMeatballMenu from '../ui/navigation/BaseMeatballMenu.vue'
import { formatSensorId } from '../../utils/formatSensorId'

const props = defineProps<{ sensors: any[] }>()
defineEmits<{ (e: 'edit', sensor: any): void, (e: 'toggleSilence', sensor: any): void, (e: 'remove', sensor: any): void }>()
</script>

<template>
    <div>
        <h3 class="text-sm font-semibold text-text-h mb-4 mt-2">Deployed Sensors</h3>
        <div v-if="(sensors?.length || 0) > 0" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-5 gap-4">
            <div v-for="sensor in sensors" :key="sensor.id" class="bg-bg-surface border border-border-default rounded-lg p-4 flex flex-col group transition-colors shadow-sm relative overflow-hidden">
                <div class="absolute top-0 left-0 right-0 h-1 transition-colors" :class="sensor.status === 'up' ? 'bg-success-main' : 'bg-danger-main'"></div>
                <div class="flex justify-between items-start mt-1">
                    <div class="flex items-center gap-3 min-w-0">
                        <div class="w-8 h-8 rounded bg-bg-base border border-border-default/50 flex items-center justify-center text-text-h shrink-0">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="sensor.icon"></path></svg>
                        </div>
                        <div class="min-w-0">
                            <h4 class="text-sm font-semibold text-text-h truncate">{{ sensor.display }}</h4>
                            <span class="text-sm text-text-m font-mono block truncate">{{ formatSensorId(sensor.name) }}</span>
                        </div>
                    </div>
                    <BaseMeatballMenu :id="`sensor-menu-${sensor.id}`">
                        <button @click="$emit('edit', sensor)" class="w-full text-left px-3 py-2 text-sm font-medium flex items-center gap-2 text-text-m hover:bg-secondary-hover hover:text-text-h transition-colors group"><svg class="w-3.5 h-3.5 overflow-visible" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5" /><path class="transition-transform duration-normal group-hover:-translate-y-0.5 group-hover:translate-x-0.5 group-hover:-rotate-6" d="M17.586 3.586a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>Edit</button>
                        <button @click="$emit('toggleSilence', sensor)" class="w-full text-left px-3 py-2 text-sm font-medium flex items-center gap-2 text-text-m hover:bg-secondary-hover transition-colors group" :class="sensor.isSilenced ? 'text-archive-text hover:bg-archive-bg' : ' hover:text-text-h'"><svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path v-if="!sensor.isSilenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/><path v-if="sensor.isSilenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/></svg>{{ sensor.isSilenced ? 'Unsilence Alerts' : 'Silence Alerts' }}</button>
                        <button @click="$emit('remove', sensor)" class="w-full text-left px-3 py-2 text-sm font-medium text-danger-text flex items-center gap-2 hover:bg-danger-bg transition-colors group border-t border-border-default mt-1 pt-2"><svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" /><path class="origin-bottom-right transition-transform duration-normal group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" /></svg>Remove Sensor</button>
                    </BaseMeatballMenu>
                </div>
                <div class="mt-3 pt-3 border-t border-border-default flex justify-between items-center">
                    <span class="px-1.5 py-0.5 rounded text-sm font-medium tracking-wider bg-bg-inset text-text-m border border-border-default/50">{{ sensor.osi }}</span>
                    <svg v-if="sensor.isSilenced" class="w-3.5 h-3.5 text-medium shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" title="Alerts Silenced"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/></svg>
                </div>
            </div>
        </div>
        <div v-else class="border border-dashed border-border-default rounded-lg p-8 flex flex-col items-center justify-center text-center bg-bg-surface/50">
            <p class="text-sm text-text-h font-medium">No sensors deployed</p>
            <p class="text-sm text-text-m mt-1">Select a sensor from the catalog below.</p>
        </div>
    </div>
</template>