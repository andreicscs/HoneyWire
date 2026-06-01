<script setup>
defineProps({
    variant: { 
        type: String, 
        default: 'secondary',
        validator: (v) => ['primary', 'secondary', 'danger', 'ghost'].includes(v)
    },
    disabled: { type: Boolean, default: false },
    type: { type: String, default: 'button' }
})
</script>

<template>
    <button 
        :type="type"
        :disabled="disabled"
        class="inline-flex items-center justify-center px-4 py-2 text-base rounded-md transition-all duration-fast outline-none active:scale-95 border disabled:shadow-none disabled:cursor-not-allowed disabled:active:scale-100"
        :class="{
            /* PRIMARY */
            'bg-primary-main text-primary-text border-primary-border hover:bg-primary-hover shadow-sm disabled:bg-disabled-bg disabled:text-disabled-text disabled:border-disabled-border': variant === 'primary',
            
            /* SECONDARY */
            'bg-secondary-main text-secondary-text border-secondary-border hover:bg-secondary-hover hover:text-text-h shadow-sm disabled:bg-disabled-bg disabled:text-disabled-text disabled:border-disabled-border': variant === 'secondary',
            
            /* DANGER */
            'bg-danger-bg border-danger-border hover:bg-danger-hover shadow-sm disabled:bg-disabled-bg disabled:text-disabled-text disabled:border-disabled-border': variant === 'danger',
            
            /* GHOST (Invisible until hovered) */
            'bg-transparent border-transparent text-secondary-text hover:text-text-h hover:bg-secondary-hover shadow-none disabled:text-disabled-text': variant === 'ghost'
        }"
    >
        <slot /> 
    </button>
</template>
