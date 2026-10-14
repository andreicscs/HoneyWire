<script setup>
import BaseInput from '../ui/forms/BaseInput.vue'
import BaseRadioGroup from '../ui/forms/BaseRadioGroup.vue'
import BaseCard from '../ui/layout/BaseCard.vue'
import BaseDivider from '../ui/layout/BaseDivider.vue'
import { settingsForm } from '../../views/settingsStyles.ts'

const settings = defineModel({ type: Object, required: true })
</script>

<template>
    <BaseCard title="SIEM Forwarding">
        <div :class="settingsForm.wide">
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
            description="Events are sent in RFC5424 syslog format. Leave blank to disable SIEM forwarding."
        />
    </BaseCard>
</template>
