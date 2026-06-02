<script setup>
defineProps({
    variant: { 
        type: String, 
        default: 'info',
        validator: (v) => ['info', 'danger', 'success', 'archive'].includes(v)
    },
    message: { type: String, required: true }
})
</script>

<template>
    <div 
        class="p-[var(--space-card-p)] rounded-md border text-center transition-colors duration-normal"
        :class="{
            'bg-danger-bg border-danger-border text-danger-text': variant === 'danger',
            'bg-success-bg border-success-border text-success-text': variant === 'success',
            'bg-archive-bg border-archive-border text-archive-text': variant === 'archive',
            'bg-bg-inset border-border-default text-text-m': variant === 'info'
        }"
    >
        <p class="text-base">{{ message }}</p>
    </div>
</template>