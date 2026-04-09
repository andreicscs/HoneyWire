<script setup>
    import { useSentinel } from '../api/useSentinel'
    import TrafficFilters from '../components/TrafficFilters.vue'
    import SeverityChart from '../components/SeverityChart.vue'
    import UptimeHeatmap from '../components/UptimeHeatmap.vue'
    import EventTable from '../components/EventTable.vue'

    // Bring in the state we need for this view
    const { 
        fleet, selectedSensor, filteredEvents, uptimeData, activeTimeframe, 
        overallUptime, viewingArchive, archiveAll,
        activeEvent, isActiveSensorSilenced, archiveEvent, toggleSilence, forgetSensor, markEventRead
    } = useSentinel()

    const handleSelect = (id) => { selectedSensor.value = id }
    const handleToggle = (id) => { selectedSensor.value = selectedSensor.value === id ? null : id }
</script>

<template>
    <div class="max-w-400 mx-auto space-y-6">
        
        <TrafficFilters 
            :fleet="fleet" 
            :selectedSensor="selectedSensor" 
            @select-sensor="handleSelect" 
            @forget-sensor="forgetSensor"
            @toggle-silence="toggleSilence"
        />

        <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">
            <div class="lg:col-span-4">
                <SeverityChart :events="filteredEvents" />
            </div>

            <div class="lg:col-span-8">
                <UptimeHeatmap 
                    :uptimeData="uptimeData"
                    :overallUptime="overallUptime"
                    :activeTimeframe="activeTimeframe"
                    :fleet="fleet"
                    :selectedSensor="selectedSensor"
                    @update:timeframe="t => activeTimeframe = t"
                    @select-sensor="handleToggle"
                />
            </div>
        </div>

        <EventTable 
            :events="filteredEvents" 
            :viewingArchive="viewingArchive" 
            @archive-all="archiveAll"
            @archive-event="archiveEvent"
            @open-event="evt => { activeEvent = evt; if(!evt.is_read) { evt.is_read = 1; fetch(`/api/v1/events/${evt.id}/read`, {method: 'PATCH'})} }"
            @mark-read="markEventRead"
        />
    </div>
</template>