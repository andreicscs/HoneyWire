<script setup lang="ts">
import { ref } from 'vue'
import BaseButton from '../ui/forms/BaseButton.vue'
import CopyButton from './CopyButton.vue'
import { nodeModalStyles } from './nodedetailsStyles.ts'

defineProps<{ show: boolean, syncCommand: string, syncComposeYaml: string }>()
defineEmits<{ (e: 'close'): void }>()

const MANUAL_COMPOSE_CMD = 'docker compose -f /opt/honeywire/sensors/honeywire-compose.yml -p honeywire up -d --remove-orphans'

const showManualSync = ref(false)
</script>

<template>
    <Teleport to="body">
        <transition enter-active-class="transition duration-normal ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-[var(--duration-fast)] ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
            <div v-if="show" :class="nodeModalStyles.overlay" @mousedown.self="$emit('close')">
                <div :class="[nodeModalStyles.panelBase, 'max-w-2xl']">
                    <div :class="nodeModalStyles.headRow">
                        <h2 class="text-base font-semibold text-text-h">Synchronize Node</h2>
                        <button @click="$emit('close')" :class="nodeModalStyles.closeButton"><svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path></svg></button>
                    </div>

                    <div class="flex-1 overflow-y-auto custom-scroll bg-bg-base p-6 md:p-8 space-y-6">
                        <div>
                            <h3 class="text-sm font-semibold text-text-h mb-2">Automatic Deployment (Recommended)</h3>
                            <p class="text-sm text-text-m mb-4">Run the HoneyWire Wizard on your server to automatically reconcile this node's configuration.</p>
                            <div class="bg-bg-inset/50 border border-border-default rounded-md p-4 relative group flex flex-col gap-3">
                                <code class="text-success-text text-xs font-mono whitespace-pre-wrap break-all leading-relaxed">{{ syncCommand }}</code>
                                <CopyButton copy-key="sync-cmd" :text="syncCommand" variant="surface" class="self-end" />
                            </div>
                        </div>

                        <div class="border-t border-border-default pt-6">
                            <button @click="showManualSync = !showManualSync" class="flex items-center justify-between w-full text-left group outline-none">
                                <span class="text-sm font-semibold text-text-h group-hover:text-primary-main transition-colors">Advanced / Manual Deployment</span>
                                <svg class="w-4 h-4 text-text-m transition-transform duration-normal" :class="showManualSync ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                            </button>
                            <div v-show="showManualSync" data-testid="sync-manual-section" class="mt-4 space-y-4">
                                <p class="text-sm text-danger-main">Docker Compose v5.0.0+ and Rootful Docker are strictly required.</p>
                                <p class="text-sm text-text-m">Save the following configuration to <code class="px-1.5 py-0.5 bg-bg-inset border border-border-default rounded text-xs font-mono">/opt/honeywire/sensors/honeywire-compose.yml</code>.</p>
                                <div class="relative bg-bg-surface border border-border-default rounded-lg p-4 text-sm text-text-h overflow-auto max-h-[30vh] custom-scroll">
                                    <CopyButton copy-key="sync-yaml" :text="syncComposeYaml" variant="secondary" class="absolute top-3 right-3 z-10" />
                                    <pre class="whitespace-pre-wrap break-words pr-16 font-mono">{{ syncComposeYaml || 'No compose output available.' }}</pre>
                                </div>
                                <p class="text-sm text-text-m">Then, apply the configuration using Docker Compose:</p>
                                <div class="bg-bg-inset/50 border border-border-default rounded-md p-4 relative group flex flex-col gap-3">
                                    <code class="text-text-h text-xs font-mono break-all leading-relaxed">{{ MANUAL_COMPOSE_CMD }}</code>
                                    <CopyButton copy-key="manual-cmd" :text="MANUAL_COMPOSE_CMD" variant="surface" class="self-end" />
                                </div>
                            </div>
                        </div>
                    </div>
                    <div :class="nodeModalStyles.footerBar"><BaseButton variant="primary" @click="$emit('close')">Done</BaseButton></div>
                </div>
            </div>
        </transition>
    </Teleport>
</template>