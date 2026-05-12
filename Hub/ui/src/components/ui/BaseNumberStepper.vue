<script setup>
const props = defineProps({
    modelValue: { type: Number, required: true },
    label: { type: String, required: true },
    description: { type: String, default: '' },
    min: { type: Number, default: 0 },
    max: { type: Number, default: 100 },
    suffix: { type: String, default: '' }
})
const emit = defineEmits(['update:modelValue'])

const adjust = (delta) => {
    let val = props.modelValue + delta
    if (val < props.min) val = props.min
    if (val > props.max) val = props.max
    emit('update:modelValue', val)
}
</script>

<template>
    <div class="flex items-center justify-between gap-4 max-w-2xl">
        <div>
            <label class="block text-base text-text-h">{{ label }}</label>
            <p v-if="description" class="text-sm text-text-m mt-[var(--space-label-gap)]">{{ description }}</p>
        </div>
        <div class="flex items-center gap-3">
            <div class="flex items-center rounded-[var(--radius-md)] border border-input-border overflow-hidden bg-input-bg shadow-inner focus-within:ring-1 focus-within:ring-focus-ring focus-within:border-primary-main transition-colors duration-fast">
                <button @click="adjust(-1)" type="button" class="px-3 py-1.5 text-text-m hover:bg-secondary-hover transition-colors duration-fast select-none outline-none">-</button>
                <input 
                    :value="modelValue" 
                    @input="$emit('update:modelValue', Number($event.target.value))" 
                    type="number" :min="min" :max="max"
                    class="w-12 text-center text-base font-mono bg-transparent border-none focus:outline-none focus:ring-0 text-text-h hide-arrows p-0" 
                />
                <button @click="adjust(1)" type="button" class="px-3 py-1.5 text-text-m hover:bg-secondary-hover transition-colors duration-fast select-none outline-none">+</button>
            </div>
            <span v-if="suffix" class="text-base text-text-m w-10">{{ suffix }}</span>
        </div>
    </div>
</template>

<style scoped>
.hide-arrows::-webkit-outer-spin-button,
.hide-arrows::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.hide-arrows[type=number] {
  -moz-appearance: textfield;
}
</style>