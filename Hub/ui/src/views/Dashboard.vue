<script setup>
    import { useSentinel } from '../api/useSentinel'
    import TrafficFilters from '../components/TrafficFilters.vue'
    import SeverityChart from '../components/SeverityChart.vue'
    import UptimeHeatmap from '../components/UptimeHeatmap.vue'
    import EventTable from '../components/EventTable.vue'
    import EventModal from '../components/EventModal.vue'

    // Bring in the state we need for this view
    const { 
        fleet, selectedSensor, filteredEvents, uptimeData, activeTimeframe, 
        overallUptime, viewingArchive, archiveAll,
        activeEvent, isActiveSensorSilenced, archiveEvent, toggleSilence 
    } = useSentinel()

    const handleSelect = (id) => { selectedSensor.value = id }
    const handleToggle = (id) => { selectedSensor.value = selectedSensor.value === id ? null : id }

</script>

<template>
    <div class="max-w-[1600px] mx-auto space-y-6">
        
        <TrafficFilters 
            :fleet="fleet" 
            :selectedSensor="selectedSensor" 
            @select-sensor="handleSelect" 
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
            @open-event="evt => { activeEvent = evt; if(!evt.is_read) { evt.is_read = 1; fetch(`/api/v1/events/${evt.id}/read`, {method: 'PATCH'})} }"
        />

        <transition enter-active-class="transition ease-out duration-150" enter-from-class="opacity-0 scale-95" enter-to-class="opacity-100 scale-100" leave-active-class="transition ease-in duration-100" leave-from-class="opacity-100 scale-100" leave-to-class="opacity-0 scale-95">
            <EventModal 
                v-if="activeEvent"
                :event="activeEvent"
                :isSilenced="isActiveSensorSilenced"
                :viewingArchive="viewingArchive"
                @close="activeEvent = null"
                @toggle-silence="toggleSilence"
                @archive-event="archiveEvent"
            />
        </transition>
    </div>
</template>