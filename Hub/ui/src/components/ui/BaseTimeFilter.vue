<script setup>
defineProps({
    modelValue: { type: String, required: true },
    options: { type: Array, default: () => ['1H', '24H', '7D', '30D'] }
})
defineEmits(['update:modelValue'])
</script>

<template>
    <div class="flex bg-secondary-main p-1 rounded-[var(--radius-md)] border border-secondary-border text-sm w-fit shadow-inner">
        <button v-for="time in options" :key="time"
                @click="$emit('update:modelValue', time)"
                class="px-3 py-1 rounded-[var(--radius-sm)] transition-all duration-fast outline-none"
                :class="modelValue === time 
                    ? 'bg-primary-selected text-primary-text shadow-sm' 
                    : 'text-secondary-text hover:bg-secondary-hover hover:text-text-h'">
            {{ time }}
        </button>
    </div>
</template>
>
