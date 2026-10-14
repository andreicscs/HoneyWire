<script setup>
import BaseNumberStepper from '../ui/forms/BaseNumberStepper.vue'
import BaseCard from '../ui/layout/BaseCard.vue'
import BaseDivider from '../ui/layout/BaseDivider.vue'
import { settingsForm, disabledInputClass } from '../../views/settingsStyles.ts'

const settings = defineModel({ type: Object, required: true })
</script>

<template>
    <BaseCard title="Database Retention Policies">
        <div :class="settingsForm.sectionStack">
            <div>
                <h3 :class="settingsForm.sectionTitle">Events Data</h3>
                <div :class="settingsForm.fieldStack">
                    <BaseNumberStepper
                        v-model="settings.autoArchiveDays"
                        label="Auto-Archive Events"
                        description="Move events from the Live Queue to the Archive automatically."
                        :min="0" :max="365"
                        :suffix="settings.autoArchiveDays === 1 ? 'Day' : 'Days'"
                        :class="settings.autoArchiveDays === 0 ? disabledInputClass : ''"
                    />
                    <BaseNumberStepper
                        v-model="settings.autoPurgeDays"
                        label="Auto-Purge Archive"
                        description="Permanently delete archived events from the SQLite database."
                        :min="0" :max="365"
                        :suffix="settings.autoPurgeDays === 1 ? 'Day' : 'Days'"
                        :class="settings.autoPurgeDays === 0 ? disabledInputClass : ''"
                    />
                </div>
            </div>

            <BaseDivider />

            <div>
                <h3 :class="settingsForm.sectionTitle">Uptime Data</h3>
                <BaseNumberStepper
                    v-model="settings.autoPurgeHeartbeatsDays"
                    label="Auto-Purge Uptime Data"
                    description="Permanently delete historical uptime state changes from the database."
                    :min="0" :max="365"
                    :suffix="settings.autoPurgeHeartbeatsDays === 1 ? 'Day' : 'Days'"
                    :class="settings.autoPurgeHeartbeatsDays === 0 ? disabledInputClass : ''"
                />
            </div>
        </div>
    </BaseCard>
</template>
