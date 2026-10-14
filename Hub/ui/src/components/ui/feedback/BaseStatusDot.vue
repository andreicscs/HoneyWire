<script setup>
import { computed } from 'vue'

const props = defineProps({
    status: {
        type: String,
        required: true,
        // Accepts: 'up', 'down', 'degraded', 'nodata', 'online', 'offline', 'unknown', 'pending'
    },
    animate: {
        type: Boolean,
        default: false
    }
})

// Collapses any status string to a single tone. Unknown statuses render
// without a tone color, exactly like the previous per-status class map.
const tone = computed(() => {
    const s = (props.status || '').toLowerCase()
    if (['up', 'online'].includes(s)) return 'success'
    if (['down', 'offline', 'critical'].includes(s)) return 'danger'
    if (['degraded', 'high'].includes(s)) return 'warning'
    if (['nodata', 'pending'].includes(s) || !s) return 'muted'
    return ''
})

const toneColors = {
    success: 'bg-success-main',
    danger: 'bg-danger-main',
    warning: 'bg-high',
    muted: 'bg-bg-inset',
}
</script>

<template>
    <span
        class="w-2 h-2 rounded-full shrink-0 transition-colors duration-[var(--duration-normal)]"
        :data-tone="tone"
        :class="[toneColors[tone], animate ? 'animate-pulse shadow-sm' : '']"
    ></span>
</template>
