<script setup lang="ts">
import { computed } from 'vue'
import { formatSensorId } from '../../utils/formatSensorId'

const props = defineProps<{
    node: any,
    recentActivity: any[]
}>()

const emit = defineEmits<{
    (e: 'viewAllEvents'): void
}>()

const maxSensorEvents = computed(() => {
  const sensors = props.node?.installedSensors || []
  if (sensors.length === 0) return 1
  return Math.max(...sensors.map((s: any) => s.events24h || 0), 1)
})

const topSensors = computed(() => {
  return [...(props.node?.installedSensors || [])]
    .sort((a: any, b: any) => (b.events24h || 0) - (a.events24h || 0))
})

const formatEventType = (type: string) => type ? type.replace(/_/g, ' ') : ''
</script>

<template>
    <div class="grid grid-cols-1 xl:grid-cols-2 gap-5">
        <!-- Event Volume (24h) -->
        <div class="bg-bg-surface w-full max-w-2xl border border-border-default rounded-lg p-5 shadow-sm flex flex-col">
            <h3 class="text-sm font-semibold text-text-h mb-4">Event Volume (24h)</h3>
            <div v-if="topSensors.length > 0" class="space-y-3 overflow-y-auto custom-scroll max-h-[240px] pr-1">
                <div v-for="sensor in topSensors" :key="sensor.id">
                    <div class="flex items-center justify-between mb-1.5">
                        <span class="text-sm font-medium text-text-h truncate pr-4">{{ sensor.display }}</span>
                        <span class="text-sm font-mono text-text-m">{{ sensor.events24h }}</span>
                    </div>
                    <div class="w-full bg-bg-inset border border-border-default rounded-full h-2">
                        <div class="bg-text-m h-full rounded-full transition-all duration-normal" :style="`width: ${(sensor.events24h / maxSensorEvents) * 100}%`"></div>
                    </div>
                </div>
            </div>
            <div v-else class="text-sm text-text-m italic">No events recorded.</div>
        </div>

        <!-- Recent Activity — mini event table -->
        <div class="bg-bg-surface border border-border-default rounded-lg flex flex-col overflow-hidden shadow-sm">
            <div class="px-4 py-3 border-b border-border-default flex items-center justify-between bg-bg-surface shrink-0">
                <h3 class="text-sm font-semibold text-text-h">Recent Activity</h3>
                <button @click="$emit('viewAllEvents')" class="text-sm font-medium text-text-m hover:text-text-h transition-colors outline-none">View All &rarr;</button>
            </div>
            
            <div class="flex-1 overflow-y-auto custom-scroll bg-bg-surface max-h-[240px]">
                <table class="w-full text-left border-collapse">
                    <thead class="text-sm font-medium text-text-m tracking-wider sticky top-0 bg-bg-surface shadow-[0_1px_0_0_var(--color-border-default)]">
                        <tr>
                            <th class="px-3 py-2 w-14"></th>
                            <th class="px-3 py-2">Event</th>
                            <th class="px-3 py-2">Source</th>
                            <th class="px-3 py-2">Sensor</th>
                            <th class="px-3 py-2 text-right">Time</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-if="recentActivity.length === 0">
                            <td colspan="5" class="px-3 py-4 text-center text-sm text-text-m">No recent activity on this node.</td>
                        </tr>
                        <tr v-for="event in recentActivity" :key="event.id" class="hover:bg-secondary-hover cursor-default transition-colors duration-[var(--duration-fast)] relative z-0" :class="'bleed-' + event.severity.toLowerCase()">
                            <td class="px-3 py-2 border-b border-border-default"><span class="px-1.5 py-0.5 rounded border text-sm font-medium bg-bg-base whitespace-nowrap capitalize" :style="{ borderColor: `var(--color-${event.severity.toLowerCase()})`, color: `var(--color-${event.severity.toLowerCase()})` }">{{ event.severity }}</span></td>
                            <td class="px-3 py-2 border-b border-border-default"><span class="text-sm text-text-h font-medium capitalize">{{ formatEventType(event.eventTrigger) }}</span></td>
                            <td class="px-3 py-2 border-b border-border-default"><span class="text-sm text-text-m font-mono truncate block max-w-[100px]">{{ event.source }}</span></td>
                            <td class="px-3 py-2 border-b border-border-default"><span class="text-sm text-text-m font-mono truncate block max-w-[80px]">{{ formatSensorId(event.sensorId) }}</span></td>
                            <td class="px-3 py-2 border-b border-border-default text-right"><span class="text-sm text-text-m font-mono whitespace-nowrap">{{ event.time }}</span></td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</template>