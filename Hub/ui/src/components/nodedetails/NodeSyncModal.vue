<script setup lang="ts">
import { ref } from 'vue'
import BaseButton from '../ui/forms/BaseButton.vue'
import { useClipboard } from '../../utils/useClipboard'

defineProps<{ show: boolean, syncCommand: string, syncComposeYaml: string }>()
defineEmits<{ (e: 'close'): void }>()

const { copiedStates, handleCopy } = useClipboard() as any
const showManualSync = ref(false)
</script>

<template>
    <Teleport to="body">
        <transition enter-active-class="transition duration-normal ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-[var(--duration-fast)] ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
            <div v-if="show" class="fixed inset-0 z-[var(--z-modal)] flex justify-center items-center p-4 sm:p-6 bg-black/60 backdrop-blur-sm" @mousedown.self="$emit('close')">
                <div class="bg-bg-base w-full max-w-2xl max-h-[85vh] rounded-lg shadow-2xl flex flex-col overflow-hidden border border-border-default transform transition-all">
                    <div class="px-6 py-5 border-b border-border-default flex justify-between items-center bg-bg-surface shrink-0">
                        <h2 class="text-base font-semibold text-text-h">Synchronize Node</h2>
                        <button @click="$emit('close')" class="p-2 -mr-2 text-text-m hover:text-text-h transition-colors duration-[var(--duration-fast)] rounded-full hover:bg-secondary-hover focus:outline-none"><svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path></svg></button>
                    </div>

                    <div class="flex-1 overflow-y-auto custom-scroll bg-bg-base p-6 md:p-8 space-y-6">
                        <div>
                            <h3 class="text-sm font-semibold text-text-h mb-2">Automatic Deployment (Recommended)</h3>
                            <p class="text-sm text-text-m mb-4">Run the HoneyWire Wizard on your server to automatically reconcile this node's configuration.</p>
                            <div class="bg-bg-inset/50 border border-border-default rounded-md p-4 relative group flex flex-col gap-3">
                                <code class="text-success-text text-xs font-mono whitespace-pre-wrap break-all leading-relaxed">{{ syncCommand }}</code>
                                <button @click="handleCopy('sync-cmd', syncCommand)" class="self-end px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 focus:outline-none border" :class="copiedStates['sync-cmd'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-bg-surface text-text-h border-border-default hover:bg-secondary-hover'">{{ copiedStates['sync-cmd'] ? 'Copied!' : 'Copy' }}</button>
                            </div>
                        </div>

                        <div class="border-t border-border-default pt-6">
                            <button @click="showManualSync = !showManualSync" class="flex items-center justify-between w-full text-left group outline-none">
                                <span class="text-sm font-semibold text-text-h group-hover:text-primary-main transition-colors">Advanced / Manual Deployment</span>
                                <svg class="w-4 h-4 text-text-m transition-transform duration-normal" :class="showManualSync ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                            </button>
                            <div v-show="showManualSync" class="mt-4 space-y-4">
                                <p class="text-sm text-danger-main">Docker Compose v5.0.0+ is strictly required.</p>
                                <p class="text-sm text-text-m">Save the following configuration to <code class="px-1.5 py-0.5 bg-bg-inset border border-border-default rounded text-xs font-mono">/opt/honeywire/sensors/honeywire-compose.yml</code>.</p>
                                <div class="relative bg-bg-surface border border-border-default rounded-lg p-4 text-sm text-text-h overflow-auto max-h-[30vh] custom-scroll">
                                    <button @click="handleCopy('sync-yaml', syncComposeYaml)" class="absolute top-3 right-3 px-3 py-1.5 rounded-md text-sm font-medium font-sans transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 z-10 focus:outline-none border" :class="copiedStates['sync-yaml'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-secondary-main text-text-h border-secondary-border hover:bg-secondary-hover'">{{ copiedStates['sync-yaml'] ? 'Copied!' : 'Copy' }}</button>
                                    <pre class="whitespace-pre-wrap break-words pr-16 font-mono">{{ syncComposeYaml || 'No compose output available.' }}</pre>
                                </div>
                                <p class="text-sm text-text-m">Then, apply the configuration using Docker Compose:</p>
                                <div class="bg-bg-inset/50 border border-border-default rounded-md p-4 relative group flex flex-col gap-3">
                                    <code class="text-text-h text-xs font-mono break-all leading-relaxed">docker compose -f /opt/honeywire/sensors/honeywire-compose.yml -p honeywire up -d --remove-orphans</code>
                                    <button @click="handleCopy('manual-cmd', 'docker compose -f /opt/honeywire/sensors/honeywire-compose.yml -p honeywire up -d --remove-orphans')" class="self-end px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 focus:outline-none border" :class="copiedStates['manual-cmd'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-bg-surface text-text-h border-border-default hover:bg-secondary-hover'">{{ copiedStates['manual-cmd'] ? 'Copied!' : 'Copy' }}</button>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="px-6 py-4 border-t border-border-default bg-bg-surface flex justify-end shrink-0"><BaseButton variant="secondary" @click="$emit('close')">Done</BaseButton></div>
                </div>
            </div>
        </transition>
    </Teleport>
</template>