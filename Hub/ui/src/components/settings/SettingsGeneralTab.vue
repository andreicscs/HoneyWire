<script setup>
import { ref } from 'vue'
import { useAppStore } from '../../stores/System/app'
import { useFleetStore } from '../../stores/Fleet/fleet'
import BaseInput from '../ui/forms/BaseInput.vue'
import BaseNumberStepper from '../ui/forms/BaseNumberStepper.vue'
import BaseCard from '../ui/layout/BaseCard.vue'
import {
    settingsForm,
    settingsWarning,
    settingsUpdateCheck,
    updateFeedbackVariantClass,
} from '../../views/settingsStyles.ts'

const settings = defineModel({ type: Object, required: true })

const appStore = useAppStore()
const fleetStore = useFleetStore()

const isCheckingUpdates = ref(false)
const updateCheckFeedback = ref(null)
let feedbackTimeout = null

const checkForUpdatesManual = async () => {
    if (isCheckingUpdates.value) return
    isCheckingUpdates.value = true
    if (feedbackTimeout) clearTimeout(feedbackTimeout)
    updateCheckFeedback.value = null

    const [result] = await Promise.all([
        appStore.checkSystemUpdatesManual().then(async res => {
            if (res && res.success) {
                await fleetStore.fetchFleet()
            }
            return res
        }),
        new Promise(resolve => setTimeout(resolve, 600))
    ])
    isCheckingUpdates.value = false

    if (result && result.success) {
        if (result.data?.warning) {
            updateCheckFeedback.value = {
                type: 'warning',
                message: result.data.warning
            }
        } else {
            const hasHubUpdate = appStore.updateAvailable
            const hasWizardUpdate = appStore.wizardUpdateAvailable
            const hasSensorUpdate = fleetStore.hasUpdatesAvailable

            if (hasHubUpdate && hasWizardUpdate && hasSensorUpdate) {
                updateCheckFeedback.value = {
                    type: 'info',
                    message: 'New updates available for Hub, Wizard, and Fleet Sensors!'
                }
            } else if (hasHubUpdate && hasWizardUpdate) {
                updateCheckFeedback.value = {
                    type: 'info',
                    message: `New Hub (${appStore.latestVersion}) and Wizard CLI (${appStore.latestWizardVersion}) available!`
                }
            } else if (hasHubUpdate && hasSensorUpdate) {
                updateCheckFeedback.value = {
                    type: 'info',
                    message: `New Hub (${appStore.latestVersion}) and Fleet Sensor updates available!`
                }
            } else if (hasWizardUpdate && hasSensorUpdate) {
                updateCheckFeedback.value = {
                    type: 'info',
                    message: `New Wizard CLI (${appStore.latestWizardVersion}) and Fleet Sensor updates available!`
                }
            } else if (hasHubUpdate) {
                updateCheckFeedback.value = {
                    type: 'info',
                    message: `New Hub version ${appStore.latestVersion} available!`
                }
            } else if (hasWizardUpdate) {
                updateCheckFeedback.value = {
                    type: 'info',
                    message: `New Wizard CLI version ${appStore.latestWizardVersion} available!`
                }
            } else if (hasSensorUpdate) {
                updateCheckFeedback.value = {
                    type: 'info',
                    message: 'New sensor updates available for your fleet!'
                }
            } else {
                updateCheckFeedback.value = {
                    type: 'success',
                    message: 'All system and sensor updates are up to date.'
                }
            }
        }
    } else {
        updateCheckFeedback.value = {
            type: 'error',
            message: (result && result.error) || 'Unable to check updates. Please try again.'
        }
    }

    feedbackTimeout = setTimeout(() => {
        updateCheckFeedback.value = null
    }, 5000)
}
</script>

<template>
    <BaseCard title="Network Configuration">
        <div :class="settingsForm.narrowStack">
            <BaseInput
                v-model="settings.hubEndpoint"
                label="Hub Endpoint URL"
                description="The publicly accessible URL or IP where sensors will send their telemetry."
            />

            <BaseInput
                v-model="settings.registryUrl"
                label="Sensor Registry URL"
                description="The public registry URL from which to download sensor updates."
            />

            <div v-if="(settings.hubEndpoint && settings.hubEndpoint.startsWith('http://')) || (settings.registryUrl && settings.registryUrl.startsWith('http://'))" :class="settingsWarning.box" data-testid="unsafe-protocol-warning">
                <svg :class="settingsWarning.icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
                <div :class="settingsWarning.body">
                    <p :class="settingsWarning.title">Unsafe Protocol Detected</p>
                    <p :class="settingsWarning.desc">Using HTTP allows unencrypted man-in-the-middle attacks. It is highly recommended to use HTTPS for all URLs to protect sensor telemetry and manifests.</p>
                </div>
            </div>
        </div>
    </BaseCard>

    <BaseCard title="System Updates" class="mt-6">
        <div :class="settingsForm.narrowStack">
            <BaseNumberStepper
                v-model="settings.updateCheckIntervalHours"
                label="Update Check Interval (Hours)"
                :min="0" :max="720"
                description="How often the hub checks for updates. Set to 0 to disable automatic checks."
            />

            <div :class="settingsUpdateCheck.divider">
                <div :class="settingsUpdateCheck.row">
                    <div :class="settingsUpdateCheck.textWrap">
                        <label :class="settingsUpdateCheck.label">Manual Check</label>
                        <p :class="settingsUpdateCheck.desc">Check for Hub, Wizard, and Sensor updates on demand.</p>
                    </div>
                    <button
                        type="button"
                        data-testid="update-check-button"
                        :disabled="isCheckingUpdates"
                        @click="checkForUpdatesManual"
                        :class="settingsUpdateCheck.button"
                    >
                        <svg v-if="isCheckingUpdates" :class="settingsUpdateCheck.spinnerIcon" fill="none" viewBox="0 0 24 24">
                            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        <svg v-else :class="settingsUpdateCheck.refreshIcon" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                        <span>{{ isCheckingUpdates ? 'Checking...' : 'Check now' }}</span>
                    </button>
                </div>

                <div v-if="updateCheckFeedback" :class="[settingsUpdateCheck.feedbackBase, updateFeedbackVariantClass(updateCheckFeedback.type)]">
                    <svg v-if="updateCheckFeedback.type === 'success'" :class="settingsUpdateCheck.feedbackIcon.success" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"></path></svg>
                    <svg v-else-if="updateCheckFeedback.type === 'info'" :class="settingsUpdateCheck.feedbackIcon.info" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                    <svg v-else-if="updateCheckFeedback.type === 'warning'" :class="settingsUpdateCheck.feedbackIcon.warning" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
                    <svg v-else :class="settingsUpdateCheck.feedbackIcon.error" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                    <span :class="settingsUpdateCheck.feedbackText">{{ updateCheckFeedback.message }}</span>
                </div>
            </div>
        </div>
    </BaseCard>

    <BaseCard title="Event Filtering" class="mt-6">
        <div :class="settingsForm.narrow">
            <BaseInput
                v-model="settings.whitelistedSources"
                label="Whitelisted Event Sources"
                description="Comma-separated list of IPs or source names to ignore. Events originating from these sources will be completely dropped and not logged."
                placeholder="e.g. 192.168.1.1, vulnerability_scanner"
            />
        </div>
    </BaseCard>
</template>
