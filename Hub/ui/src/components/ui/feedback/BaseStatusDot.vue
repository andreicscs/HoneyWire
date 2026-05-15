<script setup>
defineProps({
    status: {
        type: String,
        required: true,
        // Accepts: 'up', 'down', 'degraded', 'nodata', 'online', 'offline'
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
            'bg-success-main': ['up', 'online'].includes(status.toLowerCase()),
            'bg-danger-main': ['down', 'offline', 'critical'].includes(status.toLowerCase()),
            'bg-high': ['degraded', 'high'].includes(status.toLowerCase()),
            'bg-bg-inset': status.toLowerCase() === 'nodata',
            'animate-pulse shadow-sm': animate
        }"
    ></span>
</template>