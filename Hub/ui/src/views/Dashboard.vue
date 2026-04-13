<script setup>
    import { useSentinel } from '../api/useSentinel'
    import TrafficFilters from '../components/TrafficFilters.vue'
    import SeverityChart from '../components/SeverityChart.vue'
    import UptimeHeatmap from '../components/UptimeHeatmap.vue'
    import EventTable from '../components/EventTable.vue'
    import ThreatVelocity from '../components/ThreatVelocity.vue'

    const { 
        fleet, selectedSensor, events, uptimeData, activeTimeframe, 
        overallUptime, viewingArchive, archiveAll,
        activeEvent, isActiveSensorSilenced, archiveEvent, toggleSilence, forgetSensor, markEventRead
    } = useSentinel()

    const handleSelect = (id) => { selectedSensor.value = id }
    const handleToggle = (id) => { selectedSensor.value = selectedSensor.value === id ? null : id }
</script>

<template>
    <div class="flex flex-col gap-4 sm:gap-6 h-full max-w-[1600px] mx-auto w-full px-2 sm:px-4 lg:px-6">
        
        <TrafficFilters 
            :fleet="fleet" 
            :selectedSensor="selectedSensor" 
            @select-sensor="handleSelect" 
            @forget-sensor="forgetSensor"
            @toggle-silence="toggleSilence"
        />

        <div class="flex flex-wrap gap-4 sm:gap-6 shrink-0">
            
            <div class="flex-[1_1_350px] min-w-[100%] sm:min-w-[350px] h-[320px] lg:h-[340px] shrink-0">
                <ThreatVelocity :events="events" />
            </div>

            <div class="flex-[1_1_280px] min-w-[100%] sm:min-w-[280px] max-w-[450px] mx-auto h-[320px] lg:h-[340px] shrink-0">
                <SeverityChart :events="events" />
            </div>
            
            <div class="flex-[1.5_1_450px] min-w-[100%] lg:min-w-[450px] h-[320px] lg:h-[340px] shrink-0">
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

        <div class="flex-1 min-h-0 pb-6 mt-2">
            <EventTable 
                :events="events" 
                :viewingArchive="viewingArchive"
                @archive-all="archiveAll"
                @archive-event="archiveEvent"
                @mark-read="markEventRead"
            />
        </div>

    </div>
</template>