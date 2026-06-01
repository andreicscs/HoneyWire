<script setup>
import { ref, watch, computed } from 'vue'
import { useAppStore } from '../stores/System/app'
import { useConfigStore } from '../stores/Config/config'
import BaseButton from '../components/ui/forms/BaseButton.vue'
import BaseInput from '../components/ui/forms/BaseInput.vue'
import BaseModal from '../components/ui/feedback/BaseModal.vue'
import PageHeader from '../components/ui/layout/PageHeader.vue'
import BaseCard from '../components/ui/layout/BaseCard.vue'
import BaseDivider from '../components/ui/layout/BaseDivider.vue'
import BaseVerticalNav from '../components/ui/navigation/BaseVerticalNav.vue'
import BaseRadioGroup from '../components/ui/forms/BaseRadioGroup.vue'
import BaseNumberStepper from '../components/ui/forms/BaseNumberStepper.vue'

const appStore = useAppStore()
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
    autoArchiveDays: 0,
    autoPurgeDays: 0,
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
            autoArchiveDays: configStore.config.autoArchiveDays !== undefined ? configStore.config.autoArchiveDays : 0,
            autoPurgeDays: configStore.config.autoPurgeDays !== undefined ? configStore.config.autoPurgeDays : 0,
            webhookType: configStore.config.webhookType || 'ntfy',
            webhookUrl: configStore.config.webhookUrl || '',
            webhookEvents: configStore.config.webhookEvents && configStore.config.webhookEvents.length > 0 
                ? [...configStore.config.webhookEvents] 
                : ['critical', 'high', 'medium', 'low', 'info'],
            siemAddress: configStore.config.siemAddress || '',
            siemProtocol: configStore.config.siemProtocol || 'tcp'
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

const toggleSeverity = (sev) => {
    const index = settings.value.webhookEvents.indexOf(sev)
    if (index === -1) {
        settings.value.webhookEvents.push(sev)
    } else {
        settings.value.webhookEvents.splice(index, 1)
    }
}

const getSeverityStyle = (sev, isActive) => {
    if (!isActive) {
        return { 
            backgroundColor: 'var(--bg-inset)', 
            borderColor: 'var(--border-default)', 
            color: 'var(--text-muted)' 
        };
    }
    const color = `var(--sev-${sev.toLowerCase()})`;
    return {
        backgroundColor: `color-mix(in srgb, ${color} 15%, transparent)`,
        borderColor: `color-mix(in srgb, ${color} 50%, transparent)`,
        color: color
    };
}

// --- SECURITY MODALS (ephemeral UI + store delegation) ---

const showPasswordModal = ref(false)
const pwdData = ref({ current: '', new: '', confirmNew: '' })
const pwdError = ref('')
const pwdLoading = ref(false)

const showResetModal = ref(false)
const resetPassword = ref('')
const resetError = ref('')
const resetLoading = ref(false)

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

const submitFactoryReset = async () => {
    if (!resetPassword.value) {
        resetError.value = "Master password is required."
        return
    }
    resetLoading.value = true
    resetError.value = ''

    const result = await appStore.factoryReset(resetPassword.value)

    resetLoading.value = false
    if (result.success) {
        window.location.reload()
    } else {
        resetError.value = result.error
    }
}
</script>

<template>
    <div class="h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6 pb-4 sm:pb-6 transition-colors duration-200">
        
        <PageHeader title="System Settings" description="Manage Hub configuration, retention policies, and push notifications.">
            <template #actions>
                <span v-if="saveMessage && isError" class="text-xs text-danger-main hidden hidden sm:block">
                    {{ saveMessage }}
                </span>

                <span v-else-if="saveMessage && !isError" class="text-xs text-success-main hidden sm:block">
                    {{ saveMessage }}
                </span>

                <span v-else-if="hasUnsavedChanges" class="text-xs  text-medium animate-pulse hidden sm:flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full bg-medium inline-block"></span> Unsaved changes
                </span>

                <BaseButton variant="primary" :disabled="isSaving || !hasUnsavedChanges" @click="saveSettings">
                    {{ isSaving ? 'Saving...' : 'Save Changes' }}
                </BaseButton>
            </template>
        </PageHeader>
        
        <div class="flex flex-col md:flex-row gap-6 flex-1 min-h-0">
            
            <BaseVerticalNav v-model="activeTab" :tabs="settingTabs" />

            <div class="flex-1 overflow-y-auto custom-scroll pr-2 space-y-6 after:content-[''] after:block after:h-6 after:shrink-0">
                
                <div v-show="activeTab === 'general'">
                    <BaseCard title="Network Configuration">
                        <div class="max-w-md">
                            <BaseInput 
                                v-model="settings.hubEndpoint" 
                                label="Hub Endpoint URL" 
                                description="The publicly accessible URL or IP where sensors will send their telemetry." 
                            />
                        </div>
                        
                    </BaseCard>
                </div>

                <div v-show="activeTab === 'data'">
                    <BaseCard title="Database Retention Policies">
                        <BaseNumberStepper 
                            v-model="settings.autoArchiveDays" 
                            label="Auto-Archive Events" 
                            description="Move events from the Live Queue to the Archive automatically."
                            :min="0" :max="365" 
                            :suffix="settings.autoArchiveDays === 1 ? 'Day' : 'Days'"
                            :class="settings.autoArchiveDays === 0 ? '[&_input]:!text-disabled-text' : ''"
                        />
                        
                        <BaseDivider />
                        
                        <BaseNumberStepper 
                            v-model="settings.autoPurgeDays" 
                            label="Auto-Purge Archive" 
                            description="Permanently delete archived events from the SQLite database." 
                            :min="0" :max="365" 
                            :suffix="settings.autoPurgeDays === 1 ? 'Day' : 'Days'"
                            :class="settings.autoPurgeDays === 0 ? '[&_input]:!text-disabled-text' : ''"
                        />
                    </BaseCard>
                </div>

                <div v-show="activeTab === 'alerts'">
                    <BaseCard title="Push Notifications">
                        <BaseRadioGroup 
                            v-model="settings.webhookType" 
                            label="Service Provider" 
                            :options="['ntfy', 'gotify', 'discord', 'slack']" 
                        />
                        
                        <BaseDivider />
                        
                        <div class="max-w-xl">
                            <BaseInput 
                                v-model="settings.webhookUrl" 
                                label="Target URL" 
                                type="url"
                                placeholder="https://..."
                                :description="
                                    settings.webhookType === 'gotify' ? 'Enter your Gotify server URL and append the App Token (e.g., https://gotify.domain.com/message?token=XYZ).' : 
                                    settings.webhookType === 'ntfy' ? 'Enter your self-hosted or public Ntfy topic URL (e.g., https://ntfy.sh/my_alerts).' : 
                                    `Paste the incoming Webhook URL provided by ${settings.webhookType === 'discord' ? 'Discord' : 'Slack'}.`
                                " 
                            />
                        </div>
                        
                        <BaseDivider />
                        
                        <div>
                            <label class="block text-sm  tracking-wider  text-text-m mb-2">Trigger Severities</label>
                            <div class="flex flex-wrap gap-2.5">
                                <button v-for="sev in ['critical', 'high', 'medium', 'low', 'info']" :key="sev"
                                        @click="toggleSeverity(sev)"
                                        class="text-sm   tracking-wider px-3.5 py-1.5 rounded-md transition-all border outline-none select-none hover:opacity-80"
                                        :style="getSeverityStyle(sev, settings.webhookEvents.includes(sev))">
                                    {{ sev }}
                                </button>
                            </div>
                        </div>
                    </BaseCard>
                </div>

                <div v-show="activeTab === 'siem'">
                    <BaseCard title="SIEM Forwarding">
                        <div class="max-w-xl">
                            <BaseInput 
                                v-model="settings.siemAddress" 
                                label="Server Address" 
                                placeholder="host:port"
                                description="Forward syslog events to your SIEM (e.g., elk.example.com:514)." 
                            />
                        </div>
                        
                        <BaseDivider />
                        
                        <BaseRadioGroup 
                            v-model="settings.siemProtocol" 
                            label="Protocol" 
                            :options="['tcp', 'udp']" 
                            description="Events are sent in RFC3164 syslog format. Leave blank to disable SIEM forwarding."
                        />
                    </BaseCard>
                </div>

                <div v-show="activeTab === 'security'" class="space-y-6">
                    <BaseCard title="Authentication">
                        <p class="text-sm text-text-m mb-4 max-w-2xl">Update the master password used to access this dashboard. You will be logged out immediately upon changing this.</p>
                        <BaseButton variant="secondary" @click="pwdData = {current:'', new:'', confirmNew:''}; pwdError = ''; showPasswordModal = true">
                            Change Master Password
                        </BaseButton>
                    </BaseCard>

                    <BaseCard danger>
                        <template #icon>
                            <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
                        </template>
                        <template #default>
                            <h3 class="text-base  text-danger-text -mt-10 mb-4 ml-7">Danger Zone</h3>
                            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-bg-surface border border-danger-border p-5 rounded-lg shadow-sm transition-colors">
                                <div>
                                    <h4 class="text-base  text-text-h">Factory Reset</h4>
                                    <p class="text-xs text-text-m mt-1 max-w-xl">Wipes all configuration, logs, and authentication keys. The application will restart in setup mode.</p>
                                </div>
                                <BaseButton variant="danger" @click="resetPassword = ''; resetError = ''; showResetModal = true" class="shrink-0">
                                    Reset System
                                </BaseButton>
                            </div>
                        </template>
                    </BaseCard>
                </div>

            </div>
        </div>

        <BaseModal :show="showPasswordModal" title="Update Password" @close="showPasswordModal = false">
            <template #icon>
                <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z"></path></svg>
            </template>
            <form @submit.prevent="submitPasswordChange" class="space-y-4">
            <BaseInput v-model="pwdData.current" label="Current Password" type="password" required autofocus />
                <BaseInput v-model="pwdData.new" label="New Password" type="password" required />
                <BaseInput v-model="pwdData.confirmNew" label="Confirm New Password" type="password" required />
                
                <div v-if="pwdError" class="text-xs  text-danger-text bg-danger-bg p-2.5 rounded-md border border-danger-border">{{ pwdError }}</div>
                
                <div class="pt-4 flex justify-end gap-3">
                    <BaseButton variant="ghost" @click="showPasswordModal = false">Cancel</BaseButton>
                    <BaseButton variant="primary" type="submit" :disabled="pwdLoading">
                        {{ pwdLoading ? 'Updating...' : 'Update Password' }}
                    </BaseButton>
                </div>
            </form>
        </BaseModal>

        <BaseModal :show="showResetModal" title="Confirm Factory Reset" danger @close="showResetModal = false">
            <template #icon>
                <svg class="w-6 h-6 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
            </template>
            <p class="text-sm text-text-m mb-5">This action is irreversible. All events, sensors, and configurations will be permanently deleted. Enter your master password to confirm.</p>
            
            <form @submit.prevent="submitFactoryReset" class="space-y-4">
                <BaseInput 
                    v-model="resetPassword" 
                    type="password" 
                    placeholder="Master Password" 
                    required 
                    autofocus
                    inputClass="!border-danger-border !focus:border-danger-main"
                />
                
                <div v-if="resetError" class="text-xs  text-danger-text bg-danger-bg p-2.5 rounded-md border border-danger-border text-center">{{ resetError }}</div>
                
                <div class="pt-4 flex justify-end gap-3">
                    <BaseButton variant="ghost" @click="showResetModal = false">Cancel</BaseButton>
                    <BaseButton variant="danger" type="submit" :disabled="resetLoading">
                        {{ resetLoading ? 'Wiping...' : 'Destroy Data' }}
                    </BaseButton>
                </div>
            </form>
        </BaseModal>

    </div>
</template>