<script setup>
import ThemeToggle from '../ui/branding/ThemeToggle.vue'
import BaseLogo from '../ui/branding/BaseLogo.vue'
import PageHeader from '../ui/layout/PageHeader.vue'
import BaseCard from '../ui/layout/BaseCard.vue'
import { authStyles, SEVERITY_GRADIENT } from '../../views/authStyles.ts'

defineProps({
    width: {
        type: String,
        default: 'sm',
        validator: (v) => ['sm', 'xl'].includes(v)
    },
    scrollable: { type: Boolean, default: false }
})
defineEmits(['toggle-theme'])
</script>

<template>
    <div :class="scrollable ? authStyles.pageScroll : authStyles.pageFixed" data-testid="auth-layout">

        <div :class="authStyles.themeToggle">
            <ThemeToggle @toggle="$emit('toggle-theme')" />
        </div>

        <div :class="width === 'xl' ? authStyles.containerXl : authStyles.containerSm" data-testid="auth-container" :data-width="width">

            <div :class="authStyles.headerBlock">
                <BaseLogo />
                <PageHeader
                    center
                    title="HoneyWire Sentinel"
                    description="Authorized personnel only."
                />
            </div>

            <BaseCard :class="authStyles.card">
                <div :class="authStyles.severityBar" :style="{ background: SEVERITY_GRADIENT }"></div>
                <slot />
            </BaseCard>
        </div>
    </div>
</template>
