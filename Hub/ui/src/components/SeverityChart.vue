<script setup>
import { ref, onMounted, watch, onUnmounted, toRaw } from 'vue'
import { storeToRefs } from 'pinia'
import Chart from 'chart.js/auto'
import { useEventsStore } from '../stores/events'
import { getCssVariable, hexToRgb } from '../utils/theme'

const eventsStore = useEventsStore()
const { filteredEvents: events } = storeToRefs(eventsStore)

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
    getCssVariable('--sev-critical') || '#f43f5e',
    getCssVariable('--sev-high') || '#fb923c',
    getCssVariable('--sev-medium') || '#eab308',
    getCssVariable('--sev-low') || '#3b82f6',
    getCssVariable('--sev-info') || '#64748b'
]

const updateData = () => {
    if (!chartInstance) return;
    const counts = { critical: 0, high: 0, medium: 0, low: 0, info: 0 };
    
    const rawEvents = toRaw(events.value);
    
    rawEvents.forEach(e => {
        const s = e.severity ? e.severity.toLowerCase() : 'info';
        if (counts.hasOwnProperty(s)) counts[s]++;
    });
    
    const newData = ['critical', 'high', 'medium', 'low', 'info'].map(k => counts[k]);
    const currentData = chartInstance.data.datasets[0].data;
    const hasChanged = newData.some((val, i) => val !== currentData[i]);

    if (hasChanged) {
        chartInstance.data.datasets[0].data = newData;
        chartInstance.update(); 
    }
}

const updateTheme = () => {
    if (!chartInstance) return;
    
    const isDark = document.documentElement.classList.contains('dark')

    // FIXED: Dynamically pull tooltip colors from CSS variables
    const bgHex = getCssVariable('--bg-surface') || (isDark ? '#18181b' : '#ffffff');
    const bgRgb = hexToRgb(bgHex);
    const bgRgbStr = typeof bgRgb === 'object' ? `${bgRgb.r}, ${bgRgb.g}, ${bgRgb.b}` : (bgRgb || (isDark ? '24, 24, 27' : '255, 255, 255'));

    chartInstance.options.plugins.tooltip.backgroundColor = `rgba(${bgRgbStr}, 0.95)`
    chartInstance.options.plugins.tooltip.titleColor = getCssVariable('--text-muted') || (isDark ? '#a1a1aa' : '#64748b')
    chartInstance.options.plugins.tooltip.bodyColor = getCssVariable('--text-main') || (isDark ? '#f4f4f5' : '#0f172a')
    chartInstance.options.plugins.tooltip.borderColor = getCssVariable('--border-default') || (isDark ? '#3f3f46' : '#e2e8f0')
    
    chartInstance.data.datasets[0].backgroundColor = getChartColors();

    chartInstance.update('none'); 
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
                animation: true,
                plugins: { 
                    legend: { display: false },
                    tooltip: { 
                        borderWidth: 1, padding: 10, boxPadding: 4, 
                        usePointStyle: true, boxWidth: 8, boxHeight: 8, 
                        titleFont: { size: 11, family: 'ui-monospace, monospace', weight: 'normal' }, 
                        bodyFont: { size: 12, weight: 'bold' },
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

watch([() => events.value.length, () => events.value[0]?.id], updateData)

onUnmounted(() => {
    if (chartInstance) chartInstance.destroy()
    if (themeObserver) themeObserver.disconnect()
})
</script>

<template>
    <div class="bg-bg-surface border border-border-default rounded-lg p-4 sm:p-5 flex flex-col shadow-sm h-full w-full overflow-hidden relative group">
        <div>
            <h3 class="text-sm font-semibold text-text-main">Severity Distribution</h3>
            <div class="flex items-center gap-4 mt-1">
                <p class="text-xs text-text-muted">Active Threat Breakdown</p>
            </div>
        </div>
        
        <div class="flex-1 relative mt-2 min-h-0 w-full">
            <canvas ref="chartCanvas" class="w-full h-full"></canvas>
            <div class="absolute inset-0 flex flex-col items-center justify-center pointer-events-none mt-2">
                <span class="text-3xl font-bold transition-colors leading-none"
                      :class="events.length === 0 ? 'text-success-main' : 'text-critical'">
                    {{ events.length }}
                </span>
                <span class="text-xs font-medium text-text-muted mt-1 leading-none">Events</span>
            </div>
        </div>

        <div class="mt-auto h-4 pt-5 flex items-center justify-center gap-3 sm:gap-4 text-[8px] font-semibold text-text-muted uppercase tracking-wider shrink-0 border-t border-transparent">
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-critical"></span>Crit</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-high"></span>High</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-medium"></span>Med</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-low"></span>Low</div>
            <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-info"></span>Info</div>
        </div>

    </div>
</template>