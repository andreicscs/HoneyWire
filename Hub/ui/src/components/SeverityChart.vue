<script setup>
import { ref, onMounted, watch, onUnmounted } from 'vue'
import Chart from 'chart.js/auto'

const props = defineProps({
    events: { type: Array, required: true }
})

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

const updateChart = () => {
    if (!chartInstance) return;
    const counts = { critical: 0, high: 0, medium: 0, low: 0, info: 0 };
    props.events.forEach(e => {
        const s = e.severity ? e.severity.toLowerCase() : 'info';
        if (counts.hasOwnProperty(s)) counts[s]++;
    });
    
    const newData = ['critical', 'high', 'medium', 'low', 'info'].map(k => counts[k]);
    if (JSON.stringify(chartInstance.data.datasets[0].data) !== JSON.stringify(newData)) {
        chartInstance.data.datasets[0].data = newData;
        chartInstance.update();
    }
}

onMounted(() => {
    if (chartCanvas.value) {
        chartInstance = new Chart(chartCanvas.value, {
            type: 'doughnut',
            data: {
                labels: ['critical', 'high', 'medium', 'low', 'info'],
                datasets: [{
                    data: [0,0,0,0,0],
                    backgroundColor: ['#f43f5e', '#fb923c', '#eab308', '#3b82f6', '#64748b'],
                    borderWidth: 0, spacing: 4, borderRadius: 2
                }]
            },
            options: { 
                cutout: '82%', 
                responsive: true,
                maintainAspectRatio: false,
                animation: true,
                plugins: { legend: { display: false } } 
            }
        });
        updateChart();
    }

    themeObserver = new MutationObserver((mutations) => {
        mutations.forEach((mutation) => {
            if (mutation.attributeName === 'class' && chartInstance) {
                chartInstance.update(); 
            }
        });
    });
    themeObserver.observe(document.documentElement, { attributes: true });
})

watch(() => props.events, updateChart, { deep: true })
onUnmounted(() => {
    if (chartInstance) chartInstance.destroy()
    if (themeObserver) themeObserver.disconnect()
})
</script>

<template>
    <div class="bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 rounded-lg p-4 sm:p-5 flex flex-col shadow-sm h-full w-full overflow-hidden relative group">
        <div>
            <h3 class="text-sm font-semibold text-slate-800 dark:text-zinc-200">Severity Distribution</h3>
            <div class="flex items-center gap-4 mt-1">
                <p class="text-xs text-slate-500 dark:text-zinc-400">Active Threat Breakdown</p>
            </div>
        </div>
        
        <div class="flex-1 relative mt-2 min-h-0 w-full">
            <canvas ref="chartCanvas" class="w-full h-full"></canvas>
            <div class="absolute inset-0 flex flex-col items-center justify-center pointer-events-none mt-2">
                <span class="text-3xl font-bold transition-colors leading-none"
                      :class="events.length === 0 ? 'text-emerald-500 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-500'">
                    {{ events.length }}
                </span>
                <span class="text-xs font-medium text-slate-500 dark:text-zinc-400 mt-1 leading-none">Events</span>
            </div>
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