<script setup>
import { ref, onMounted, watch, onUnmounted, shallowRef, nextTick, toRaw } from 'vue'
import { storeToRefs } from 'pinia'
import Chart from 'chart.js/auto'
import { useAppStore } from '../../stores/app'
import { useEventsStore } from '../../stores/events'
import { getComputedRgb, injectAlpha } from '../../utils/theme'
import { baseTooltipConfig, applyChartTheme } from '../../utils/chartConfig'
import BaseTimeFilter from '../ui/forms/BaseTimeFilter.vue'
import BaseLegend from '../ui/feedback/BaseLegend.vue'
import BaseWidget from '../ui/layout/BaseWidget.vue'

const appStore = useAppStore()
const eventsStore = useEventsStore()

const { velocityTimeframe } = storeToRefs(appStore)
const { filteredEvents: events } = storeToRefs(eventsStore)

const chartCanvas = ref(null)
const recentEventCount = ref(0)
let chartInstance = shallowRef(null)
let themeObserver = null
let liveTicker = null 

const severities = ['critical', 'high', 'medium', 'low', 'info']

const getSeverityRgb = (sev) => {
    const hex = getCssVariable(`--sev-${sev}`);
    return hexToRgb(hex);
}

let exactTimesList = [] 

const initChart = () => {
    const ctx = chartCanvas.value.getContext('2d')
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
            animation: { duration: 0 }, 
            plugins: { 
                legend: { display: false }, 
                tooltip: { 
                    ...baseTooltipConfig,
                    mode: 'index', 
                    intersect: false, 
                    callbacks: {
                        title: (context) => exactTimesList[context[0].dataIndex],
                        labelColor: (context) => {
                            return { borderColor: context.dataset.borderColor, backgroundColor: context.dataset.borderColor }
                        }
                    }
                } 
            },
            scales: {
                x: { grid: { display: false, drawBorder: false }, ticks: { maxRotation: 0, minRotation: 0, maxTicksLimit: 5, font: { size: 10, family: 'ui-monospace, monospace' }, align: 'inner' } },
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

const updateData = () => {
    if (!chartInstance.value) return

    const now = new Date()
    let buckets = 30
    let bucketSizeMs = 120000

    if (velocityTimeframe.value === '24H') { buckets = 24; bucketSizeMs = 3600000 } 
    else if (velocityTimeframe.value === '7D') { buckets = 14; bucketSizeMs = 43200000 } 
    else if (velocityTimeframe.value === '30D') { buckets = 30; bucketSizeMs = 86400000 }

    const labels = new Array(buckets).fill('')
    exactTimesList = new Array(buckets).fill('')
    
    for (let i = 0; i < buckets; i++) {
        const stepsAgo = buckets - 1 - i
        const d = new Date(now.getTime() - stepsAgo * bucketSizeMs)
        exactTimesList[i] = d.toLocaleTimeString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
        
        if (stepsAgo === 0) labels[i] = 'Now'
        else {
            if (velocityTimeframe.value === '1H') labels[i] = `-${stepsAgo * 2}m`
            else if (velocityTimeframe.value === '24H') labels[i] = `-${stepsAgo}h`
            else if (velocityTimeframe.value === '7D') labels[i] = `-${stepsAgo * 12}h`
            else if (velocityTimeframe.value === '30D') labels[i] = `-${stepsAgo}d`
        }
    }

    const data = { critical: new Array(buckets).fill(0), high: new Array(buckets).fill(0), medium: new Array(buckets).fill(0), low: new Array(buckets).fill(0), info: new Array(buckets).fill(0) }
    let count = 0

    const rawEvents = toRaw(events.value)

    rawEvents.forEach(e => {
        if (!e.timestamp) return
        const eTime = new Date(e.timestamp)
        const diffMins = Math.floor((now - eTime) / bucketSizeMs)
        
        if (diffMins >= 0 && diffMins < buckets) {
            const sev = e.severity ? e.severity.toLowerCase() : 'info'
            if (data[sev]) data[sev][buckets - 1 - diffMins]++
            count++
        }
    })
    recentEventCount.value = count

    chartInstance.value.data.labels = labels
    chartInstance.value.data.datasets.forEach((dataset, index) => {
        const sev = severities[index]
        dataset.data = data[sev]
        dataset.hidden = data[sev].every(v => v === 0)
    })

    chartInstance.value.update('none') 
}

const updateTheme = () => {
    if (!chartInstance.value || !chartCanvas.value) return
    
    const isDark = document.documentElement.classList.contains('dark')
    const ctx = chartCanvas.value.getContext('2d')
    const chartHeight = chartInstance.value.chartArea?.bottom || chartInstance.value.height || 200

    chartInstance.value.data.datasets.forEach((dataset, index) => {
        const sev = severities[index]
        // This now safely returns 'rgb(x, y, z)' computed perfectly by the browser
        const baseRgb = getComputedRgb(`--sev-${sev}`) 
        
        const gradient = ctx.createLinearGradient(0, 0, 0, chartHeight)
        gradient.addColorStop(0, injectAlpha(baseRgb, isDark ? 0.3 : 0.15))
        gradient.addColorStop(1, injectAlpha(baseRgb, 0))
        
        dataset.borderColor = baseRgb
        dataset.backgroundColor = gradient
        dataset.pointHoverBackgroundColor = baseRgb
    })

    // Update Tooltips securely with browser-computed RGBs
    const bgSurfaceRgb = getComputedRgb('--bg-surface')
    chartInstance.value.options.plugins.tooltip.backgroundColor = injectAlpha(bgSurfaceRgb, 0.95)
    
    chartInstance.value.options.plugins.tooltip.titleColor = getComputedRgb('--text-m')
    chartInstance.value.options.plugins.tooltip.bodyColor = getComputedRgb('--text-h')
    chartInstance.value.options.plugins.tooltip.borderColor = getComputedRgb('--border-default')
    chartInstance.value.options.scales.x.ticks.color = getComputedRgb('--text-m')

    chartInstance.value.update('none')
}

onMounted(async () => {
    await nextTick()
    if (chartCanvas.value) {
        initChart()
        updateTheme()
        updateData()
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

    liveTicker = setInterval(() => { updateData() }, 2000)
})

watch([() => events.value[0]?.id, velocityTimeframe, () => events.value.length], () => {
    updateData()
})

onUnmounted(() => {
    if (chartInstance.value) chartInstance.value.destroy()
    if (themeObserver) themeObserver.disconnect()
    if (liveTicker) clearInterval(liveTicker)
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
                        <span class="text-sm" :class="recentEventCount > 0 ? 'text-critical' : 'text-success-main'">{{ recentEventCount }}</span>
                        <span class="text-sm text-text-m">Events Recorded</span>
                    </div>
                </div>
                
                <BaseTimeFilter v-model="appStore.velocityTimeframe" />
            </div>
        </template>

        <div class="flex-1 relative mt-2 min-h-0 w-full">
            <div v-if="recentEventCount === 0" class="absolute inset-0 flex items-center justify-center text-sm text-text-m z-20">
                Awaiting telemetry...
            </div>
            <canvas ref="chartCanvas" class="w-full h-full"></canvas>
        </div>

        <template #footer>
            <BaseLegend :items="legendItems" />
        </template>
    </BaseWidget>
</template>
emplate>
