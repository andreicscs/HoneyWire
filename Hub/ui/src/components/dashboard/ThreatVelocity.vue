<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted, shallowRef, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import Chart from 'chart.js/auto'
import { useAppStore } from '../../stores/System/app'
import { useEventsStore } from '../../stores/Events/events'
import { useFleetStore } from '../../stores/Fleet/fleet'
import { getComputedRgb, injectAlpha } from '../../utils/theme'
import { baseTooltipConfig } from '../../utils/chartConfig'
import BaseTimeFilter from '../ui/forms/BaseTimeFilter.vue'
import BaseLegend from '../ui/feedback/BaseLegend.vue'
import BaseWidget from '../ui/layout/BaseWidget.vue'

const appStore = useAppStore()
const eventsStore = useEventsStore()
const fleetStore = useFleetStore()

const { velocityTimeframe, viewingArchive } = storeToRefs(appStore)
const { threatVelocityProjection: projection, isFetchingThreatVelocityProjection, lastVelocityInvalidation } = storeToRefs(eventsStore)
const { selectedNode, selectedSensor } = storeToRefs(fleetStore)

const chartCanvas = ref<HTMLCanvasElement | null>(null)
const chartInstance = shallowRef<any>(null)
let themeObserver: MutationObserver | null = null
let rolloverTimeout: any = null 
let isFirstRender = true

const severities = ['critical', 'high', 'medium', 'low', 'info']

const initChart = () => {
    if (!chartCanvas.value) return
    const ctx = chartCanvas.value.getContext('2d')
    if (!ctx) return
    chartInstance.value = new Chart(ctx, {
        type: 'line',
        data: { 
            labels: [], 
            datasets: severities.map(sev => ({ 
                label: sev.charAt(0).toUpperCase() + sev.slice(1), 
                data: [], fill: true, tension: 0.5, borderWidth: 1.5, 
                pointRadius: 0, pointHoverRadius: 4, borderJoinStyle: 'round' 
            })) 
        },
        options: {
            responsive: true, maintainAspectRatio: false,
            layout: { padding: { top: 15, left: 0, right: 0, bottom: 0 } },
            animation: { duration: 800, easing: 'easeOutQuart' }, 
            plugins: { 
                legend: { display: false }, 
                tooltip: { 
                    ...(baseTooltipConfig as any),
                    mode: 'index', 
                    intersect: false, 
                    callbacks: {
                        title: (context: any) => projection.value?.exactTimes?.[context[0].dataIndex] || '',
                        labelColor: (context: any) => {
                            return { borderColor: context.dataset.borderColor, backgroundColor: context.dataset.borderColor }
                        }
                    }
                } 
            },
            scales: {
                x: { grid: { display: false }, ticks: { maxRotation: 0, minRotation: 0, maxTicksLimit: 5, font: { size: 10, family: 'ui-monospace, monospace' }, align: 'inner' } },
                y: { 
                    display: false, 
                    beginAtZero: true,
                    grace: '15%'
                } 
            },
            interaction: { intersect: false, mode: 'index' }
        }
    })
}

const updateTheme = () => {
    if (!chartInstance.value || !chartCanvas.value) return
    
    const isDark = document.documentElement.classList.contains('dark')
    const ctx = chartCanvas.value.getContext('2d')
    if (!ctx) return
    const chartHeight = chartInstance.value.chartArea?.bottom || chartInstance.value.height || 200

    chartInstance.value.data.datasets.forEach((dataset: any, index: number) => {
        const sev = severities[index]
        const baseRgb = getComputedRgb(`--sev-${sev}`) 
        
        const gradient = ctx.createLinearGradient(0, 0, 0, chartHeight)
        gradient.addColorStop(0, injectAlpha(baseRgb, isDark ? 0.3 : 0.15))
        gradient.addColorStop(1, injectAlpha(baseRgb, 0))
        
        dataset.borderColor = baseRgb
        dataset.backgroundColor = gradient
        dataset.pointHoverBackgroundColor = baseRgb
    })

    const bgSurfaceRgb = getComputedRgb('--bg-surface')
    chartInstance.value.options.plugins.tooltip.backgroundColor = injectAlpha(bgSurfaceRgb, 0.95)
    
    chartInstance.value.options.plugins.tooltip.titleColor = getComputedRgb('--text-m')
    chartInstance.value.options.plugins.tooltip.bodyColor = getComputedRgb('--text-h')
    chartInstance.value.options.plugins.tooltip.borderColor = getComputedRgb('--border-default')
    chartInstance.value.options.scales.x.ticks.color = getComputedRgb('--text-m')

    chartInstance.value.update('none')
}

const updateData = () => {
    const p = projection.value;
    if (!chartInstance.value || !p) return;

    chartInstance.value.data.labels = p.labels;
    
    chartInstance.value.data.datasets.forEach((dataset: any, index: number) => {
        const sev = severities[index]
        const data = p.series[sev as keyof typeof p.series] || []
        dataset.data = data
        dataset.hidden = data.every((v: number) => v === 0)
    })

    if (isFirstRender) {
        chartInstance.value.update();
        isFirstRender = false;
    } else {
        chartInstance.value.update('none');
    }
}

const fetchContextualProjection = () => {
    eventsStore.fetchThreatVelocityProjection(
        velocityTimeframe.value,
        selectedNode.value?.id,
        selectedSensor.value?.sensorId,
        viewingArchive.value
    );
}

const scheduleNextRollover = () => {
    // Clear any existing timeout so they don't pile up
    if (rolloverTimeout) clearTimeout(rolloverTimeout)
    if (!projection.value?.bucketSizeMs) return

    const bucketMs = projection.value.bucketSizeMs
    const now = Date.now()
    
    // Find the exact millisecond of the next bucket boundary
    const nextBoundary = Math.ceil(now / bucketMs) * bucketMs
    
    // Add a tiny 100ms buffer to ensure we safely crossed the time boundary 
    // before asking the backend for the new data.
    const delay = nextBoundary - now + 100

    rolloverTimeout = setTimeout(() => {
        // When the boundary hits, request fresh data
        fetchContextualProjection()
        
        // Note: We don't recursively call scheduleNextRollover() here.
        // Why? Because fetchContextualProjection() will fetch a new projection,
        // which will trigger the watcher below, which will safely schedule the NEXT tick.
    }, delay)
}

onMounted(async () => {
    await nextTick()
    if (chartCanvas.value) {
        initChart()
        updateTheme()
        if (projection.value?.generatedAt) {
            updateData() // Draw immediately if data arrived before mount
        }
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
    themeObserver.observe(document.documentElement, { attributes: true })
})

// TRIGGER 1: Context Changes (User clicks filters)
watch(
    [velocityTimeframe, () => selectedNode.value?.id, () => selectedSensor.value?.sensorId, viewingArchive],
    fetchContextualProjection,
    { immediate: true }
)

// TRIGGER 2: Global Invalidation (WebSocket says data is stale)
watch(
    lastVelocityInvalidation,
    () => {
        if (!viewingArchive.value) {
            fetchContextualProjection();
        }
    }
)

// TRIGGER 3: Snapshot Arrives (Refetch completed)
watch(
    () => projection.value?.generatedAt,
    (newVal) => {
        if (newVal) {
            updateData();
            scheduleNextRollover(); // Predict and schedule the next rollover perfectly
        }
    }
)

onUnmounted(() => {
    if (chartInstance.value) chartInstance.value.destroy()
    if (themeObserver) themeObserver.disconnect()
    if (rolloverTimeout) clearTimeout(rolloverTimeout)
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
            <div class="flex justify-between items-start h-14 relative z-10 shrink-0 w-full">
                <div>
                    <h3 class="text-base font-medium text-text-h">Events Velocity</h3>
                    <div class="flex items-center gap-2 mt-1 leading-none">
                        <span class="text-sm transition-colors" :class="(projection?.recentEventCount || 0) > 0 ? 'text-critical' : 'text-success-main'">
                            {{ projection?.recentEventCount || 0 }}
                        </span>
                        <span class="text-sm text-text-m">Events Recorded</span>
                    </div>
                </div>
                
                <BaseTimeFilter 
                    :model-value="velocityTimeframe" 
                    @update:model-value="appStore.setVelocityTimeframe($event)" 
                />
            </div>
        </template>

        <div class="flex-1 relative mt-2 min-h-0 w-full">
            <div v-if="(!projection || projection.recentEventCount === 0) && !isFetchingThreatVelocityProjection" class="absolute inset-0 flex items-center justify-center text-center text-base text-text-m z-20">
                Awaiting telemetry...
            </div>
            <canvas ref="chartCanvas" class="relative z-0"></canvas>
        </div>

        <template #footer>
            <BaseLegend :items="legendItems" />
        </template>
    </BaseWidget>
</template>