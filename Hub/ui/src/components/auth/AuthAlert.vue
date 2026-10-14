<script setup>
import BaseAlert from '../ui/feedback/BaseAlert.vue'

defineProps({
    show: { type: Boolean, required: true },
    message: { type: String, required: true },
    alertClass: { type: String, default: '' },
    testid: { type: String, default: 'auth-alert' }
})
</script>

<template>
    <Transition
        enter-active-class="transition duration-300 ease-out"
        enter-from-class="transform -translate-y-2 opacity-0"
        enter-to-class="transform translate-y-0 opacity-100"
        leave-active-class="transition duration-200 ease-in"
        leave-from-class="transform translate-y-0 opacity-100"
        leave-to-class="transform -translate-y-2 opacity-0"
    >
        <div v-if="show" class="mt-6" :data-testid="testid">
            <BaseAlert variant="danger" :message="message" :class="alertClass" />
        </div>
    </Transition>
</template>
