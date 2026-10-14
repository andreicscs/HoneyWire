<script setup lang="ts">
import { computed } from 'vue'
import { useClipboard } from '../../utils/useClipboard'

const props = withDefaults(defineProps<{
    copyKey: string,
    text: string,
    variant?: 'surface' | 'secondary',
    compact?: boolean
}>(), {
    variant: 'secondary',
    compact: false
})

const { copiedStates, handleCopy } = useClipboard() as any

const isCopied = computed(() => !!copiedStates.value[props.copyKey])

// Visual variants shared by every copy button in the node modals.
const baseButton =
    'rounded-md text-sm font-medium font-sans transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 border focus:outline-none'
const regularPadding = 'px-3 py-1.5'
const compactPadding = 'px-2.5 py-1'
const successColors = 'bg-success-bg text-success-text border-success-border'
const secondaryColors = 'bg-secondary-main text-text-h border-secondary-border hover:bg-secondary-hover'
const surfaceColors = 'bg-bg-surface text-text-h border-border-default hover:bg-secondary-hover'
</script>

<template>
    <button
        @click="handleCopy(copyKey, text)"
        data-testid="copy-button"
        :data-copy-key="copyKey"
        :class="[baseButton, compact ? compactPadding : regularPadding, isCopied ? successColors : (variant === 'surface' ? surfaceColors : secondaryColors)]"
    >{{ isCopied ? 'Copied!' : 'Copy' }}</button>
</template>
