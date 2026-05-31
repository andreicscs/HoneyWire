<script setup>
import TrafficFilters from '../components/dashboard/TrafficFilters.vue'
import SeverityChart from '../components/dashboard/SeverityChart.vue'
import UptimeHeatmap from '../components/dashboard/UptimeHeatmap.vue'
import EventTable from '../components/dashboard/EventTable.vue'
import ThreatVelocity from '../components/dashboard/ThreatVelocity.vue'
</script>

<template>
    <div class="min-h-full flex flex-col gap-4 sm:gap-6 max-w-[1600px] mx-auto w-full px-2 sm:px-4 lg:px-6 pb-4 sm:pb-6">
        <TrafficFilters />

        <div class="flex flex-wrap gap-4 sm:gap-6 shrink-0">
            <div class="flex-[1_1_350px] min-w-[100%] sm:min-w-[350px] h-[320px] lg:h-[340px] shrink-0">
                <ThreatVelocity />
            </div>

            <div class="flex-[1_1_280px] min-w-[100%] sm:min-w-[280px] max-w-[450px] mx-auto h-[320px] lg:h-[340px] shrink-0">
                <SeverityChart />
            </div>
            
            <div class="flex-[1.5_1_450px] min-w-[100%] lg:min-w-[450px] h-[320px] lg:h-[340px] shrink-0">
                <UptimeHeatmap />
            </div>
        </div>

        <div class="flex-1 min-h-0 mt-2">
            <EventTable />
        </div>
    </div>
</template>