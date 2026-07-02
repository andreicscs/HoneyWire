<script setup lang="ts">
import BaseModal from '../ui/feedback/BaseModal.vue'
import BaseButton from '../ui/forms/BaseButton.vue'
import { useClipboard } from '../../utils/useClipboard'

defineProps<{ show: boolean, apiKey: string | null }>()
defineEmits<{ (e: 'close'): void }>()

const { copiedStates, handleCopy } = useClipboard() as any
</script>

<template>
    <BaseModal :show="show" @close="$emit('close')" title="Manage Node Key">
        <div class="space-y-4">
            <p class="text-sm text-text-m">This is the unique API key for this node. It is used to authenticate the node with the hub.</p>
            <div class="bg-bg-surface border border-border-default rounded-lg p-4 text-sm break-all">
                <div class="flex items-center justify-between mb-2">
                    <span class="text-sm text-text-h font-semibold">Node API Key</span>
                    <button @click="handleCopy('key-modal', apiKey)" class="px-2.5 py-1 rounded-md text-sm font-medium font-sans transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 border outline-none" :class="copiedStates['key-modal'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-secondary-main text-text-h border-secondary-border hover:bg-secondary-hover'">{{ copiedStates['key-modal'] ? 'Copied!' : 'Copy' }}</button>
                </div>
                <div class="text-text-m select-all font-mono">{{ apiKey || 'Unavailable' }}</div>
            </div>
            <div class="flex justify-end">
                <BaseButton variant="primary" @click="$emit('close')">Close</BaseButton>
            </div>
        </div>
    </BaseModal>
</template>