<script setup lang="ts">
import { ref } from 'vue'
import type { FleetNode } from '../../stores/Fleet/fleet'
import BaseStatusDot from '../ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../ui/navigation/BaseMeatballMenu.vue'
import BaseButton from '../ui/forms/BaseButton.vue'
import { useClipboard } from '../../utils/useClipboard'
import { useNodeFieldEdit } from '../../utils/useNodeFieldEdit'
import { nodeHeaderStyles, nodeMenuStyles } from './nodedetailsStyles.ts'

const props = defineProps<{
    node: FleetNode | null,
    lastEventTime: string
}>()

const emit = defineEmits<{
    (e: 'update', updates: any): void,
    (e: 'silence'): void,
    (e: 'delete'): void,
    (e: 'sync'): void,
    (e: 'manageKey'): void,
    (e: 'upgradeAll'): void
}>()

const { copiedStates, handleCopy } = useClipboard() as any

// --- INLINE EDITING STATE (shared with FleetNodeWidget) ---
const aliasInput = ref<HTMLInputElement | null>(null)
const pubIpInput = ref<HTMLInputElement | null>(null)
const privIpInput = ref<HTMLInputElement | null>(null)
const tagInput = ref<HTMLInputElement | null>(null)
const {
    isEditingAlias, aliasValue, enableAliasEdit, cancelAliasEdit, saveAlias,
    isEditingPubIp, pubIpValue, enablePubIpEdit, cancelPubIpEdit, savePubIp,
    isEditingPrivIp, privIpValue, enablePrivIpEdit, cancelPrivIpEdit, savePrivIp,
    isEditingTag, newTagValue, enableTagEdit, cancelTag, saveTag, removeTag,
} = useNodeFieldEdit(() => props.node, emit, { aliasInput, pubIpInput, privIpInput, tagInput })
</script>

<template>
    <div v-if="node" :class="nodeHeaderStyles.root">
        <div :class="nodeHeaderStyles.titleRow">
            <div>
                <div class="flex items-center gap-3 mb-3">
                    <h1 v-if="!isEditingAlias" @click="enableAliasEdit" :class="nodeHeaderStyles.aliasTitle" :title="`Click to rename ${node.alias}`">
                        {{ node.alias }}
                    </h1>
                    <input v-else ref="aliasInput" v-model="aliasValue" @keyup.enter="saveAlias" @keyup.esc="cancelAliasEdit" @blur="saveAlias" :class="nodeHeaderStyles.aliasInput" />
                    
                    <BaseStatusDot :status="node.status || 'unknown'" />
                    
                    <span v-if="node.hasPendingConfig" class="shrink-0 text-high" title="Pending sync, click Sync Node below to apply changes">
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                    </span>
                    <span v-if="node.hasUpdateAvailable" class="relative flex h-2.5 w-2.5 shrink-0" title="Updates available for installed sensors">
                        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-update-main opacity-75"></span>
                        <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-update-main"></span>
                    </span>
                </div>
                
                <div :class="nodeHeaderStyles.metaRow">
                    <div :class="[nodeHeaderStyles.ipPillBase, 'group/pub']">
                        <svg @click="node.publicIp ? handleCopy('detail-pub', node.publicIp) : null" class="w-4 h-4 shrink-0" :class="node.publicIp ? 'cursor-pointer hover:text-primary-main' : 'opacity-50'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                        <span v-if="!isEditingPubIp" @click="enablePubIpEdit" class="font-mono cursor-edit border-b border-dashed border-transparent hover:border-primary-main" :class="copiedStates['detail-pub'] ? 'text-success-main' : ''" :title="copiedStates['detail-pub'] ? 'Copied!' : 'Click to edit Public IP'">{{ copiedStates['detail-pub'] ? 'Copied!' : (node.publicIp || 'Unknown') }}</span>
                        <input v-else ref="pubIpInput" v-model="pubIpValue" @keyup.enter="savePubIp" @keyup.esc="cancelPubIpEdit" @blur="savePubIp" :class="nodeHeaderStyles.ipInput" placeholder="0.0.0.0" />
                    </div>
                    <div :class="[nodeHeaderStyles.ipPillBase, 'group/priv']">
                        <svg @click="node.privateIp ? handleCopy('detail-priv', node.privateIp) : null" class="w-4 h-4 shrink-0" :class="node.privateIp ? 'cursor-pointer hover:text-primary-main' : 'opacity-50'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="2" y="14" width="8" height="6" rx="2" ry="2"/><rect x="14" y="14" width="8" height="6" rx="2" ry="2"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 14v-2a2 2 0 012-2h8a2 2 0 012 2v2M12 2v8"/><rect x="8" y="2" width="8" height="6" rx="2" ry="2"/></svg>
                        <span v-if="!isEditingPrivIp" @click="enablePrivIpEdit" class="font-mono cursor-edit border-b border-dashed border-transparent hover:border-primary-main" :class="copiedStates['detail-priv'] ? 'text-success-main' : ''" :title="copiedStates['detail-priv'] ? 'Copied!' : 'Click to edit Private IP'">{{ copiedStates['detail-priv'] ? 'Copied!' : (node.privateIp || 'Unknown') }}</span>
                        <input v-else ref="privIpInput" v-model="privIpValue" @keyup.enter="savePrivIp" @keyup.esc="cancelPrivIpEdit" @blur="savePrivIp" :class="nodeHeaderStyles.ipInput" placeholder="0.0.0.0" />
                    </div>
                    <div :class="nodeHeaderStyles.divider"></div>
                    <div class="flex items-center gap-1.5">
                        <span class="text-text-h font-medium">Last Event:</span> {{ lastEventTime }}
                    </div>
                    <div :class="nodeHeaderStyles.divider"></div>
                    <div class="flex items-center gap-1.5 flex-wrap">
                        <span v-for="(tag, index) in (node.tags || [])" :key="tag" :class="nodeHeaderStyles.tagPill">
                            {{ tag }}
                            <button @click.stop="removeTag(index)" class="opacity-0 group-hover/tag:opacity-100 text-text-m hover:text-danger-main transition-all outline-none focus:opacity-100">
                                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                            </button>
                        </span>
                        <div v-if="isEditingTag" class="relative flex items-center">
                            <span class="absolute left-2 text-text-m text-sm pointer-events-none">+</span>
                            <input ref="tagInput" v-model="newTagValue" @keyup.enter="saveTag" @keyup.esc="cancelTag" @blur="saveTag" :class="nodeHeaderStyles.tagInput" placeholder="tag name..." />
                        </div>
                        <button v-else @click.stop="enableTagEdit" :class="nodeHeaderStyles.addTagButton">
                            + Tag
                        </button>
                    </div>
                </div>
            </div>

            <div :class="nodeHeaderStyles.actionsRow">
                <BaseButton variant="secondary" class="!py-1.5 !px-3 !text-base flex items-center gap-2" @click="$emit('manageKey')">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/></svg>
                    Manage Key
                </BaseButton>
                <BaseMeatballMenu id="node-super-menu">            
                    <button @click="$emit('silence')" data-testid="node-menu-silence" :class="[nodeMenuStyles.item, node.isSilenced ? 'text-archive-text hover:bg-archive-bg' : ' hover:text-text-h']">
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path v-if="!node.isSilenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                            <path v-if="node.isSilenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                        </svg>
                        {{ node.isSilenced ? 'Unsilence Node' : 'Silence Node' }}
                    </button>
                    
                    <button v-if="node.hasUpdateAvailable" @click="$emit('upgradeAll')" data-testid="node-menu-upgrade" :class="nodeMenuStyles.itemUpdate">
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-180" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                        Update Node
                    </button>
                    
                    <button @click="$emit('delete')" data-testid="node-menu-delete" :class="nodeMenuStyles.itemDanger">
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" />
                            <path class="origin-bottom-right transition-transform duration-normal group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                        </svg>
                        Delete Node
                    </button>
                </BaseMeatballMenu>
            </div>
        </div>

        <div v-if="node.hasPendingConfig" :class="nodeHeaderStyles.pendingBanner">
            <div class="flex items-start gap-3">
                <svg class="w-5 h-5 text-high mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                <div>
                    <h4 class="text-sm font-semibold text-high">Pending Sync</h4>
                    <p class="text-sm text-text-h opacity-90 mt-0.5">Sensors have been added or modified. Sync this node to apply changes.</p>
                </div>
            </div>
            <BaseButton @click="$emit('sync')" variant="secondary" class="!border-high/30 !bg-bg-surface hover:!bg-high/10 !text-high shrink-0">Sync Node</BaseButton>
        </div>
    </div>
</template>