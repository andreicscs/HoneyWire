<script setup>
defineProps({
    show: { type: Boolean, required: true },
    title: { type: String, required: true },
    danger: { type: Boolean, default: false }
})
defineEmits(['close'])
</script>

<template>
    <Teleport to="body">
        <transition 
            enter-active-class="transition duration-normal ease-out" 
            enter-from-class="opacity-0" 
            enter-to-class="opacity-100" 
            leave-active-class="transition duration-fast ease-in" 
            leave-from-class="opacity-100" 
            leave-to-class="opacity-0"
        >
            <div v-if="show" class="fixed inset-0 z-modal flex justify-center items-center p-4 bg-black/60 backdrop-blur-sm" @click.self="$emit('close')">
                <div 
                    class="bg-bg-surface w-full max-w-sm rounded-lg shadow-lg p-[var(--space-card-p)] transform transition-all border"
                    :class="danger ? 'border-danger-border' : 'border-border-default'"
                >
                    <div class="flex items-center gap-3 mb-5" :class="danger ? 'text-danger-text' : 'text-text-h'">
                        <slot name="icon" />
                        <h3 class="text-h1">{{ title }}</h3>
                    </div>
                    
                    <slot />
                </div>
            </div>
        </transition>
    </Teleport>
</template>