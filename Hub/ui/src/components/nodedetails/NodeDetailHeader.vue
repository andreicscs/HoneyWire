<script setup lang="ts">
import { ref, nextTick } from 'vue'
import type { FleetNode } from '../../stores/Fleet/fleet'
import BaseStatusDot from '../ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../ui/navigation/BaseMeatballMenu.vue'
import BaseButton from '../ui/forms/BaseButton.vue'
import { useClipboard } from '../../utils/useClipboard'

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

// --- INLINE EDITING STATE ---
const editingAlias = ref(false)
const aliasValue = ref('')
const aliasInput = ref<HTMLInputElement | null>(null)
const enableAliasEdit = async () => {
    if (!props.node) return
    editingAlias.value = true; aliasValue.value = props.node.alias; await nextTick(); aliasInput.value?.focus(); aliasInput.value?.select()
}
const saveAlias = () => {
    if (!editingAlias.value || !props.node) return
    const val = aliasValue.value.trim()
    if (val && val !== props.node.alias) emit('update', { alias: val })
    editingAlias.value = false
}

const editingPubIp = ref(false)
const pubIpValue = ref('')
const pubIpInput = ref<HTMLInputElement | null>(null)
const enablePubIpEdit = async () => {
    if (!props.node) return
    editingPubIp.value = true; pubIpValue.value = props.node.publicIp || ''; await nextTick(); pubIpInput.value?.focus(); pubIpInput.value?.select()
}
const savePubIp = () => {
    if (!editingPubIp.value || !props.node) return
    const val = pubIpValue.value.trim()
    if (val !== (props.node.publicIp || '')) emit('update', { publicIp: val })
    editingPubIp.value = false
}

const editingPrivIp = ref(false)
const privIpValue = ref('')
const privIpInput = ref<HTMLInputElement | null>(null)
const enablePrivIpEdit = async () => {
    if (!props.node) return
    editingPrivIp.value = true; privIpValue.value = props.node.privateIp || ''; await nextTick(); privIpInput.value?.focus(); privIpInput.value?.select()
}
const savePrivIp = () => {
    if (!editingPrivIp.value || !props.node) return
    const val = privIpValue.value.trim()
    if (val !== (props.node.privateIp || '')) emit('update', { privateIp: val })
    editingPrivIp.value = false
}

const editingTag = ref(false)
const newTagValue = ref('')
const tagInput = ref<HTMLInputElement | null>(null)
const enableTagEdit = async () => {
    editingTag.value = true; await nextTick(); tagInput.value?.focus()
}
const saveTag = () => {
    if (!editingTag.value || !props.node) return
    const val = newTagValue.value.trim()
    if (val && !props.node.tags.includes(val)) emit('update', { tags: [...props.node.tags, val] })
    editingTag.value = false; newTagValue.value = ''
}
const removeTag = (index: number) => {
    if (!props.node) return
    const newTags = [...props.node.tags]
    newTags.splice(index, 1)
    emit('update', { tags: newTags })
}
</script>

<template>
    <div v-if="node" class="flex flex-col gap-6">
        <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
            <div>
                <div class="flex items-center gap-3 mb-3">
                    <h1 v-if="!editingAlias" @click="enableAliasEdit" class="text-[length:var(--text-h1)] font-semibold text-text-h leading-tight truncate max-w-[400px] cursor-edit hover:text-primary-main border-b border-dashed border-transparent hover:border-primary-main transition-colors select-none" :title="`Click to rename ${node.alias}`">
                        {{ node.alias }}
                    </h1>
                    <input v-else ref="aliasInput" v-model="aliasValue" @keyup.enter="saveAlias" @keyup.esc="editingAlias = false" @blur="saveAlias" class="text-[length:var(--text-h1)] font-semibold text-text-h bg-input-bg border border-primary-main rounded px-2 py-0.5 focus:outline-none ring-1 ring-focus-ring max-w-[400px] truncate" />
                    
                    <BaseStatusDot :status="node.status || 'unknown'" />
                    
                    <span v-if="node.hasPendingConfig" class="shrink-0 text-high" title="Pending sync — click Sync Node below to apply changes">
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                    </span>
                    <span v-if="node.hasUpdateAvailable" class="w-2.5 h-2.5 rounded-full bg-low/70 shadow-[0_0_8px_rgba(var(--color-low),0.5)] shrink-0" title="Updates available for installed sensors"></span>
                </div>
                
                <div class="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-text-m">
                    <div class="flex items-center gap-1.5 transition-colors duration-[var(--duration-fast)] group/pub w-max rounded px-1 -ml-1 py-0.5 border border-transparent text-text-m hover:text-text-h hover:bg-secondary-hover">
                        <svg @click="node.publicIp ? handleCopy('detail-pub', node.publicIp) : null" class="w-4 h-4 shrink-0" :class="node.publicIp ? 'cursor-pointer hover:text-primary-main' : 'opacity-50'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                        <span v-if="!editingPubIp" @click="enablePubIpEdit" class="font-mono cursor-edit border-b border-dashed border-transparent hover:border-primary-main" :class="copiedStates['detail-pub'] ? 'text-success-main' : ''" :title="copiedStates['detail-pub'] ? 'Copied!' : 'Click to edit Public IP'">{{ copiedStates['detail-pub'] ? 'Copied!' : (node.publicIp || 'Unknown') }}</span>
                        <input v-else ref="pubIpInput" v-model="pubIpValue" @keyup.enter="savePubIp" @keyup.esc="editingPubIp = false" @blur="savePubIp" class="font-mono text-sm text-text-h bg-input-bg border border-primary-main rounded px-1 py-0 focus:outline-none ring-1 ring-focus-ring w-28" placeholder="0.0.0.0" />
                    </div>
                    <div class="flex items-center gap-1.5 transition-colors duration-[var(--duration-fast)] group/priv w-max rounded px-1 -ml-1 py-0.5 border border-transparent text-text-m hover:text-text-h hover:bg-secondary-hover">
                        <svg @click="node.privateIp ? handleCopy('detail-priv', node.privateIp) : null" class="w-4 h-4 shrink-0" :class="node.privateIp ? 'cursor-pointer hover:text-primary-main' : 'opacity-50'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="2" y="14" width="8" height="6" rx="2" ry="2"/><rect x="14" y="14" width="8" height="6" rx="2" ry="2"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 14v-2a2 2 0 012-2h8a2 2 0 012 2v2M12 2v8"/><rect x="8" y="2" width="8" height="6" rx="2" ry="2"/></svg>
                        <span v-if="!editingPrivIp" @click="enablePrivIpEdit" class="font-mono cursor-edit border-b border-dashed border-transparent hover:border-primary-main" :class="copiedStates['detail-priv'] ? 'text-success-main' : ''" :title="copiedStates['detail-priv'] ? 'Copied!' : 'Click to edit Private IP'">{{ copiedStates['detail-priv'] ? 'Copied!' : (node.privateIp || 'Unknown') }}</span>
                        <input v-else ref="privIpInput" v-model="privIpValue" @keyup.enter="savePrivIp" @keyup.esc="editingPrivIp = false" @blur="savePrivIp" class="font-mono text-sm text-text-h bg-input-bg border border-primary-main rounded px-1 py-0 focus:outline-none ring-1 ring-focus-ring w-28" placeholder="0.0.0.0" />
                    </div>
                    <div class="h-4 w-px bg-border-default hidden sm:block"></div>
                    <div class="flex items-center gap-1.5">
                        <span class="text-text-h font-medium">Last Event:</span> {{ lastEventTime }}
                    </div>
                    <div class="h-4 w-px bg-border-default hidden sm:block"></div>
                    <div class="flex items-center gap-1.5 flex-wrap">
                        <span v-for="(tag, index) in (node.tags || [])" :key="tag" class="px-2 py-0.5 bg-bg-inset border border-border-default text-text-m text-sm font-medium rounded-md tracking-wider flex items-center gap-1.5 group/tag transition-colors hover:border-text-m">
                            {{ tag }}
                            <button @click.stop="removeTag(index)" class="opacity-0 group-hover/tag:opacity-100 text-text-m hover:text-danger-main transition-all outline-none focus:opacity-100">
                                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                            </button>
                        </span>
                        <div v-if="editingTag" class="relative flex items-center">
                            <span class="absolute left-2 text-text-m text-sm pointer-events-none">+</span>
                            <input ref="tagInput" v-model="newTagValue" @keyup.enter="saveTag" @keyup.esc="editingTag = false" @blur="saveTag" class="pl-5 pr-2 py-0.5 bg-input-bg border border-primary-main text-text-h text-sm rounded-md focus:outline-none ring-1 ring-focus-ring w-28 shadow-sm transition-all placeholder:text-text-m/50" placeholder="tag name..." />
                        </div>
                        <button v-else @click.stop="enableTagEdit" class="px-1.5 py-0.5 border border-dashed border-border-default text-text-m text-sm rounded-md hover:text-text-h hover:border-text-m transition-colors outline-none focus:ring-1 focus:ring-focus-ring">
                            + Tag
                        </button>
                    </div>
                </div>
            </div>

            <div class="flex items-center gap-3 shrink-0">
                <BaseButton variant="secondary" class="!py-1.5 !px-3 !text-sm flex items-center gap-2" @click="$emit('manageKey')">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/></svg>
                    Manage Key
                </BaseButton>
                <BaseMeatballMenu id="node-super-menu">            
                    <button @click="$emit('silence')" class="w-full text-left px-3 py-2 text-sm font-medium flex items-center gap-2 text-text-m hover:bg-secondary-hover transition-colors group" :class="node.isSilenced ? 'text-archive-text hover:bg-archive-bg' : ' hover:text-text-h'">
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path v-if="!node.isSilenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                            <path v-if="node.isSilenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                        </svg>
                        {{ node.isSilenced ? 'Unsilence Node' : 'Silence Node' }}
                    </button>
                    
                    <button v-if="node.hasUpdateAvailable" @click="$emit('upgradeAll')" class="w-full text-left px-3 py-2 text-sm font-medium text-low flex items-center gap-2 hover:bg-low/10 transition-colors group">
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-180" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                        Update Node
                    </button>
                    
                    <button @click="$emit('delete')" class="w-full text-left px-3 py-2 text-sm font-medium text-danger-text flex items-center gap-2 hover:bg-danger-bg transition-colors group border-t border-border-default mt-1 pt-2">
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" />
                            <path class="origin-bottom-right transition-transform duration-normal group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                        </svg>
                        Delete Node
                    </button>
                </BaseMeatballMenu>
            </div>
        </div>

        <div v-if="node.hasPendingConfig" class="flex items-center justify-between w-full max-w-xl bg-high/10 border border-high/30 rounded-lg p-4 transition-all duration-normal">
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