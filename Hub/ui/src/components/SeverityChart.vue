<script setup>
import { ref, onMounted, watch, onUnmounted } from 'vue'
import Chart from 'chart.js/auto'

const props = defineProps({
    events: { type: Array, required: true }
})

// Vue reference to the actual <canvas> element in the template
const chartCanvas = ref(null)
let chartInstance = null

// --- Chart.js Custom Neon Glow Plugin (From Monolith) ---
const neonGlowPlugin = {
    id: 'neonGlow',
    beforeDatasetsDraw(chart) {
        if (!document.documentElement.classList.contains('dark')) return;
        const ctx = chart.ctx;
        const meta = chart.getDatasetMeta(0);
        ctx.save();
        meta.data.forEach(arc => {
            ctx.shadowColor = arc.options.backgroundColor;
            ctx.shadowBlur = 8;
            ctx.shadowOffsetX = 0;
            ctx.shadowOffsetY = 0;
            arc.draw(ctx); 
        });
        ctx.restore();
    }
}

Chart.register(neonGlowPlugin)

// --- Logic to Update the Chart Data ---
const updateChart = () => {
    if (!chartInstance) return;
    
    const counts = { critical: 0, high: 0, medium: 0, low: 0, info: 0 };
    props.events.forEach(e => {
        const s = e.severity ? e.severity.toLowerCase() : 'info';
        if (counts.hasOwnProperty(s)) counts[s]++;
    });
    
    const newData = ['critical', 'high', 'medium', 'low', 'info'].map(k => counts[k]);
    
    // Only trigger a chart re-render if the actual numbers changed
    if (JSON.stringify(chartInstance.data.datasets[0].data) !== JSON.stringify(newData)) {
        chartInstance.data.datasets[0].data = newData;
        chartInstance.update();
    }
}

// --- Lifecycle Hooks ---
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
        // Initial populate
        updateChart();
    }
})

// Watch the events prop for changes and update chart automatically
watch(() => props.events, updateChart, { deep: true })

// Clean up memory when changing views
onUnmounted(() => {
    if (chartInstance) chartInstance.destroy()
})
</script>

<template>
    <div class="bg-slate-50 dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 rounded-lg p-5 flex flex-col backdrop-blur-sm h-full w-full">
        <h3 class="text-sm font-semibold mb-4 text-slate-800 dark:text-zinc-200">Severity Distribution</h3>
        
        <div class="flex-1 relative min-h-[220px]">
            <canvas ref="chartCanvas"></canvas>
            
            <div class="absolute inset-0 flex flex-col items-center justify-center pointer-events-none mt-2">
                <span class="text-3xl font-bold text-slate-900 dark:text-zinc-100">{{ events.length }}</span>
                <span class="text-xs font-medium text-slate-500 dark:text-zinc-500">Events</span>
            </div>
        </div>
    </div>
</template>