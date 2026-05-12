<script setup>
import { ref, watch, computed } from 'vue'
import { useConfig } from '../api/useConfig'

const { config, patchConfig } = useConfig()
const activeTab = ref('general')

const settings = ref({
    hubEndpoint: window.location.origin,
    hubKey: '',
    autoArchiveDays: 0,
    autoPurgeDays: 0,
    webhookType: 'ntfy',
    webhookUrl: '',
    webhookEvents: [],
    siemAddress: '',
    siemProtocol: 'tcp'
})

const initialSettings = ref(null)

watch(() => config.isLoaded, (loaded) => {
    if (loaded) {
        const loadedSettings = {
            hubEndpoint: config.hubEndpoint || window.location.origin,
            hubKey: config.hubKey || '',
            autoArchiveDays: config.autoArchiveDays !== undefined ? config.autoArchiveDays : 0,
            autoPurgeDays: config.autoPurgeDays !== undefined ? config.autoPurgeDays : 0,
            webhookType: config.webhookType || 'ntfy',
            webhookUrl: config.webhookUrl || '',
            webhookEvents: config.webhookEvents && config.webhookEvents.length > 0 
                ? [...config.webhookEvents] 
                : ['critical', 'high', 'medium', 'low', 'info'],
            siemAddress: config.siemAddress || '',
            siemProtocol: config.siemProtocol || 'tcp'
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

const saveSettings = async () => {
    isSaving.value = true
    saveMessage.value = ''
    
    const success = await patchConfig(settings.value)
    
    isSaving.value = false
    if (success) {
        initialSettings.value = JSON.parse(JSON.stringify(settings.value))
        saveMessage.value = 'Configuration saved successfully.'
        setTimeout(() => saveMessage.value = '', 3000)
    } else {
        saveMessage.value = 'Failed to save configuration. Check console and server logs.'
    }
}

const regenerateKey = () => {
    if(confirm("Regenerating the Hub Key will immediately disconnect all active sensors. You must save changes to apply this. Continue?")) {
        const array = new Uint8Array(16)
        crypto.getRandomValues(array)
        settings.value.hubKey = 'hw_sk_' + Array.from(array).map(b => b.toString(16).padStart(2, '0')).join('')
    }
}

const adjustDays = (field, delta, min, max) => {
    let val = Number(settings.value[field]) + delta
    if (val < min) val = min
    if (val > max) val = max
    settings.value[field] = val
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
    try {
        const res = await fetch('/api/v1/system/password', {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ current_password: pwdData.value.current, new_password: pwdData.value.new })
        })
        if (res.ok) window.location.reload()
        else if (res.status === 401) pwdError.value = "Incorrect current password."
        else {
            const err = await res.text()
            pwdError.value = err || "Failed to update password."
        }
    } catch (e) {
        pwdError.value = "Network error."
    } finally {
        pwdLoading.value = false
    }
}

const submitFactoryReset = async () => {
    if (!resetPassword.value) {
        resetError.value = "Master password is required."
        return
    }
    resetLoading.value = true
    resetError.value = ''
    try {
        const res = await fetch('/api/v1/system/reset', { 
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ password: resetPassword.value })
        })
        if (res.ok) window.location.reload()
        else if (res.status === 401) resetError.value = "Incorrect password."
        else resetError.value = "Factory reset failed."
    } catch (e) {
        resetError.value = "Network error."
    } finally {
        resetLoading.value = false
    }
}
</script>

<template>
    <div class="h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6 transition-colors duration-200">
        <div class="mb-6 shrink-0 mt-4 sm:mt-6 flex justify-between items-end">
            <div>
                <h1 class="text-2xl font-bold text-text-main">System Settings</h1>
                <p class="text-sm text-text-muted mt-1 max-w-3xl">Manage Hub configuration, retention policies, and push notifications.</p>
            </div>
            <div class="flex items-center gap-4">
                
                <span v-if="saveMessage" class="text-xs font-bold text-success-main animate-pulse hidden sm:block">{{ saveMessage }}</span>
                <span v-else-if="hasUnsavedChanges" class="text-xs font-bold text-medium animate-pulse hidden sm:flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full bg-medium inline-block"></span> Unsaved changes
                </span>

                <button @click="saveSettings" :disabled="isSaving || !hasUnsavedChanges"
                        class="px-4 py-2 rounded-md text-xs font-bold uppercase tracking-wider transition-all shadow-sm active:scale-95 border outline-none focus:ring-2 focus:ring-focus-ring"
                        :class="hasUnsavedChanges ? 'bg-primary-main text-primary-text border-primary-main hover:bg-primary-hover' : 'bg-disabled-bg text-disabled-text border-disabled-border shadow-none cursor-not-allowed active:scale-100'">
                    {{ isSaving ? 'Saving...' : 'Save Changes' }}
                </button>
            </div>
        </div>

        <div class="flex flex-col md:flex-row gap-6 pb-10 flex-1 min-h-0">
            <nav class="w-full md:w-56 shrink-0 flex flex-col gap-2">
                <button @click="activeTab = 'general'" 
                        class="w-full text-left px-4 py-2.5 rounded-lg text-sm transition-all flex items-center gap-3 border"
                        :class="activeTab === 'general' ? 'bg-secondary-selected text-text-main font-bold shadow-sm border-secondary-border' : 'bg-secondary-main font-medium text-secondary-text hover:bg-secondary-hover hover:text-text-main border-secondary-border/50'">
                    <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>
                    Hub Configuration
                </button>
                <button @click="activeTab = 'data'" 
                        class="w-full text-left px-4 py-2.5 rounded-lg text-sm transition-all flex items-center gap-3 border"
                        :class="activeTab === 'data' ? 'bg-secondary-selected text-text-main font-bold shadow-sm border-secondary-border' : 'bg-secondary-main font-medium text-secondary-text hover:bg-secondary-hover hover:text-text-main border-secondary-border/50'">
                    <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"></path></svg>
                    Data Retention
                </button>
                <button @click="activeTab = 'alerts'" 
                        class="w-full text-left px-4 py-2.5 rounded-lg text-sm transition-all flex items-center gap-3 border"
                        :class="activeTab === 'alerts' ? 'bg-secondary-selected text-text-main font-bold shadow-sm border-secondary-border' : 'bg-secondary-main font-medium text-secondary-text hover:bg-secondary-hover hover:text-text-main border-secondary-border/50'">
                    <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"></path></svg>
                    Push Notifications
                </button>
                <button @click="activeTab = 'siem'" 
                        class="w-full text-left px-4 py-2.5 rounded-lg text-sm transition-all flex items-center gap-3 border"
                        :class="activeTab === 'siem' ? 'bg-secondary-selected text-text-main font-bold shadow-sm border-secondary-border' : 'bg-secondary-main font-medium text-secondary-text hover:bg-secondary-hover hover:text-text-main border-secondary-border/50'">
                    <svg class="w-5 h-5 shrink-0" fill="currentColor" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" xml:space="preserve">
                        <path d="M77.2,56.2h-3.7c-1,0-1.8,1-1.8,1.8v12.3c0,1-0.9,1.8-1.8,1.8H29.3c-1,0-1.8-0.9-1.8-1.8V58 c0-0.9-0.9-1.8-1.8-1.8h-3.7c-1,0-1.8,1-1.8,1.8v16.6c0,2.7,2.2,4.9,4.9,4.9h49.1c2.7,0,4.9-2.2,4.9-4.9V58 C79,57.1,78.2,56.2,77.2,56.2z M50.8,21c-0.7-0.7-1.8-0.7-2.6,0L31.6,37.6c-0.7,0.7-0.7,1.8,0,2.6l2.6,2.6c0.7,0.7,1.8,0.7,2.6,0 l6.9-6.9c0.7-0.7,2.2-0.2,2.2,0.9v26c0,1,0.7,1.8,1.7,1.8h3.7c1,0,2-1,2-1.8V36.9c0-1.1,1.2-1.6,2.1-0.9l6.9,6.9 c0.7,0.7,1.8,0.7,2.6,0l2.6-2.6c0.7-0.7,0.7-1.8,0-2.6C67.3,37.7,50.8,21,50.8,21z"></path>
                    </svg>
                    SIEM Forwarding
                </button>
                <div class="h-px bg-border-default/50 my-2 mx-4"></div>
                <button @click="activeTab = 'security'" 
                        class="w-full text-left px-4 py-2.5 rounded-lg text-sm transition-all flex items-center gap-3 border"
                        :class="activeTab === 'security' ? 'bg-danger-bg-hover text-danger-text font-bold shadow-sm border-danger-border' : 'bg-bg-surface font-medium text-text-muted hover:bg-danger-bg-subtle hover:text-danger-text border-border-default/50'">
                    <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path></svg>
                    Security & Access
                </button>
            </nav>

            <div class="flex-1 overflow-y-auto custom-scroll pr-2 space-y-6">
                <div v-show="activeTab === 'general'" class="space-y-6">
                    <div class="bg-bg-surface border border-border-default rounded-lg p-5 md:p-6 shadow-sm transition-colors">
                        <h3 class="text-base font-bold text-text-main mb-5">Network Configuration</h3>
                        <div class="space-y-6">
                            <div>
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-2">Hub Endpoint URL</label>
                                <p class="text-xs text-text-muted mb-3">The publicly accessible URL or IP where sensors will send their telemetry.</p>
                                <input type="text" v-model="settings.hubEndpoint" 
                                       class="w-full max-w-md px-3 py-2 rounded-md bg-input-bg border border-input-border text-sm mono text-text-main focus:outline-none focus:ring-1 focus:border-highlight-border focus:ring-focus-ring shadow-inner transition-colors">
                            </div>
                            <div class="pt-5 border-t border-border-default/50">
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-2">Hub Secret Key</label>
                                <p class="text-xs text-text-muted mb-3">The shared secret required by sensors to authenticate with the Hub API.</p>
                                <div class="flex gap-3 items-center flex-wrap sm:flex-nowrap">
                                    <input type="text" v-model="settings.hubKey"
                                           class="flex-1 w-full max-w-md px-3 py-2 rounded-md bg-input-bg border border-input-border text-sm mono text-text-main focus:outline-none focus:ring-1 focus:border-highlight-border focus:ring-focus-ring shadow-inner transition-colors">
                                    <button @click="regenerateKey" 
                                            class="px-4 py-2 rounded-md bg-secondary-main border border-secondary-border text-secondary-text text-xs font-bold hover:bg-secondary-hover hover:text-text-main transition-colors shadow-sm whitespace-nowrap outline-none focus:ring-2 focus:ring-focus-ring">
                                        Regenerate Key
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <div v-show="activeTab === 'data'" class="space-y-6">
                    <div class="bg-bg-surface border border-border-default rounded-lg p-5 md:p-6 shadow-sm transition-colors">
                        <h3 class="text-base font-bold text-text-main mb-5">Database Retention Policies</h3>
                        <div class="space-y-6">
                            <div class="flex items-center justify-between gap-4 max-w-2xl">
                                <div>
                                    <label class="block text-[13px] font-bold text-text-main">Auto-Archive Events</label>
                                    <p class="text-xs text-text-muted mt-1">Move events from the Live Queue to the Archive automatically.</p>
                                </div>
                                <div class="flex items-center gap-3">
                                    <div class="flex items-center rounded-md border border-input-border overflow-hidden bg-input-bg shadow-inner focus-within:ring-1 focus-within:ring-focus-ring focus-within:border-highlight-border transition-colors">
                                        <button @click="adjustDays('autoArchiveDays', -1, 0, 90)" class="px-3 py-1.5 text-text-muted hover:bg-button-hover transition-colors font-bold select-none outline-none">-</button>
                                        <input type="number" v-model="settings.autoArchiveDays" min="0" max="90"
                                               class="w-12 text-center text-sm mono font-bold bg-transparent border-none focus:outline-none focus:ring-0 text-text-main hide-arrows p-0" />
                                        <button @click="adjustDays('autoArchiveDays', 1, 0, 90)" class="px-3 py-1.5 text-text-muted hover:bg-button-hover transition-colors font-bold select-none outline-none">+</button>
                                    </div>
                                    <span class="text-xs font-bold uppercase tracking-wider text-text-muted w-10">Days</span>
                                </div>
                            </div>
                            <div class="h-px w-full bg-border-default/50"></div>
                            <div class="flex items-center justify-between gap-4 max-w-2xl">
                                <div>
                                    <label class="block text-[13px] font-bold text-text-main">Auto-Purge Archive</label>
                                    <p class="text-xs text-text-muted mt-1">Permanently delete archived events from the SQLite database.</p>
                                </div>
                                <div class="flex items-center gap-3">
                                    <div class="flex items-center rounded-md border border-input-border overflow-hidden bg-input-bg shadow-inner focus-within:ring-1 focus-within:ring-focus-ring focus-within:border-highlight-border transition-colors">
                                        <button @click="adjustDays('autoPurgeDays', -1, 0, 365)" class="px-3 py-1.5 text-text-muted hover:bg-button-hover transition-colors font-bold select-none outline-none">-</button>
                                        <input type="number" v-model="settings.autoPurgeDays" min="0" max="365"
                                               class="w-12 text-center text-sm mono font-bold bg-transparent border-none focus:outline-none focus:ring-0 text-text-main hide-arrows p-0" />
                                        <button @click="adjustDays('autoPurgeDays', 1, 0, 365)" class="px-3 py-1.5 text-text-muted hover:bg-button-hover transition-colors font-bold select-none outline-none">+</button>
                                    </div>
                                    <span class="text-xs font-bold uppercase tracking-wider text-text-muted w-10">Days</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <div v-show="activeTab === 'alerts'" class="space-y-6">
                    <div class="bg-bg-surface border border-border-default rounded-lg p-5 md:p-6 shadow-sm transition-colors">
                        <div class="flex items-center gap-3 mb-6">
                            <h3 class="text-base font-bold text-text-main">Push Notifications</h3>
                        </div>
                        <div class="space-y-6">
                            <div>
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-3">Service Provider</label>
                                <div class="flex flex-wrap gap-2.5">
                                    <button v-for="provider in ['ntfy', 'gotify', 'discord', 'slack']" :key="provider"
                                            @click="settings.webhookType = provider"
                                            class="px-3.5 py-1.5 rounded-md text-[11px] font-bold uppercase tracking-wider border transition-all flex items-center justify-center outline-none focus:ring-2 focus:ring-focus-ring"
                                            :class="settings.webhookType === provider 
                                                ? 'bg-primary-selected text-primary-text border-primary-selected shadow-sm' 
                                                : 'bg-secondary-main border-secondary-border text-secondary-text hover:bg-secondary-hover hover:text-text-main'">
                                        {{ provider }}
                                    </button>
                                </div>
                            </div>
                            <div class="pt-5 border-t border-border-default/50">
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-2">Target URL</label>
                                <p class="text-xs text-text-muted mb-3">
                                    <span v-if="settings.webhookType === 'gotify'">Enter your Gotify server URL and append the App Token (e.g., <code>https://gotify.domain.com/message?token=XYZ</code>).</span>
                                    <span v-else-if="settings.webhookType === 'ntfy'">Enter your self-hosted or public Ntfy topic URL (e.g., <code>https://ntfy.sh/my_alerts</code>).</span>
                                    <span v-else>Paste the incoming Webhook URL provided by {{ settings.webhookType === 'discord' ? 'Discord' : 'Slack' }}.</span>
                                </p>
                                <input type="url" v-model="settings.webhookUrl" placeholder="https://..."
                                       class="w-full max-w-xl px-4 py-2 rounded-md bg-input-bg border border-input-border text-sm mono text-text-main focus:outline-none focus:ring-1 focus:border-highlight-border focus:ring-focus-ring shadow-inner transition-colors placeholder:text-text-muted/50">
                            </div>
                            <div class="pt-5 border-t border-border-default/50">
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-3">Trigger Severities</label>
                                <div class="flex flex-wrap gap-2.5">
                                    <button v-for="sev in ['critical', 'high', 'medium', 'low', 'info']" :key="sev"
                                            @click="toggleSeverity(sev)"
                                            class="px-3.5 py-1.5 rounded-md text-[11px] font-bold uppercase tracking-wider transition-all border outline-none select-none hover:opacity-80 focus:ring-2 focus:ring-focus-ring"
                                            :style="getSeverityStyle(sev, settings.webhookEvents.includes(sev))">
                                        {{ sev }}
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <div v-show="activeTab === 'siem'" class="space-y-6">
                    <div class="bg-bg-surface border border-border-default rounded-lg p-5 md:p-6 shadow-sm transition-colors">
                        <div class="flex items-center gap-3 mb-6">
                            <h3 class="text-base font-bold text-text-main">SIEM Forwarding</h3>
                        </div>
                        <div class="space-y-6">
                            <div>
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-2">Server Address</label>
                                <p class="text-xs text-text-muted mb-3">Forward syslog events to your SIEM (e.g., <code>elk.example.com:514</code>).</p>
                                <input type="text" v-model="settings.siemAddress" placeholder="host:port"
                                       class="w-full max-w-xl px-4 py-2 rounded-md bg-input-bg border border-input-border text-sm mono text-text-main focus:outline-none focus:ring-1 focus:border-highlight-border focus:ring-focus-ring shadow-inner transition-colors placeholder:text-text-muted/50">
                            </div>
                            <div class="pt-5 border-t border-border-default/50">
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-3">Protocol</label>
                                <div class="flex flex-wrap gap-2.5">
                                    <button v-for="proto in ['tcp', 'udp']" :key="proto"
                                            @click="settings.siemProtocol = proto"
                                            class="px-3.5 py-1.5 rounded-md text-[11px] font-bold uppercase tracking-wider border transition-all flex items-center justify-center outline-none focus:ring-2 focus:ring-focus-ring"
                                            :class="settings.siemProtocol === proto 
                                                ? 'bg-primary-selected text-primary-text border-primary-selected shadow-sm' 
                                                : 'bg-secondary-main border-secondary-border text-secondary-text hover:bg-secondary-hover hover:text-text-main'">
                                        {{ proto }}
                                    </button>
                                </div>
                            </div>
                            <div class="pt-5 border-t border-border-default/50">
                                <p class="text-xs text-text-muted">Events are sent in RFC3164 syslog format. Leave blank to disable SIEM forwarding.</p>
                            </div>
                        </div>
                    </div>
                </div>

                <div v-show="activeTab === 'security'" class="space-y-6">
                    <div class="bg-bg-surface border border-border-default rounded-lg p-5 md:p-6 shadow-sm transition-colors">
                        <h3 class="text-base font-bold text-text-main mb-4">Authentication</h3>
                        <div>
                            <p class="text-sm text-text-muted mb-4 max-w-2xl">Update the master password used to access this dashboard. You will be logged out immediately upon changing this.</p>
                            <button @click="pwdData = {current:'', new:'', confirmNew:''}; pwdError = ''; showPasswordModal = true" 
                                    class="px-4 py-2 rounded-md bg-secondary-main border border-secondary-border text-secondary-text text-sm font-bold hover:bg-secondary-hover hover:text-text-main transition-colors shadow-sm outline-none focus:ring-2 focus:ring-focus-ring">
                                Change Master Password
                            </button>
                        </div>
                    </div>

                    <div class="bg-danger-bg border border-danger-border rounded-lg p-5 md:p-6 shadow-sm transition-colors">
                        <h3 class="text-base font-bold text-danger-text mb-4 flex items-center gap-2">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
                            Danger Zone
                        </h3>
                        <div class="space-y-4 mt-2">
                            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-bg-surface border border-danger-border p-5 rounded-lg shadow-sm transition-colors">
                                <div>
                                    <h4 class="text-sm font-bold text-text-main">Factory Reset</h4>
                                    <p class="text-xs text-text-muted mt-1 max-w-xl">Wipes all configuration, logs, and authentication keys. The application will restart in setup mode.</p>
                                </div>
                                <button @click="resetPassword = ''; resetError = ''; showResetModal = true" 
                                        class="shrink-0 px-4 py-2 rounded-md bg-danger-main border border-danger-main hover:bg-danger-hover text-white text-xs font-bold uppercase tracking-wider transition-colors shadow-sm outline-none focus:ring-2 focus:ring-danger-border">
                                    Reset System
                                </button>
                            </div>
                        </div>
                    </div>
                </div>

            </div>
        </div>

        <Teleport to="body">
            <transition enter-active-class="transition duration-200 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-150 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <div v-if="showPasswordModal" class="fixed inset-0 z-50 flex justify-center items-center p-4 bg-black/60 backdrop-blur-sm" @click.self="showPasswordModal = false">
                    <div class="bg-bg-surface w-full max-w-sm rounded-lg shadow-2xl border border-border-default p-6 transform transition-all">
                        
                        <div class="flex items-center gap-3 mb-5 text-text-main">
                            <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z"></path></svg>
                            <h3 class="text-lg font-bold">Update Password</h3>
                        </div>

                        <form @submit.prevent="submitPasswordChange" class="space-y-4">
                            <div>
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-1.5">Current Password</label>
                                <input type="password" v-model="pwdData.current" class="w-full px-3 py-2 rounded-md bg-input-bg border border-input-border text-sm mono text-text-main focus:outline-none focus:ring-1 focus:border-highlight-border focus:ring-focus-ring shadow-inner transition-all" required autofocus>
                            </div>
                            
                            <div class="pt-2">
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-1.5">New Password</label>
                                <input type="password" v-model="pwdData.new" class="w-full px-3 py-2 rounded-md bg-input-bg border border-input-border text-sm mono text-text-main focus:outline-none focus:ring-1 focus:border-highlight-border focus:ring-focus-ring shadow-inner transition-all" required>
                            </div>

                            <div>
                                <label class="block text-[11px] font-bold text-text-muted uppercase tracking-wider mb-1.5">Confirm New Password</label>
                                <input type="password" v-model="pwdData.confirmNew" class="w-full px-3 py-2 rounded-md bg-input-bg border border-input-border text-sm mono text-text-main focus:outline-none focus:ring-1 focus:border-highlight-border focus:ring-focus-ring shadow-inner transition-all" required>
                            </div>

                            <div v-if="pwdError" class="text-xs font-bold text-danger-text bg-danger-bg-subtle p-2.5 rounded-md border border-danger-border">{{ pwdError }}</div>
                            
                            <div class="pt-4 flex justify-end gap-3">
                                <button type="button" @click="showPasswordModal = false" class="px-4 py-2 text-sm font-medium text-secondary-text hover:text-text-main transition-colors outline-none focus:ring-2 focus:ring-focus-ring rounded-md">Cancel</button>
                                <button type="submit" :disabled="pwdLoading" 
                                        class="px-4 py-2 rounded-md text-sm font-bold shadow-sm transition-all active:scale-95 border outline-none focus:ring-2 focus:ring-focus-ring"
                                        :class="pwdLoading ? 'bg-disabled-bg text-disabled-text border-disabled-border cursor-not-allowed active:scale-100 shadow-none' : 'bg-primary-main text-primary-text border-primary-main hover:bg-primary-hover'">
                                    {{ pwdLoading ? 'Updating...' : 'Update Password' }}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </transition>
        </Teleport>

        <Teleport to="body">
            <transition enter-active-class="transition duration-200 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-150 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <div v-if="showResetModal" class="fixed inset-0 z-50 flex justify-center items-center p-4 bg-black/60 backdrop-blur-sm" @click.self="showResetModal = false">
                    <div class="bg-bg-surface w-full max-w-sm rounded-lg shadow-2xl border border-danger-border p-6 transform transition-all">
                        <div class="flex items-center gap-3 mb-4 text-danger-text">
                            <svg class="w-6 h-6 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
                            <h3 class="text-lg font-bold">Confirm Factory Reset</h3>
                        </div>
                        <p class="text-sm text-text-muted mb-5">This action is irreversible. All events, sensors, and configurations will be permanently deleted. Enter your master password to confirm.</p>
                        
                        <form @submit.prevent="submitFactoryReset" class="space-y-4">
                            <div>
                                <input type="password" v-model="resetPassword" placeholder="Master Password" class="w-full px-3 py-2 rounded-md bg-input-bg border border-input-border text-sm mono text-text-main focus:outline-none focus:ring-1 focus:border-danger-border focus:ring-danger-border/50 shadow-inner transition-all" required autofocus>
                            </div>
                            
                            <div v-if="resetError" class="text-xs font-bold text-danger-text bg-danger-bg-subtle p-2.5 rounded-md border border-danger-border text-center">{{ resetError }}</div>
                            
                            <div class="pt-4 flex justify-end gap-3">
                                <button type="button" @click="showResetModal = false" class="px-4 py-2 text-sm font-medium text-secondary-text hover:text-text-main transition-colors outline-none focus:ring-2 focus:ring-focus-ring rounded-md">Cancel</button>
                                <button type="submit" :disabled="resetLoading" 
                                        class="px-4 py-2 rounded-md text-sm font-bold shadow-sm transition-colors active:scale-95 border outline-none focus:ring-2 focus:ring-danger-border/50"
                                        :class="resetLoading ? 'bg-disabled-bg text-disabled-text border-disabled-border cursor-not-allowed shadow-none active:scale-100' : 'bg-danger-main text-white border-danger-main hover:bg-danger-hover'">
                                    {{ resetLoading ? 'Wiping...' : 'Destroy Data' }}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </transition>
        </Teleport>

    </div>
</template>

<style scoped>
.hide-arrows::-webkit-outer-spin-button,
.hide-arrows::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.hide-arrows[type=number] {
  -moz-appearance: textfield;
}
</style>