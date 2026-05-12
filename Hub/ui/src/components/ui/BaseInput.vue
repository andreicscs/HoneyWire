<script setup>
defineProps({
    modelValue: { type: [String, Number], default: '' },
    label: { type: String, default: '' },
    description: { type: String, default: '' },
    type: { type: String, default: 'text' },
    placeholder: { type: String, default: '' },
    required: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false }
})
defineEmits(['update:modelValue', 'focus', 'blur'])
</script>

<template>
    <div class="space-y-[var(--space-label-gap)] w-full">
        <label v-if="label" class="block text-base text-text-m">
            {{ label }} <span v-if="required" class="text-danger-text">*</span>
        </label>
        
        <input 
            :type="type" 
            :value="modelValue"
            @input="$emit('update:modelValue', $event.target.value)"
            @focus="$emit('focus', $event)"
            @blur="$emit('blur', $event)"
            :placeholder="placeholder"
            :required="required"
            :disabled="disabled"
            class="w-full px-3 py-2 rounded-[var(--radius-md)] text-base text-text-h transition-colors duration-fast shadow-inner outline-none disabled:cursor-not-allowed"
            :class="disabled 
                ? 'bg-disabled-bg border border-disabled-border text-disabled-text' 
                : 'bg-input-bg border border-input-border focus:border-primary-main focus:ring-1 focus:ring-focus-ring placeholder:text-text-m/50'" 
        />

        <p v-if="description" class="text-sm text-text-m">{{ description }}</p>
    </div>
</template>