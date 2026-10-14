<script setup lang="ts">
import BaseModal from '../ui/feedback/BaseModal.vue'
import BaseButton from '../ui/forms/BaseButton.vue'
import CopyButton from './CopyButton.vue'
import { nodeKeyModalStyles } from './nodedetailsStyles.ts'

defineProps<{ show: boolean, apiKey: string | null }>()
defineEmits<{ (e: 'close'): void }>()
</script>

<template>
    <BaseModal :show="show" @close="$emit('close')" title="Manage Node Key">
        <div :class="nodeKeyModalStyles.stack">
            <p class="text-sm text-text-m">This is the unique API key for this node. It is used to authenticate the node with the hub.</p>
            <div :class="nodeKeyModalStyles.keyBox">
                <div :class="nodeKeyModalStyles.keyHead">
                    <span class="text-sm text-text-h font-semibold">Node API Key</span>
                    <CopyButton copy-key="key-modal" :text="apiKey ?? ''" variant="secondary" compact />
                </div>
                <div :class="nodeKeyModalStyles.keyValue">{{ apiKey || 'Unavailable' }}</div>
            </div>
            <div :class="nodeKeyModalStyles.footer">
                <BaseButton variant="primary" @click="$emit('close')">Close</BaseButton>
            </div>
        </div>
    </BaseModal>
</template>