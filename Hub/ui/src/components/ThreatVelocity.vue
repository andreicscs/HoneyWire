<script setup>
import { ref, onMounted, watch, onUnmounted, shallowRef, nextTick, toRaw } from 'vue'
import Chart from 'chart.js/auto'

const props = defineProps({
    events: { type: Array, required: true }
})

const chartCanvas = ref(null)
const recentEventCount = ref(0)
const activeTimeframe = ref('24H')
let chartInstance = shallowRef(null)
let themeObserver = null
let liveTicker = null 

const severities = ['critical', 'high', 'medium', 'low', 'info']

const solidColors = { 
    critical: '244, 63, 94',  // #f43f5e
    high: '251, 146, 60',     // #fb923c
    medium: '234, 179, 8',    // #eab308
    low: '59, 130, 246',      // #3b82f6
    info: '100, 116, 139'     // #64748b
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
            layout: { padding: { top: 10, left: -5, right: -5, bottom: 0 } },
            animation: { duration: 0 }, 
            plugins: { 
                legend: { display: false }, 
                tooltip: { 
                    mode: 'index', intersect: false, 
                    borderWidth: 1, padding: 10, boxPadding: 4, 
                    usePointStyle: true, boxWidth: 8, boxHeight: 8, 
                    titleFont: { size: 11, family: 'ui-monospace, monospace', weight: 'normal' }, 
                    bodyFont: { size: 12, weight: 'bold' },
                    callbacks: {
                        title: (context) => exactTimesList[context[0].dataIndex],
                        labelColor: (context) => {
                            const sev = severities[context.datasetIndex]
                            return { borderColor: `rgb(${solidColors[sev]})`, backgroundColor: `rgb(${solidColors[sev]})` }
                        }
                    }
                } 
            },
            scales: {
                x: { grid: { display: false, drawBorder: false }, ticks: { maxRotation: 0, minRotation: 0, maxTicksLimit: 5, font: { size: 10, family: 'ui-monospace, monospace' }, align: 'inner' } },
                y: { display: false, beginAtZero: true } 
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

    if (activeTimeframe.value === '24H') { buckets = 24; bucketSizeMs = 3600000 } 
    else if (activeTimeframe.value === '7D') { buckets = 14; bucketSizeMs = 43200000 } 
    else if (activeTimeframe.value === '30D') { buckets = 30; bucketSizeMs = 86400000 }

    const labels = new Array(buckets).fill('')
    exactTimesList = new Array(buckets).fill('')
    
    for (let i = 0; i < buckets; i++) {
        const stepsAgo = buckets - 1 - i
        const d = new Date(now.getTime() - stepsAgo * bucketSizeMs)
        exactTimesList[i] = d.toLocaleTimeString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
        
        if (stepsAgo === 0) labels[i] = 'Now'
        else {
            if (activeTimeframe.value === '1H') labels[i] = `-${stepsAgo * 2}m`
            else if (activeTimeframe.value === '24H') labels[i] = `-${stepsAgo}h`
            else if (activeTimeframe.value === '7D') labels[i] = `-${stepsAgo * 12}h`
            else if (activeTimeframe.value === '30D') labels[i] = `-${stepsAgo}d`
        }
    }

    const data = { critical: new Array(buckets).fill(0), high: new Array(buckets).fill(0), medium: new Array(buckets).fill(0), low: new Array(buckets).fill(0), info: new Array(buckets).fill(0) }
    let count = 0

    // FIX 1: Strip Vue Reactivity for 10x faster looping
    const rawEvents = toRaw(props.events)

    rawEvents.forEach(e => {
        if (!e.timestamp) return
        const eTime = new Date(e.timestamp.replace(' ', 'T') + 'Z')
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
    
    // FIX 2: Read height from Chart.js memory, NOT the DOM. Kills Layout Thrashing!
    const chartHeight = chartInstance.value.chartArea?.bottom || chartInstance.value.height || 200

    chartInstance.value.data.datasets.forEach((dataset, index) => {
        const sev = severities[index]
        const gradient = ctx.createLinearGradient(0, 0, 0, chartHeight)
        gradient.addColorStop(0, `rgba(${solidColors[sev]}, ${isDark ? '0.3' : '0.15'})`)
        gradient.addColorStop(1, `rgba(${solidColors[sev]}, 0)`)
        
        dataset.borderColor = `rgb(${solidColors[sev]})`
        dataset.backgroundColor = gradient
        dataset.pointHoverBackgroundColor = `rgb(${solidColors[sev]})`
    })

    chartInstance.value.options.plugins.tooltip.backgroundColor = isDark ? 'rgba(24, 24, 27, 0.95)' : 'rgba(255, 255, 255, 0.95)'
    chartInstance.value.options.plugins.tooltip.titleColor = isDark ? '#a1a1aa' : '#64748b'
    chartInstance.value.options.plugins.tooltip.bodyColor = isDark ? '#f4f4f5' : '#0f172a'
    chartInstance.value.options.plugins.tooltip.borderColor = isDark ? '#3f3f46' : '#e2e8f0'
    chartInstance.value.options.scales.x.ticks.color = isDark ? '#52525b' : '#94a3b8'

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

watch([() => props.events[0]?.id, activeTimeframe, () => props.events.length], () => {
    updateData()
})

onUnmounted(() => {
    if (chartInstance.value) chartInstance.value.destroy()
    if (themeObserver) themeObserver.disconnect()
    if (liveTicker) clearInterval(liveTicker)
})
</script>

<template>
    <div class="bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 rounded-lg p-4 sm:p-5 flex flex-col shadow-sm h-full w-full overflow-hidden relative group">
        
        <div class="flex justify-between items-start h-14 relative z-10 shrink-0 w-full">

            <div>
                <h3 class="text-sm font-semibold text-slate-800 dark:text-zinc-200">Events velocity</h3>
                <div class="flex items-center gap-2 mt-1 leading-none">
                    <span class="text-xs font-bold" :class="recentEventCount > 0 ? 'text-rose-500 dark:text-rose-400'  : 'text-emerald-500'">{{ recentEventCount }}</span>
                    <span class="text-xs font-medium text-slate-500 dark:text-zinc-400">Events Recorded</span>
                </div>
            </div>
            
            <div class="flex bg-slate-50 border border-slate-100 dark:border-transparent dark:bg-zinc-800 p-0.5 rounded-md text-[11px] font-medium text-slate-500 dark:text-zinc-400">
                <button v-for="time in ['1H', '24H', '7D', '30D']" :key="time"
                        @click="activeTimeframe = time"
                        class="px-2.5 py-1 rounded transition-colors"
                        :class="activeTimeframe === time ? 'bg-white dark:bg-zinc-700 text-slate-800 dark:text-zinc-100 shadow-sm border border-slate-200 dark:border-transparent' : 'hover:text-slate-700 dark:hover:text-zinc-200'">
                    {{ time }}
                </button>
            </div>
        </div>

        <div class="flex-1 relative mt-2 min-h-0 w-full -mx-2">
            <div v-if="events.length === 0" class="absolute inset-0 flex items-center justify-center text-xs text-slate-400 dark:text-zinc-500 z-20">
                Awaiting telemetry...
            </div>
            <canvas ref="chartCanvas" class="w-full h-full"></canvas>
        </div>

        <div class="mt-auto h-4 pt-5 flex items-center justify-center gap-3 sm:gap-4 text-[8px] font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wider shrink-0 border-t border-transparent">
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-[#f43f5e]"></span>Crit</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-[#fb923c]"></span>High</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-[#eab308]"></span>Med</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-[#3b82f6]"></span>Low</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-[#64748b]"></span>Info</div>
        </div>

    </div>
</template>