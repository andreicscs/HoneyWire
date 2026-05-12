<script setup>
defineProps({
    title: { type: String, required: true },
    description: { type: String, default: '' },
    size: { type: String, default: 'md' }, // 'sm' or 'md'
    center: { type: Boolean, default: false }
})
</script>

<template>
    <div :class="[
        size === 'md' ? 'mb-[var(--space-flow)] mt-4 sm:mt-6' : 'mb-4',
        'shrink-0 flex flex-col gap-4',
        /* If not centered, go row-based on small screens and up */
        !center ? 'sm:flex-row sm:items-end sm:justify-between' : 'items-center text-center'
    ]">
        <div :class="['flex flex-col gap-1', !center ? 'text-left' : 'text-center']">
            <h1 v-if="size === 'md'" class="text-h1 text-text-h tracking-tight">
                {{ title }}
            </h1>
            <h3 v-else class="text-base tracking-wider text-text-h">
                {{ title }}
            </h3>
            
            <p v-if="description" :class="[
                size === 'md' ? 'text-base' : 'text-sm',
                'text-text-m max-w-3xl',
                center ? 'mx-auto' : ''
            ]">
                {{ description }}
            </p>
        </div>

        <div v-if="$slots.actions" 
             :class="['flex items-center gap-4', center ? 'justify-center' : 'sm:justify-end shrink-0']"
        >
            <slot name="actions" />
        </div>
    </div>
</template>