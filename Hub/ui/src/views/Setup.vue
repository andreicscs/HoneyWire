<script setup>
import { ref, onMounted } from 'vue'
import { useAppStore } from '../stores/app'
import BaseInput from '../components/ui/forms/BaseInput.vue'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import BaseCard from '../components/ui/layout/BaseCard.vue'
import BaseDivider from '../components/ui/layout/BaseDivider.vue'
import BaseAlert from '../components/ui/feedback/BaseAlert.vue'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import ThemeToggle from '../components/ui/branding/ThemeToggle.vue'
import BaseLogo from '../components/ui/branding/BaseLogo.vue'

const emit = defineEmits(['setup-complete', 'toggle-theme'])
const appStore = useAppStore()

const password = ref('')
const confirmPassword = ref('')
const hubEndpoint = ref('')
const hubKey = ref('')

const loading = ref(false)
const error = ref('')

const generateKey = () => {
    const array = new Uint8Array(16)
    crypto.getRandomValues(array)
    hubKey.value = 'hw_sk_' + Array.from(array).map(b => b.toString(16).padStart(2, '0')).join('')
}

onMounted(() => {
    hubEndpoint.value = window.location.origin
    generateKey()
})

const doSetup = async () => {
    if (!password.value || !confirmPassword.value || !hubEndpoint.value || !hubKey.value) {
        error.value = "All fields are required."
        return
    }
    if (password.value !== confirmPassword.value) {
        error.value = "Passwords do not match."
        return
    }
    loading.value = true
    error.value = ''

    const result = await appStore.completeSetup(password.value, hubEndpoint.value, hubKey.value)

    loading.value = false
    if (result.success) {
        emit('setup-complete')
    } else {
        error.value = result.error || "Setup failed."
    }
}
</script>

<template>
    <div class="min-h-screen flex flex-col items-center justify-center bg-bg-base p-6 relative py-12 overflow-y-auto transition-colors duration-normal">
        
        <div class="absolute top-6 right-6">
            <ThemeToggle @toggle="$emit('toggle-theme')" />
        </div>

        <div class="w-full max-w-xl z-10 my-auto">
            
            <div class="mb-10">
                <BaseLogo />
                <PageHeader 
                    center 
                    title="HoneyWire Sentinel" 
                    description="Authorized personnel only." 
                />
            </div>

            <BaseCard class="relative overflow-hidden transition-all duration-normal">
                <div class="absolute top-0 left-0 right-0 h-1 opacity-90" style="background: linear-gradient(to right, var(--sev-info), var(--sev-low), var(--sev-medium), var(--sev-high), var(--sev-critical))"></div>

                <form @submit.prevent="doSetup" class="mt-4 space-y-6">
                    
                    <div>
                        <PageHeader size="sm" title="Master Authentication" />
                        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                            <BaseInput 
                                v-model="password" 
                                label="Master Password" 
                                type="password" 
                                placeholder="Required" 
                                required 
                            />
                            <BaseInput 
                                v-model="confirmPassword" 
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
                        <div class="space-y-6">
                            <BaseInput 
                                v-model="hubEndpoint" 
                                label="Hub Endpoint URL" 
                                placeholder="http://yourip:8080"
                                description="The URL sensors use to send telemetry."
                                required 
                            />
                            
                            <div class="flex gap-2 items-end">
                                <BaseInput 
                                    v-model="hubKey" 
                                    label="Sensor Secret Key"
                                    placeholder="Secure API Key" 
                                    required 
                                    class="flex-1"
                                />
                                <div class="mb-0.5">
                                    <BaseButton variant="secondary" @click="generateKey">
                                        Generate
                                    </BaseButton>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="pt-4">
                        <BaseButton 
                            variant="primary" 
                            type="submit" 
                            :disabled="loading" 
                            class="w-full h-11"
                        >
                            {{ loading ? 'Initializing Sentinel...' : 'Initialize Hub' }}
                        </BaseButton>
                    </div>
                </form>

                <div v-if="error" class="mt-6">
                    <BaseAlert variant="danger" :message="error" />
                </div>
            </BaseCard>
        </div>
    </div>
</template>