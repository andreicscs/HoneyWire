<script setup>
defineProps({
    title: { type: String, default: '' },
    danger: { type: Boolean, default: false }
})
</script>

<template>
    <div 
        class="border rounded-[var(--radius-lg)] p-[var(--space-card-p)] shadow-sm transition-colors duration-normal"
        :class="danger ? 'bg-danger-bg border-danger-border' : 'bg-bg-surface border-border-default'"
    >
        <div v-if="title || $slots.icon" class="mb-5 flex items-center gap-2">
            <span v-if="$slots.icon" :class="danger ? 'text-danger-text' : 'text-text-h'">
                <slot name="icon" />
            </span>
            <h3 class="text-base" :class="danger ? 'text-danger-text' : 'text-text-h'">
                {{ title }}
            </h3>
        </div>
        
        <div class="space-y-6">
            <slot />
        </div>
    </div>
</template>