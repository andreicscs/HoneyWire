<script setup>
import { ref } from 'vue'
import BaseInput from '../components/ui/BaseInput.vue'
import BaseButton from '../components/ui/BaseButton.vue'
import BaseCard from '../components/ui/BaseCard.vue'
import BaseAlert from '../components/ui/BaseAlert.vue'
import PageHeader from '../components/ui/PageHeader.vue'
import ThemeToggle from '../components/ui/ThemeToggle.vue'
import BaseLogo from '../components/ui/BaseLogo.vue'

const emit = defineEmits(['login-success', 'toggle-theme'])
const password = ref('')
const loading = ref(false)
const error = ref(false)
const rateLimited = ref(false)

const doLogin = async () => {
    loading.value = true
    error.value = false
    rateLimited.value = false
    
    try {
        const res = await fetch('/login', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({ password: password.value })
        })
        
        if (res.ok) {
            emit('login-success')
        } else if (res.status === 429) {
            rateLimited.value = true
            password.value = ''
        } else {
            error.value = true
            password.value = ''
        }
    } catch (err) {
        error.value = true
    } finally {
        loading.value = false
    }
}
</script>

<template>
    <div class="h-screen flex flex-col items-center justify-center bg-bg-base p-6 relative transition-colors duration-[var(--duration-normal)]">
        
        <div class="absolute top-6 right-6">
            <ThemeToggle @toggle="$emit('toggle-theme')" />
        </div>

        <div class="w-full max-w-sm z-10">
            
            <div class="mb-10">
                <BaseLogo />
                <PageHeader 
                    center 
                    title="HoneyWire Sentinel" 
                    description="Authorized personnel only." 
                />
            </div>

            <BaseCard class="relative overflow-hidden transition-all duration-[var(--duration-normal)]">
                <div class="absolute top-0 left-0 right-0 h-1 opacity-90" style="background: linear-gradient(to right, var(--sev-info), var(--sev-low), var(--sev-medium), var(--sev-high), var(--sev-critical))"></div>

                <form @submit.prevent="doLogin" class="mt-2 space-y-6">
                    <BaseInput 
                        v-model="password" 
                        label="Authentication Key" 
                        type="password" 
                        placeholder="••••••••••••" 
                        required 
                        autofocus
                    />
                    
                    <BaseButton 
                        variant="primary" 
                        type="submit" 
                        :disabled="loading || rateLimited"
                        class="w-full h-11"
                    >
                        {{ loading ? 'Authenticating...' : 'Sign in' }}
                    </BaseButton>
                </form>

                <div v-if="error" class="mt-6">
                    <BaseAlert variant="danger" message="Access Denied: Invalid Key" />
                </div>
                
                <div v-if="rateLimited" class="mt-6">
                    <BaseAlert 
                        variant="archive" 
                        message="Too many attempts.<br/>Please try again in 15 minutes." 
                    />
                </div>
            </BaseCard>
            
        </div>
    </div>
</template>