<script setup lang="ts">
import TrafficFilters from '../components/dashboard/TrafficFilters.vue'
import SeverityChart from '../components/dashboard/SeverityChart.vue'
import UptimeHeatmap from '../components/dashboard/UptimeHeatmap.vue'
import EventTable from '../components/dashboard/EventTable.vue'
import ThreatVelocity from '../components/dashboard/ThreatVelocity.vue'
import { dashboardStyles } from './dashboardStyles.ts'
</script>

<template>
    <div :class="dashboardStyles.page" data-testid="dashboard-page">
        <TrafficFilters />

        <div :class="dashboardStyles.chartsRow" data-testid="dashboard-charts">
            <div :class="dashboardStyles.chartVelocity" data-testid="chart-velocity">
                <ThreatVelocity />
            </div>

            <div :class="dashboardStyles.chartSeverity" data-testid="chart-severity">
                <SeverityChart />
            </div>

            <div :class="dashboardStyles.chartUptime" data-testid="chart-uptime">
                <UptimeHeatmap />
            </div>
        </div>

        <div :class="dashboardStyles.eventsWrap" data-testid="dashboard-events">
            <EventTable />
        </div>
    </div>
</template>
