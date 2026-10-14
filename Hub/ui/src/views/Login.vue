<script setup>
import { ref } from 'vue'
import { useAppStore } from '../stores/System/app'
import BaseInput from '../components/ui/forms/BaseInput.vue'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import AuthLayout from '../components/auth/AuthLayout.vue'
import AuthAlert from '../components/auth/AuthAlert.vue'
import { authStyles } from './authStyles.ts'

const emit = defineEmits(['login-success', 'toggle-theme'])
const appStore = useAppStore()

const password = ref('')
const loading = ref(false)
const error = ref(false)
const rateLimited = ref(false)

const doLogin = async () => {
    loading.value = true
    error.value = false
    rateLimited.value = false

    const result = await appStore.login(password.value)

    if (result.success) {
        emit('login-success')
    } else if (result.status === 429) {
        rateLimited.value = true
        password.value = ''
    } else {
        error.value = true
        password.value = ''
    }

    loading.value = false
}
</script>

<template>
    <AuthLayout @toggle-theme="$emit('toggle-theme')">
        <form @submit.prevent="doLogin" :class="authStyles.formTight">
            <BaseInput
                v-model="password"
                data-testid="login-password"
                label="Authentication Key"
                type="password"
                placeholder="••••••••••••"
                required
                autofocus
            />

            <BaseButton
                variant="primary"
                type="submit"
                data-testid="login-submit"
                :disabled="loading || rateLimited"
                class="w-full h-11"
            >
                {{ loading ? 'Authenticating...' : 'Sign in' }}
            </BaseButton>
        </form>

        <AuthAlert :show="error" message="Access Denied: Invalid Key" testid="login-error" />
        <AuthAlert
            :show="rateLimited"
            :message="'Too many attempts.\nPlease try again in 15 minutes.'"
            alert-class="whitespace-pre-line"
            testid="login-rate-limited"
        />
    </AuthLayout>
</template>
