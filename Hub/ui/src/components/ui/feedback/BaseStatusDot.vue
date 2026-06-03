<script setup>
defineProps({
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
</script>

<template>
    <span 
        class="w-2 h-2 rounded-full shrink-0 transition-colors duration-[var(--duration-normal)]"
        :class="{
            'bg-success-main': ['up', 'online'].includes((status || '').toLowerCase()),
            'bg-danger-main': ['down', 'offline', 'critical'].includes((status || '').toLowerCase()),
            'bg-high': ['degraded', 'high'].includes((status || '').toLowerCase()),
            'bg-bg-inset': ['nodata', 'pending'].includes((status || '').toLowerCase()) || !status,
            'animate-pulse shadow-sm': animate
        }"
    ></span>
</template>