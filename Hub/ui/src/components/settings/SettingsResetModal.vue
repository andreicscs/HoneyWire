<script setup>
import { ref, watch } from 'vue'
import { useAppStore } from '../../stores/System/app'
import BaseButton from '../ui/forms/BaseButton.vue'
import BaseInput from '../ui/forms/BaseInput.vue'
import BaseModal from '../ui/feedback/BaseModal.vue'
import { settingsModal } from '../../views/settingsStyles.ts'

const props = defineProps({
    show: { type: Boolean, required: true }
})
defineEmits(['close'])

const appStore = useAppStore()

const resetPassword = ref('')
const resetError = ref('')
const resetLoading = ref(false)

watch(() => props.show, (open) => {
    if (open) {
        resetPassword.value = ''
        resetError.value = ''
    }
})

const submitFactoryReset = async () => {
    if (!resetPassword.value) {
        resetError.value = "Master password is required."
        return
    }
    resetLoading.value = true
    resetError.value = ''

    // Step 1: Dry run
    const dryRunResult = await appStore.factoryReset(resetPassword.value, true)

    if (!dryRunResult.success) {
        resetLoading.value = false
        resetError.value = dryRunResult.error || "Dry run failed."
        return
    }

    resetLoading.value = false
    const stats = dryRunResult.stats

    if (confirm(`Confirm Factory Reset?\n\nThis will permanently delete:\n- ${stats.events} Security Events\n- ${stats.sensor_heartbeats} Sensor Heartbeats\n- ${stats.node_sensors} Deployed Sensors\n- ${stats.nodes} Nodes\n\nThis action cannot be undone.`)) {
        resetLoading.value = true
        const result = await appStore.factoryReset(resetPassword.value, false)
        resetLoading.value = false

        if (result.success) {
            window.location.reload()
        } else {
            resetError.value = result.error || "Reset failed."
        }
    }
}
</script>

<template>
    <BaseModal :show="show" title="Confirm Factory Reset" danger @close="$emit('close')">
        <template #icon>
            <svg :class="settingsModal.iconLg" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
        </template>
        <p :class="settingsModal.confirmCopy">This action is irreversible. All events, sensors, and configurations will be permanently deleted. Enter your master password to confirm.</p>

        <form @submit.prevent="submitFactoryReset" :class="settingsModal.form" data-testid="reset-form">
            <BaseInput
                v-model="resetPassword"
                type="password"
                placeholder="Master Password"
                required
                autofocus
                data-testid="reset-password-input"
                inputClass="!border-danger-border !focus:border-danger-main"
            />

            <div v-if="resetError" :class="settingsModal.errorCentered" data-testid="reset-error">{{ resetError }}</div>

            <div :class="settingsModal.footer">
                <BaseButton variant="ghost" @click="$emit('close'); resetPassword = ''; resetError = ''">Cancel</BaseButton>
                <BaseButton variant="danger" type="submit" :disabled="resetLoading">
                    {{ resetLoading ? 'Processing...' : 'Destroy Data' }}
                </BaseButton>
            </div>
        </form>
    </BaseModal>
</template>
