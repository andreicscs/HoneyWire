<script setup>
import { ref, onMounted, watch, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import Chart from 'chart.js/auto'
import { useEventsStore } from '../../stores/Events/events'
import { useFleetStore } from '../../stores/Fleet/fleet'
import { useAppStore } from '../../stores/System/app'
import { getComputedRgb } from '../../utils/theme'
import { baseTooltipConfig, applyChartTheme } from '../../utils/chartConfig'
import BaseLegend from '../ui/feedback/BaseLegend.vue'
import BaseWidget from '../ui/layout/BaseWidget.vue'

const eventsStore = useEventsStore()
const fleetStore = useFleetStore()
const appStore = useAppStore()

const { severityProjection } = storeToRefs(eventsStore)
const { selectedNode, selectedSensor } = storeToRefs(fleetStore)

const chartCanvas = ref(null)
let chartInstance = null
let themeObserver = null

const neonGlowPlugin = {
    id: 'neonGlow',
    beforeDatasetsDraw(chart) {
        if (!document.documentElement.classList.contains('dark')) return;
        const ctx = chart.ctx;
        const meta = chart.getDatasetMeta(0);
        ctx.save();
        meta.data.forEach(arc => {
            ctx.shadowColor = arc.options.backgroundColor;
            ctx.shadowBlur = 5;
            ctx.shadowOffsetX = 0;
            ctx.shadowOffsetY = 0;
            arc.draw(ctx); 
        });
        ctx.restore();
    }
}

Chart.register(neonGlowPlugin)

const getChartColors = () => [
    getComputedRgb('--sev-critical') || 'rgb(244, 63, 94)',
    getComputedRgb('--sev-high') || 'rgb(251, 146, 60)',
    getComputedRgb('--sev-medium') || 'rgb(234, 179, 8)',
    getComputedRgb('--sev-low') || 'rgb(59, 130, 246)',
    getComputedRgb('--sev-info') || 'rgb(100, 116, 139)'
]

const updateData = () => {
    const p = severityProjection.value;
    if (!chartInstance || !p) return;
    
    const newData = [p.critical, p.high, p.medium, p.low, p.info];
    const currentData = chartInstance.data.datasets[0].data;
    
    const hasChanged = newData.some((val, i) => val !== currentData[i]);

    if (hasChanged) {
        chartInstance.data.datasets[0].data = newData;
        chartInstance.update(); 
    }
}

const updateTheme = () => {
    if (!chartInstance) return;
    
    // Re-grab the colors in case the theme switched to dark mode
    chartInstance.data.datasets[0].backgroundColor = getChartColors();
    
    applyChartTheme(chartInstance);
}

onMounted(() => {
    if (chartCanvas.value) {
        chartInstance = new Chart(chartCanvas.value, {
            type: 'doughnut',
            data: {
                labels: ['critical', 'high', 'medium', 'low', 'info'],
                datasets: [{
                    data: [0,0,0,0,0],
                    backgroundColor: getChartColors(),
                    borderWidth: 0, spacing: 4, borderRadius: 2
                }]
            },
            options: { 
                cutout: '82%', 
                responsive: true,
                maintainAspectRatio: false,
                layout: {
                    padding: { bottom: 5 }
                },
                animation: true,
                plugins: { 
                    legend: { display: false },
                    tooltip: { 
                        ...baseTooltipConfig,
                        callbacks: {
                            labelColor: (context) => {
                                const color = context.dataset.backgroundColor[context.dataIndex];
                                return { borderColor: color, backgroundColor: color }
                            }
                        }
                    }
                } 
            }
        });
        updateTheme();
        updateData();
    }

    themeObserver = new MutationObserver((mutations) => {
        let themeToggled = false
        mutations.forEach((m) => { if (m.attributeName === 'class') themeToggled = true })
        
        if (themeToggled) {
            setTimeout(() => {
                updateTheme()
            }, 50)
        }
    })
    themeObserver.observe(document.documentElement, { attributes: true });
})

watch(
    () => [selectedNode.value, selectedSensor.value, appStore.viewingArchive],
    ([node, sensor]) => {
        eventsStore.fetchSeverityProjection('alltime', node?.id, sensor?.sensorId)
    },
    { immediate: true }
)

watch(
    () => severityProjection.value,
    updateData
)

onUnmounted(() => {
    if (chartInstance) chartInstance.destroy()
    if (themeObserver) themeObserver.disconnect()
})

const legendItems = [
    { label: 'Crit', colorClass: 'bg-critical' },
    { label: 'High', colorClass: 'bg-high' },
    { label: 'Med', colorClass: 'bg-medium' },
    { label: 'Low', colorClass: 'bg-low' },
    { label: 'Info', colorClass: 'bg-info' }
]
</script>

<template>
    <BaseWidget>
        <template #header>
            <div>
                <h3 class="text-base font-medium text-text-h">Severity Distribution</h3>
                <div class="flex items-center gap-4 mt-1">
                    <p class="text-sm text-text-m">Active Threat Breakdown</p>
                </div>
            </div>
        </template>
        
        <div class="flex-1 relative mt-2 min-h-0 w-full">
            <canvas ref="chartCanvas" class="w-full h-full"></canvas>
            <div class="absolute inset-0 flex flex-col items-center justify-center pointer-events-none mt-2">
                <span class="text-3xl transition-colors leading-none"
                      :class="(severityProjection?.total || 0) === 0 ? 'text-success-main' : 'text-critical'">
                    {{ severityProjection?.total || 0 }}
                </span>
                <span class="text-sm text-text-m mt-1 leading-none">Events</span>
            </div>
        </div>

        <template #footer>
            <BaseLegend :items="legendItems" />
        </template>
    </BaseWidget>
</template>
