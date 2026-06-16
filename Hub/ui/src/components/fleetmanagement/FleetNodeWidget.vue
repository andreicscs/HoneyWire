<script setup lang="ts">
import { ref, nextTick } from 'vue'
import BaseWidget from '../ui/layout/BaseWidget.vue'
import BaseStatusDot from '../ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../ui/navigation/BaseMeatballMenu.vue'
import { useClipboard } from '../../utils/useClipboard.js'

const props = defineProps<{
    node: any,
    isManifestLoading: boolean,
    isDeleting: boolean
}>()

const emit = defineEmits<{
    (e: 'update', updates: any): void,
    (e: 'silence'): void,
    (e: 'delete'): void,
    (e: 'openDetail'): void,
    (e: 'upgradeAll'): void
}>()

const { copiedStates, handleCopy } = useClipboard() as any

// --- Alias Edit ---
const isEditingAlias = ref(false)
const aliasValue = ref('')
const aliasInputRef = ref<HTMLInputElement | null>(null)
const enableAliasEdit = async () => {
    isEditingAlias.value = true; aliasValue.value = props.node.alias; await nextTick(); aliasInputRef.value?.focus(); aliasInputRef.value?.select()
}
const cancelAliasEdit = () => { isEditingAlias.value = false; aliasValue.value = '' }
const saveAlias = () => {
    if (!isEditingAlias.value) return
    const val = aliasValue.value.trim()
    if (val && val !== props.node.alias) emit('update', { alias: val })
    isEditingAlias.value = false
}

// --- Public IP Edit ---
const isEditingPubIp = ref(false)
const pubIpValue = ref('')
const pubIpInputRef = ref<HTMLInputElement | null>(null)
const enablePubIpEdit = async () => {
    isEditingPubIp.value = true; pubIpValue.value = props.node.publicIp || ''; await nextTick(); pubIpInputRef.value?.focus(); pubIpInputRef.value?.select()
}
const cancelPubIpEdit = () => { isEditingPubIp.value = false; pubIpValue.value = '' }
const savePubIp = () => {
    if (!isEditingPubIp.value) return
    const val = pubIpValue.value.trim()
    if (val !== (props.node.publicIp || '')) emit('update', { publicIp: val })
    isEditingPubIp.value = false
}

// --- Private IP Edit ---
const isEditingPrivIp = ref(false)
const privIpValue = ref('')
const privIpInputRef = ref<HTMLInputElement | null>(null)
const enablePrivIpEdit = async () => {
    isEditingPrivIp.value = true; privIpValue.value = props.node.privateIp || ''; await nextTick(); privIpInputRef.value?.focus(); privIpInputRef.value?.select()
}
const cancelPrivIpEdit = () => { isEditingPrivIp.value = false; privIpValue.value = '' }
const savePrivIp = () => {
    if (!isEditingPrivIp.value) return
    const val = privIpValue.value.trim()
    if (val !== (props.node.privateIp || '')) emit('update', { privateIp: val })
    isEditingPrivIp.value = false
}

// --- Tag Edit ---
const isEditingTag = ref(false)
const newTagValue = ref('')
const tagInputRef = ref<HTMLInputElement | null>(null)
const enableTagEdit = async () => {
    isEditingTag.value = true; await nextTick(); tagInputRef.value?.focus()
}
const cancelTag = () => { isEditingTag.value = false; newTagValue.value = '' }
const saveTag = () => {
    if (!isEditingTag.value) return
    const val = newTagValue.value.trim()
    if (val && !props.node.tags.includes(val)) emit('update', { tags: [...props.node.tags, val] })
    isEditingTag.value = false; newTagValue.value = ''
}
const removeTag = (index: number | string) => {
    const newTags = [...props.node.tags]
    newTags.splice(Number(index), 1)
    emit('update', { tags: newTags })
}
</script>

<template>
    <BaseWidget class="flex flex-col h-full min-h-[280px] transition-all duration-[var(--duration-fast)] !overflow-visible hover:z-elevated focus-within:z-elevated" :class="{ 'opacity-50 pointer-events-none': isDeleting }">
        <template #header>
            <div class="flex items-start justify-between w-full">
                <div class="flex items-center gap-2.5 min-w-0">
                    <BaseStatusDot :status="node.status" />
                    <div>
                        <div class="flex items-center gap-2">
                            <span v-if="!isEditingAlias" @click="enableAliasEdit" class="text-base font-medium text-text-h truncate max-w-[180px] cursor-edit hover:text-primary-main border-b border-dashed border-transparent hover:border-primary-main transition-colors select-none" :title="`Click to rename ${node.alias}`">{{ node.alias }}</span>
                            <input v-else ref="aliasInputRef" v-model="aliasValue" @keyup.enter="saveAlias" @keyup.esc="cancelAliasEdit" @blur="saveAlias" class="text-base font-medium text-text-h bg-input-bg border border-primary-main rounded px-1.5 py-0 focus:outline-none ring-1 ring-focus-ring max-w-[180px] truncate" />
                            
                            <span v-if="node.hasPendingConfig" class="shrink-0 text-high" title="Pending sync — open node details to apply changes">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                            </span>
                            <span v-if="node.hasUpdateAvailable" class="w-2 h-2 rounded-full bg-low/70 shrink-0" title="Sensor Updates Available"></span>
                            <svg v-if="node.isSilenced" class="w-4 h-4 text-medium shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" title="Node Silenced"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/></svg>
                        </div>
                    </div>
                </div>

                <BaseMeatballMenu v-if="node.id !== 'unassigned'" :id="`node-menu-${node.id}`">
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
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" /><path class="origin-bottom-right transition-transform duration-normal group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" /></svg>
                        <span v-if="isDeleting">Deleting...</span><span v-else>Delete</span>
                    </button>
                </BaseMeatballMenu>
            </div>
        </template>

        <div class="flex-1 mt-3 flex flex-col">
            <div v-if="node.id !== 'unassigned'" class="grid grid-cols-2 gap-y-2 gap-x-4 text-sm mb-4">
                <div class="flex items-center gap-1.5 transition-colors duration-[var(--duration-fast)] group/pub w-max rounded px-1 -ml-1 py-0.5 border border-transparent text-text-m hover:text-text-h hover:bg-secondary-hover">
                    <svg @click="node.publicIp ? handleCopy(node.id + '-pub', node.publicIp) : null" class="w-3.5 h-3.5 shrink-0" :class="node.publicIp ? 'cursor-pointer hover:text-primary-main' : 'opacity-50'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                    <span v-if="!isEditingPubIp" @click="enablePubIpEdit" class="font-mono truncate cursor-edit border-b border-dashed border-transparent hover:border-primary-main" :class="copiedStates[node.id + '-pub'] ? 'text-success-main' : ''" :title="copiedStates[node.id + '-pub'] ? 'Copied!' : 'Click to edit Public IP'">{{ copiedStates[node.id + '-pub'] ? 'Copied!' : (node.publicIp || 'Unknown') }}</span>
                    <input v-else ref="pubIpInputRef" v-model="pubIpValue" @keyup.enter="savePubIp" @keyup.esc="cancelPubIpEdit" @blur="savePubIp" class="font-mono text-sm text-text-h bg-input-bg border border-primary-main rounded px-1 py-0 focus:outline-none ring-1 ring-focus-ring w-28 truncate" placeholder="0.0.0.0" />
                </div>
                <div class="flex items-center gap-1.5 transition-colors duration-[var(--duration-fast)] group/priv w-max rounded px-1 -ml-1 py-0.5 border border-transparent text-text-m hover:text-text-h hover:bg-secondary-hover">
                    <svg @click="node.privateIp ? handleCopy(node.id + '-priv', node.privateIp) : null" class="w-3.5 h-3.5 shrink-0" :class="node.privateIp ? 'cursor-pointer hover:text-primary-main' : 'opacity-50'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="2" y="14" width="8" height="6" rx="2" ry="2"/><rect x="14" y="14" width="8" height="6" rx="2" ry="2"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 14v-2a2 2 0 012-2h8a2 2 0 012 2v2M12 2v8"/><rect x="8" y="2" width="8" height="6" rx="2" ry="2"/></svg>
                    <span v-if="!isEditingPrivIp" @click="enablePrivIpEdit" class="font-mono truncate cursor-edit border-b border-dashed border-transparent hover:border-primary-main" :class="copiedStates[node.id + '-priv'] ? 'text-success-main' : ''" :title="copiedStates[node.id + '-priv'] ? 'Copied!' : 'Click to edit Private IP'">{{ copiedStates[node.id + '-priv'] ? 'Copied!' : (node.privateIp || 'Unknown') }}</span>
                    <input v-else ref="privIpInputRef" v-model="privIpValue" @keyup.enter="savePrivIp" @keyup.esc="cancelPrivIpEdit" @blur="savePrivIp" class="font-mono text-sm text-text-h bg-input-bg border border-primary-main rounded px-1 py-0 focus:outline-none ring-1 ring-focus-ring w-28 truncate" placeholder="0.0.0.0" />
                </div>
            </div>

            <div class="flex flex-wrap gap-1.5 mb-4">
                <span v-for="(tag, index) in node.tags" :key="tag" class="px-2 py-0.5 bg-bg-inset border border-border-default text-text-m text-sm font-medium rounded-md tracking-wider flex items-center gap-1.5 group/tag transition-colors hover:border-text-m">
                    {{ tag }}
                    <button @click.stop="removeTag(index)" class="opacity-0 group-hover/tag:opacity-100 text-text-m hover:text-danger-main transition-all outline-none focus:opacity-100">
                        <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                    </button>
                </span>
                <div v-if="isEditingTag" class="relative flex items-center">
                    <span class="absolute left-2 text-text-m text-sm pointer-events-none">+</span>
                    <input ref="tagInputRef" v-model="newTagValue" @keyup.enter="saveTag" @keyup.esc="cancelTag" @blur="cancelTag" class="pl-5 pr-2 py-0.5 bg-input-bg border border-primary-main text-text-h text-sm rounded-md focus:outline-none ring-1 ring-focus-ring w-32 shadow-sm transition-all placeholder:text-text-m/50" placeholder="tag name..." />
                </div>
                <button v-else @click.stop="enableTagEdit" class="px-1.5 py-0.5 border border-dashed border-border-default text-text-m text-sm rounded-md hover:text-text-h hover:border-text-m transition-colors outline-none focus:ring-1 focus:ring-focus-ring">+ Tag</button>
            </div>

            <div v-if="!node.isAwaitingCheckIn" class="mt-auto bg-bg-surface border border-border-default rounded-lg p-3">
                <div class="flex items-center justify-between mb-2">
                    <span class="text-sm font-normal text-text-h">Deployed Sensors</span>
                    <span class="text-sm text-text-m">{{ node.onlineSensors }} / {{ node.totalSensors }} Online</span>
                </div>
                
                <div class="flex flex-wrap gap-1.5 mt-1">
                    <template v-if="isManifestLoading">
                        <div v-for="i in Math.min(node.totalSensors, 3)" :key="i" class="px-2 py-1 rounded-md text-sm bg-secondary-main border border-border-default animate-pulse w-16 h-6"></div>
                    </template>
                    <template v-else>
                        <div v-for="summary in node.sensorSummary" :key="summary.type" class="relative group/tooltip">
                            <span class="px-2 py-1 rounded-md text-sm font-medium flex items-center gap-1.5 border border-border-default bg-secondary-main text-text-m cursor-default hover:border-text-m transition-colors"><span class="text-text-h">{{ summary.count }}</span> {{ summary.type }}</span>
                            <div class="absolute bottom-full left-0 mb-2 w-max min-w-[180px] opacity-0 invisible group-hover/tooltip:opacity-100 group-hover/tooltip:visible transition-all duration-fast z-dropdown bg-bg-surface border border-border-default shadow-lg rounded-md p-2 pointer-events-none">
                                <div class="text-sm font-medium text-text-h mb-2">{{ summary.type }}</div>
                                <div class="flex flex-col gap-2">
                                    <div v-for="s in summary.sensors" :key="s.name" class="flex items-center gap-2 text-sm text-text-h"><BaseStatusDot :status="s.status || 'offline'" /><span class="font-mono truncate">{{ s.name.replace(/^hw-sensor-/, '') }}</span></div>
                                </div>
                                <div class="absolute top-full left-6 -mt-px border-4 border-transparent border-t-border-default"></div><div class="absolute top-full left-6 -mt-[2px] border-4 border-transparent border-t-bg-surface"></div>
                            </div>
                        </div>
                    </template>
                    <span v-if="node.sensorSummary.length === 0 && !isManifestLoading" class="text-sm text-text-m italic">No sensors deployed.</span>
                </div>
            </div>
            <div v-else class="mt-auto bg-disabled-bg border border-dashed border-disabled-border rounded-lg p-4 flex flex-col items-center justify-center text-center opacity-70">
                <svg class="w-6 h-6 text-text-h mb-2 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" /></svg>
                <span class="text-sm font-medium text-text-h">Awaiting Initial Check-in</span>
                <span class="text-sm text-text-m mt-1 max-w-[200px]">Deploy your first Sensor!</span>
            </div>
        </div>

        <template #footer>
            <button @click="$emit('openDetail')" class="w-full py-3 text-sm font-medium text-text-h hover:text-text-h bg-bg-secondary hover:bg-secondary-hover border-t border-border-default rounded-b-xl transition-colors flex items-center justify-center gap-2 group/btn outline-none">
                <span>{{ node.totalSensors === 0 ? 'Install First Sensor' : 'Manage Node & Sensors' }}</span>
                <svg class="w-4 h-4 transition-transform duration-normal group-hover/btn:translate-x-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
            </button>
        </template>
    </BaseWidget>
</template>
