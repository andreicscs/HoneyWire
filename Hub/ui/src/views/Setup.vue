<script setup>
import { ref, onMounted } from 'vue'
import { useAppStore } from '../stores/System/app'
import BaseInput from '../components/ui/forms/BaseInput.vue'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import BaseDivider from '../components/ui/layout/BaseDivider.vue'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import AuthLayout from '../components/auth/AuthLayout.vue'
import AuthAlert from '../components/auth/AuthAlert.vue'
import { authStyles } from './authStyles.ts'

const emit = defineEmits(['setup-complete', 'toggle-theme'])
const appStore = useAppStore()

const password = ref('')
const confirmPassword = ref('')
const hubEndpoint = ref('')

const loading = ref(false)
const error = ref('')

onMounted(() => {
    hubEndpoint.value = window.location.origin
})

const doSetup = async () => {
    if (!password.value || !confirmPassword.value || !hubEndpoint.value) {
        error.value = "All fields are required."
        return
    }
    if (password.value !== confirmPassword.value) {
        error.value = "Passwords do not match."
        return
    }
    loading.value = true
    error.value = ''

    const result = await appStore.completeSetup(password.value, hubEndpoint.value)

    loading.value = false
    if (result.success) {
        emit('setup-complete')
    } else {
        error.value = result.error || "Setup failed."
    }
}
</script>

<template>
    <AuthLayout width="xl" scrollable @toggle-theme="$emit('toggle-theme')">
        <form @submit.prevent="doSetup" :class="authStyles.formRelaxed">

            <div>
                <PageHeader size="sm" title="Master Authentication" />
                <div :class="authStyles.setupGrid">
                    <BaseInput
                        v-model="password"
                        data-testid="setup-password"
                        label="Master Password"
                        type="password"
                        placeholder="Required"
                        required
                        autofocus
                    />
                    <BaseInput
                        v-model="confirmPassword"
                        data-testid="setup-confirm"
                        label="Confirm Password"
                        type="password"
                        placeholder="Repeat"
                        required
                    />
                </div>
            </div>

            <BaseDivider />

            <div>
                <PageHeader size="sm" title="Hub Connectivity" />
                <div :class="authStyles.setupStack">
                    <BaseInput
                        v-model="hubEndpoint"
                        data-testid="setup-endpoint"
                        label="Hub Endpoint URL"
                        placeholder="http://yourip:8080"
                        description="The URL sensors use to send telemetry."
                        required
                    />
                </div>
            </div>

            <div :class="authStyles.setupSubmitWrap">
                <BaseButton
                    variant="primary"
                    type="submit"
                    data-testid="setup-submit"
                    :disabled="loading"
                    class="w-full h-11"
                >
                    {{ loading ? 'Initializing Sentinel...' : 'Initialize Hub' }}
                </BaseButton>
            </div>
        </form>

        <AuthAlert :show="!!error" :message="error" testid="setup-error" />
    </AuthLayout>
</template>
