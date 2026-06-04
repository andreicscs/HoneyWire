<script setup lang="ts">
import { ref } from 'vue'
import BaseModal from '../ui/feedback/BaseModal.vue'
import BaseButton from '../ui/forms/BaseButton.vue'
import BaseInput from '../ui/forms/BaseInput.vue'
import { useClipboard } from '../../utils/useClipboard.js'
import { useFleetStore } from '../../stores/Fleet/fleet.ts'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const fleetStore = useFleetStore()
const { copiedStates, handleCopy } = useClipboard() as any

const step = ref(1)
const form = ref({ alias: '' })
const nodeKey = ref('')

const submit = async () => {
    if (!form.value.alias) return
    try {
        const result = await fleetStore.createNode(form.value.alias)
        nodeKey.value = result.apiKey
        step.value = 2
    } catch (err) {
        alert('Could not create node. Please try again.')
    }
}

const close = () => {
    emit('close')
    setTimeout(() => {
        step.value = 1
        form.value.alias = ''
        nodeKey.value = ''
    }, 300)
}
</script>

<template>
    <BaseModal :show="show" @close="close" title="Deploy New Node">
        <form v-if="step === 1" @submit.prevent="submit" class="space-y-5">
            <p class="text-sm text-text-m leading-normal">
                Create a logical node in the hub before installing the agent on your server.
            </p>
            
            <div>
                <label class="block text-sm font-medium text-text-h mb-1.5">Node Alias</label>
                <BaseInput v-model="form.alias" placeholder="e.g., AWS-East-Gateway" autofocus />
            </div>

            <div class="flex justify-end gap-3 pt-5 border-t border-border-default mt-6">
                <BaseButton variant="ghost" @click="close">Cancel</BaseButton>
                <BaseButton variant="primary" type="submit" :disabled="!form.alias">Create Node</BaseButton>
            </div>
        </form>

        <div v-else class="space-y-6">
            <div class="flex items-center gap-3 text-success-text bg-success-bg border border-success-border p-3.5 rounded-md">
                <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                <span class="text-sm font-medium">Node created successfully.</span>
            </div>

            <div>
                <div class="flex items-center justify-between mb-1.5">
                    <label class="block text-sm font-medium text-text-h">Wizard Installation Command</label>
                    <BaseButton variant="ghost" class="!py-1 !px-2 !text-xs transition-colors" :class="copiedStates['modal-cmd'] ? '!text-success-main hover:!text-success-main' : ''" @click="handleCopy('modal-cmd', `curl -sL https://hub.honeywire.local/wizard.sh | bash -s -- --key ${nodeKey}`)">
                        <span class="flex items-center gap-1.5">
                            <svg v-if="!copiedStates['modal-cmd']" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                            <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                            {{ copiedStates['modal-cmd'] ? 'Copied!' : 'Copy' }}
                        </span>
                    </BaseButton>
                </div>
                <code class="block w-full p-4 bg-bg-inset border border-border-default rounded-md text-sm font-mono text-text-h whitespace-pre-wrap break-all leading-normal select-all">curl -sL https://hub.honeywire.local/wizard.sh | bash -s -- --key {{ nodeKey }}</code>
            </div>

            <div>
                <div class="flex items-center justify-between mb-1.5">
                    <label class="block text-sm font-medium text-text-m">Node API Key</label>
                    <BaseButton variant="ghost" class="!py-1 !px-2 !text-xs transition-colors" :class="copiedStates['modal-key'] ? '!text-success-main hover:!text-success-main' : ''" @click="handleCopy('modal-key', nodeKey)">
                        <span class="flex items-center gap-1.5">
                            <svg v-if="!copiedStates['modal-key']" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                            <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                            {{ copiedStates['modal-key'] ? 'Copied!' : 'Copy' }}
                        </span>
                    </BaseButton>
                </div>
                <div class="flex items-center gap-2">
                    <code class="flex-1 block px-3 py-2.5 bg-bg-inset border border-border-default rounded-md text-sm font-mono text-text-m truncate select-all">{{ nodeKey }}</code>
                </div>
                <p class="text-sm text-success-main font-medium mt-2">You can view this key again in the node details.</p>
            </div>

            <div class="flex justify-end pt-5 border-t border-border-default mt-6">
                <BaseButton variant="secondary" @click="close">Done</BaseButton>
            </div>
        </div>
    </BaseModal>
</template>