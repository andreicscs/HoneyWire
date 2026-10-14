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

const pwdData = ref({ current: '', new: '', confirmNew: '' })
const pwdError = ref('')
const pwdLoading = ref(false)

watch(() => props.show, (open) => {
    if (open) {
        pwdData.value = { current: '', new: '', confirmNew: '' }
        pwdError.value = ''
    }
})

const submitPasswordChange = async () => {
    if (!pwdData.value.current || !pwdData.value.new || !pwdData.value.confirmNew) {
        pwdError.value = "All fields are required."
        return
    }
    if (pwdData.value.new !== pwdData.value.confirmNew) {
        pwdError.value = "New passwords do not match."
        return
    }
    pwdLoading.value = true
    pwdError.value = ''

    const result = await appStore.changePassword(pwdData.value.current, pwdData.value.new)

    pwdLoading.value = false
    if (result.success) {
        window.location.reload()
    } else {
        pwdError.value = result.error
    }
}
</script>

<template>
    <BaseModal :show="show" title="Update Password" @close="$emit('close')">
        <template #icon>
            <svg :class="settingsModal.icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z"></path></svg>
        </template>
        <form @submit.prevent="submitPasswordChange" :class="settingsModal.form" data-testid="password-form">
            <BaseInput v-model="pwdData.current" label="Current Password" type="password" required autofocus data-testid="pwd-current" />
            <BaseInput v-model="pwdData.new" label="New Password" type="password" required data-testid="pwd-new" />
            <BaseInput v-model="pwdData.confirmNew" label="Confirm New Password" type="password" required data-testid="pwd-confirm" />

            <div v-if="pwdError" :class="settingsModal.error" data-testid="password-error">{{ pwdError }}</div>

            <div :class="settingsModal.footer">
                <BaseButton variant="ghost" @click="$emit('close')">Cancel</BaseButton>
                <BaseButton variant="primary" type="submit" :disabled="pwdLoading">
                    {{ pwdLoading ? 'Updating...' : 'Update Password' }}
                </BaseButton>
            </div>
        </form>
    </BaseModal>
</template>
