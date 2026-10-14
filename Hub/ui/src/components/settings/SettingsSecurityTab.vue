<script setup>
import BaseButton from '../ui/forms/BaseButton.vue'
import BaseCard from '../ui/layout/BaseCard.vue'
import { settingsSecurity } from '../../views/settingsStyles.ts'

defineEmits(['change-password', 'reset-system'])
</script>

<template>
    <BaseCard title="Authentication">
        <p :class="settingsSecurity.desc">Update the master password used to access this dashboard. You will be logged out immediately upon changing this.</p>
        <BaseButton variant="secondary" data-testid="change-password-button" @click="$emit('change-password')">
            Change Master Password
        </BaseButton>
    </BaseCard>

    <BaseCard danger>
        <template #icon>
            <svg :class="settingsSecurity.icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
        </template>
        <template #default>
            <h3 :class="settingsSecurity.dangerTitle">Danger Zone</h3>
            <div :class="settingsSecurity.dangerRow" data-testid="danger-zone">
                <div>
                    <h4 :class="settingsSecurity.dangerName">Factory Reset</h4>
                    <p :class="settingsSecurity.dangerDesc">Wipes all configuration, logs, and authentication keys. The application will restart in setup mode.</p>
                </div>
                <BaseButton variant="danger" data-testid="reset-system-button" @click="$emit('reset-system')" :class="settingsSecurity.shrink">
                    Reset System
                </BaseButton>
            </div>
        </template>
    </BaseCard>
</template>
