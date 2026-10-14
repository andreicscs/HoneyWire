<script setup lang="ts">
import { computed } from 'vue'
import { formatSensorId } from '../../utils/formatSensorId'
import { formatEventType } from '../../utils/formatEventType'
import { useEventsStore } from '../../stores/Events/events'
import { nodeWidgetsStyles } from './nodedetailsStyles.ts'

const props = defineProps<{
    node: any,
    recentActivity: any[]
}>()

const emit = defineEmits<{
    (e: 'viewAllEvents'): void
}>()

const eventsStore = useEventsStore()

const maxSensorEvents = computed(() => {
  const summary = eventsStore.summaryProjection
  if (!summary || !summary.bySensor) return 1
  const values = Object.values(summary.bySensor)
  if (values.length === 0) return 1
  return Math.max(...values, 1)
})

const topSensors = computed(() => {
  const summary = eventsStore.summaryProjection
  return [...(props.node?.installedSensors || [])]
    .map((s: any) => ({ ...s, events24h: summary?.bySensor?.[s.sensorId] || 0 }))
    .sort((a: any, b: any) => (b.events24h || 0) - (a.events24h || 0))
})
</script>

<template>
    <div :class="nodeWidgetsStyles.grid">
        <!-- Event Volume (24h) -->
        <div :class="nodeWidgetsStyles.volumeCard">
            <h3 class="text-base font-semibold text-text-h mb-4">Event Volume (24h)</h3>
            <div v-if="topSensors.length > 0" :class="nodeWidgetsStyles.volumeList">
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
            <div v-else :class="nodeWidgetsStyles.emptyText">No events recorded.</div>
        </div>

        <!-- Recent Activity - mini event table -->
        <div :class="nodeWidgetsStyles.activityCard">
            <div :class="nodeWidgetsStyles.activityHead">
                <h3 class="text-base font-semibold text-text-h">Recent Activity</h3>
                <button @click="$emit('viewAllEvents')" :class="nodeWidgetsStyles.viewAllButton">View All &rarr;</button>
            </div>
            
            <div :class="nodeWidgetsStyles.tableScroll">
                <table class="w-full text-left border-collapse">
                    <thead :class="nodeWidgetsStyles.tableHead">
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
                        <tr v-for="event in recentActivity" :key="event.id" :class="[nodeWidgetsStyles.activityRow, 'bleed-' + event.severity.toLowerCase()]">
                            <td :class="nodeWidgetsStyles.tableCell"><span class="px-1.5 py-0.5 rounded border text-sm font-medium bg-bg-base whitespace-nowrap capitalize" :style="{ borderColor: `var(--color-${event.severity.toLowerCase()})`, color: `var(--color-${event.severity.toLowerCase()})` }">{{ event.severity }}</span></td>
                            <td :class="nodeWidgetsStyles.tableCell"><span class="text-sm text-text-h font-medium capitalize">{{ formatEventType(event.eventTrigger) }}</span></td>
                            <td :class="nodeWidgetsStyles.tableCell"><span class="text-sm text-text-m font-mono truncate block max-w-[100px]">{{ event.source }}</span></td>
                            <td :class="nodeWidgetsStyles.tableCell"><span class="text-sm text-text-m font-mono truncate block max-w-[80px]">{{ formatSensorId(event.sensorId) }}</span></td>
                            <td :class="nodeWidgetsStyles.tableCell" class="text-right"><span class="text-sm text-text-m font-mono whitespace-nowrap">{{ event.time }}</span></td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</template>