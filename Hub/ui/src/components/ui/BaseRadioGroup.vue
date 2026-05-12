<script setup>
defineProps({
    modelValue: { type: String, required: true },
    options: { type: Array, required: true }, // e.g. ['ntfy', 'slack']
    label: { type: String, default: '' },
    description: { type: String, default: '' }
})
defineEmits(['update:modelValue'])
</script>

<template>
    <div>
        <label v-if="label" class="block text-xs  text-text-m  tracking-wider mb-2">
            {{ label }}
        </label>
        <div class="flex flex-wrap gap-2.5">
            <button v-for="opt in options" :key="opt"
                @click="$emit('update:modelValue', opt)"
                type="button"
                class="px-3.5 py-1.5 rounded-[var(--radius-md)] text-base border transition-all duration-fast flex items-center justify-center outline-none"
                :class="modelValue === opt 
                    ? 'bg-primary-selected text-primary-text border-primary-selected shadow-sm' 
                    : 'bg-secondary-main border-secondary-border text-secondary-text hover:bg-secondary-hover hover:text-text-h'">
                {{ opt }}
            </button>
        </div>
        <p v-if="description" class="text-sm text-text-m mt-3">{{ description }}</p>
    </div>
</template>