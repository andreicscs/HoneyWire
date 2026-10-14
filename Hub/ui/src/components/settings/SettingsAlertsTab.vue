<script setup>
import BaseInput from '../ui/forms/BaseInput.vue'
import BaseRadioGroup from '../ui/forms/BaseRadioGroup.vue'
import BaseCard from '../ui/layout/BaseCard.vue'
import BaseDivider from '../ui/layout/BaseDivider.vue'
import { settingsForm, settingsSeverity } from '../../views/settingsStyles.ts'

const settings = defineModel({ type: Object, required: true })

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
</script>

<template>
    <BaseCard title="Push Notifications">
        <BaseRadioGroup
            v-model="settings.webhookType"
            label="Service Provider"
            :options="['ntfy', 'gotify', 'discord', 'slack']"
        />

        <BaseDivider />

        <div :class="settingsForm.wide">
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
            <label :class="settingsSeverity.label">Trigger Severities</label>
            <div :class="settingsSeverity.row">
                <button v-for="sev in ['critical', 'high', 'medium', 'low', 'info']" :key="sev"
                        data-testid="severity-trigger" :data-severity="sev"
                        @click="toggleSeverity(sev)"
                        :class="settingsSeverity.button"
                        :style="getSeverityStyle(sev, settings.webhookEvents.includes(sev))">
                    {{ sev }}
                </button>
            </div>
        </div>
    </BaseCard>
</template>
