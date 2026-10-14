<script setup>
import { ref, watch, computed } from 'vue'
import { useConfigStore } from '../stores/Config/config'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import BaseVerticalNav from '../components/ui/navigation/BaseVerticalNav.vue'
import SettingsGeneralTab from '../components/settings/SettingsGeneralTab.vue'
import SettingsDataTab from '../components/settings/SettingsDataTab.vue'
import SettingsAlertsTab from '../components/settings/SettingsAlertsTab.vue'
import SettingsSiemTab from '../components/settings/SettingsSiemTab.vue'
import SettingsSecurityTab from '../components/settings/SettingsSecurityTab.vue'
import SettingsPasswordModal from '../components/settings/SettingsPasswordModal.vue'
import SettingsResetModal from '../components/settings/SettingsResetModal.vue'
import {
    settingsLayout,
    settingsHeader,
} from './settingsStyles.ts'

const configStore = useConfigStore()
const activeTab = ref('general')

const settingTabs = [
    { id: 'general', label: 'Hub Configuration', iconSvg: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>' },
    { id: 'data', label: 'Data Retention', iconSvg: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"></path></svg>' },
    { id: 'alerts', label: 'Push Notifications', iconSvg: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"></path></svg>' },
    { id: 'siem', label: 'SIEM Forwarding', iconSvg: '<svg fill="currentColor" viewBox="0 0 100 100"><path d="M77.2,56.2h-3.7c-1,0-1.8,1-1.8,1.8v12.3c0,1-0.9,1.8-1.8,1.8H29.3c-1,0-1.8-0.9-1.8-1.8V58 c0-0.9-0.9-1.8-1.8-1.8h-3.7c-1,0-1.8,1-1.8,1.8v16.6c0,2.7,2.2,4.9,4.9,4.9h49.1c2.7,0,4.9-2.2,4.9-4.9V58 C79,57.1,78.2,56.2,77.2,56.2z M50.8,21c-0.7-0.7-1.8-0.7-2.6,0L31.6,37.6c-0.7,0.7-0.7,1.8,0,2.6l2.6,2.6c0.7,0.7,1.8,0.7,2.6,0 l6.9-6.9c0.7-0.7,2.2-0.2,2.2,0.9v26c0,1,0.7,1.8,1.7,1.8h3.7c1,0,2-1,2-1.8V36.9c0-1.1,1.2-1.6,2.1-0.9l6.9,6.9 c0.7,0.7,1.8,0.7,2.6,0l2.6-2.6c0.7-0.7,0.7-1.8,0-2.6C67.3,37.7,50.8,21,50.8,21z"></path></svg>' },
    { divider: true },
    { id: 'security', label: 'Security & Access', danger: true, iconSvg: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path></svg>' }
]

const settings = ref({
    hubEndpoint: window.location.origin,
    registryUrl: '',
    autoArchiveDays: 0,
    autoPurgeDays: 0,
    autoPurgeHeartbeatsDays: 0,
    updateCheckIntervalHours: 24,
    webhookType: 'ntfy',
    webhookUrl: '',
    webhookEvents: [],
    siemAddress: '',
    siemProtocol: 'tcp'
})

const initialSettings = ref(null)

watch(() => configStore.config.isLoaded, (loaded) => {
    if (loaded) {
        const loadedSettings = {
            hubEndpoint: configStore.config.hubEndpoint || window.location.origin,
            registryUrl: configStore.config.registryUrl || '',
            autoArchiveDays: configStore.config.autoArchiveDays !== undefined ? configStore.config.autoArchiveDays : 0,
            autoPurgeDays: configStore.config.autoPurgeDays !== undefined ? configStore.config.autoPurgeDays : 0,
            autoPurgeHeartbeatsDays: configStore.config.autoPurgeHeartbeatsDays !== undefined ? configStore.config.autoPurgeHeartbeatsDays : 0,
            updateCheckIntervalHours: configStore.config.updateCheckIntervalHours !== undefined ? configStore.config.updateCheckIntervalHours : 24,
            webhookType: configStore.config.webhookType || 'ntfy',
            webhookUrl: configStore.config.webhookUrl || '',
            webhookEvents: configStore.config.webhookEvents && configStore.config.webhookEvents.length > 0 
                ? [...configStore.config.webhookEvents] 
                : ['critical', 'high', 'medium', 'low', 'info'],
            siemAddress: configStore.config.siemAddress || '',
            siemProtocol: configStore.config.siemProtocol || 'tcp',
            whitelistedSources: configStore.config.whitelistedSources || ''
        }
        settings.value = JSON.parse(JSON.stringify(loadedSettings))
        initialSettings.value = JSON.parse(JSON.stringify(loadedSettings))
    }
}, { immediate: true })

const hasUnsavedChanges = computed(() => {
    if (!initialSettings.value) return false
    return JSON.stringify(settings.value) !== JSON.stringify(initialSettings.value)
})

const isSaving = ref(false)
const saveMessage = ref('')
const isError = ref(false)

const saveSettings = async () => {
    isSaving.value = true
    saveMessage.value = ''
    isError.value = false

    const success = await configStore.patchConfig(settings.value)
    
    isSaving.value = false
    if (success) {
        initialSettings.value = JSON.parse(JSON.stringify(settings.value))
        saveMessage.value = 'Configuration saved successfully.'
        setTimeout(() => saveMessage.value = '', 3000)
    } else {
        isError.value = true
        saveMessage.value = 'Failed to save configuration.'
    }
}

// --- SECURITY MODALS (open state only; form state lives in the modal components) ---

const showPasswordModal = ref(false)
const showResetModal = ref(false)
</script>

<template>
    <div :class="settingsLayout.pageRoot" data-testid="settings-page">
        
        <PageHeader title="System Settings" description="Manage Hub configuration, retention policies, and push notifications.">
            <template #actions>
                <span v-if="saveMessage && isError" :class="settingsHeader.saveError">
                    {{ saveMessage }}
                </span>

                <span v-else-if="saveMessage && !isError" :class="settingsHeader.saveSuccess">
                    {{ saveMessage }}
                </span>

                <span v-else-if="hasUnsavedChanges" :class="settingsHeader.unsavedHint">
                    <span :class="settingsHeader.unsavedDot"></span> Unsaved changes
                </span>

                <BaseButton variant="primary" data-testid="settings-save-button" :disabled="isSaving || !hasUnsavedChanges" @click="saveSettings">
                    {{ isSaving ? 'Saving...' : 'Save Changes' }}
                </BaseButton>
            </template>
        </PageHeader>
        
        <div :class="settingsLayout.contentRow">
            
            <BaseVerticalNav v-model="activeTab" :tabs="settingTabs" />

            <div :class="settingsLayout.scrollRegion" data-testid="settings-scroll-region">
                
                <div v-show="activeTab === 'general'">
                    <SettingsGeneralTab v-model="settings" />
                </div>

                <div v-show="activeTab === 'data'">
                    <SettingsDataTab v-model="settings" />
                </div>

                <div v-show="activeTab === 'alerts'">
                    <SettingsAlertsTab v-model="settings" />
                </div>

                <div v-show="activeTab === 'siem'">
                    <SettingsSiemTab v-model="settings" />
                </div>

                <div v-show="activeTab === 'security'">
                    <SettingsSecurityTab
                        @change-password="showPasswordModal = true"
                        @reset-system="showResetModal = true"
                    />
                </div>

            </div>
        </div>

        <SettingsPasswordModal :show="showPasswordModal" @close="showPasswordModal = false" />
        <SettingsResetModal :show="showResetModal" @close="showResetModal = false" />

    </div>
</template>