<script setup lang="ts">
import { ref } from 'vue'
import BaseWidget from '../ui/layout/BaseWidget.vue'
import BaseStatusDot from '../ui/feedback/BaseStatusDot.vue'
import BaseMeatballMenu from '../ui/navigation/BaseMeatballMenu.vue'
import { useClipboard } from '../../utils/useClipboard.js'
import { useNodeFieldEdit } from '../../utils/useNodeFieldEdit.js'
import { nodeHeaderStyles, nodeMenuStyles } from '../nodedetails/nodedetailsStyles.ts'
import { fleetWidgetStyles } from './fleetmanagementStyles.ts'

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

// Input refs stay local (template `ref` attributes); shared edit logic below.
const aliasInputRef = ref<HTMLInputElement | null>(null)
const pubIpInputRef = ref<HTMLInputElement | null>(null)
const privIpInputRef = ref<HTMLInputElement | null>(null)
const tagInputRef = ref<HTMLInputElement | null>(null)

const {
    isEditingAlias, aliasValue, enableAliasEdit, cancelAliasEdit, saveAlias,
    isEditingPubIp, pubIpValue, enablePubIpEdit, cancelPubIpEdit, savePubIp,
    isEditingPrivIp, privIpValue, enablePrivIpEdit, cancelPrivIpEdit, savePrivIp,
    isEditingTag, newTagValue, enableTagEdit, cancelTag, saveTag, removeTag,
} = useNodeFieldEdit(() => props.node, emit, {
    aliasInput: aliasInputRef,
    pubIpInput: pubIpInputRef,
    privIpInput: privIpInputRef,
    tagInput: tagInputRef,
})
</script>

<template>
    <BaseWidget :class="[fleetWidgetStyles.frame, { 'opacity-50 pointer-events-none': isDeleting }]">
        <template #header>
            <div :class="fleetWidgetStyles.headerRow">
                <div class="flex items-center gap-2.5 min-w-0">
                    <BaseStatusDot :status="node.status" />
                    <div>
                        <div class="flex items-center gap-2">
                            <span v-if="!isEditingAlias" @click="enableAliasEdit" :class="fleetWidgetStyles.aliasTitle" :title="`Click to rename ${node.alias}`">{{ node.alias }}</span>
                            <input v-else ref="aliasInputRef" v-model="aliasValue" @keyup.enter="saveAlias" @keyup.esc="cancelAliasEdit" @blur="saveAlias" :class="fleetWidgetStyles.aliasInput" />
                            
                            <span v-if="node.hasPendingConfig" class="shrink-0 text-high" title="Pending sync, open node details to apply changes">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                            </span>
                            <span v-if="node.hasUpdateAvailable" class="relative flex h-2 w-2 shrink-0" title="Sensor Updates Available">
                                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-update-main opacity-75"></span>
                                <span class="relative inline-flex rounded-full h-2 w-2 bg-update-main"></span>
                            </span>
                            <svg v-if="node.isSilenced" class="w-4 h-4 text-medium shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" title="Node Silenced"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/></svg>
                        </div>
                    </div>
                </div>

                <BaseMeatballMenu v-if="node.id !== 'unassigned'" :id="`node-menu-${node.id}`">
                    <button @click="$emit('silence')" data-testid="widget-menu-silence" :class="[nodeMenuStyles.item, node.isSilenced ? 'text-archive-text hover:bg-archive-bg' : ' hover:text-text-h']">
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-12 group-active:-rotate-12 origin-top" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path v-if="!node.isSilenced" d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
                            <path v-if="node.isSilenced" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"/>
                        </svg>
                        {{ node.isSilenced ? 'Unsilence Node' : 'Silence Node' }}
                    </button>
                    <button v-if="node.hasUpdateAvailable" @click="$emit('upgradeAll')" data-testid="widget-menu-upgrade" :class="nodeMenuStyles.itemUpdate">
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-180" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                        Update Node
                    </button>
                    <button @click="$emit('delete')" data-testid="widget-menu-delete" :class="nodeMenuStyles.itemDanger">
                        <svg class="w-3.5 h-3.5 transition-transform duration-normal group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 6v14a2 2 0 002 2h10a2 2 0 002-2V6M10 11v6M14 11v6" /><path class="origin-bottom-right transition-transform duration-normal group-hover:-rotate-[15deg] group-hover:-translate-y-0.5" d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" /></svg>
                        <span v-if="isDeleting">Deleting...</span><span v-else>Delete</span>
                    </button>
                </BaseMeatballMenu>
            </div>
        </template>

        <div :class="fleetWidgetStyles.bodyWrap">
            <div v-if="node.id !== 'unassigned'" :class="fleetWidgetStyles.ipGrid">
                <div :class="[nodeHeaderStyles.ipPillBase, 'group/pub']">
                    <svg @click="node.publicIp ? handleCopy(node.id + '-pub', node.publicIp) : null" class="w-3.5 h-3.5 shrink-0" :class="node.publicIp ? 'cursor-pointer hover:text-primary-main' : 'opacity-50'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                    <span v-if="!isEditingPubIp" @click="enablePubIpEdit" class="font-mono truncate cursor-edit border-b border-dashed border-transparent hover:border-primary-main" :class="copiedStates[node.id + '-pub'] ? 'text-success-main' : ''" :title="copiedStates[node.id + '-pub'] ? 'Copied!' : 'Click to edit Public IP'">{{ copiedStates[node.id + '-pub'] ? 'Copied!' : (node.publicIp || 'Unknown') }}</span>
                    <input v-else ref="pubIpInputRef" v-model="pubIpValue" @keyup.enter="savePubIp" @keyup.esc="cancelPubIpEdit" @blur="savePubIp" :class="[nodeHeaderStyles.ipInput, 'truncate']" placeholder="0.0.0.0" />
                </div>
                <div :class="[nodeHeaderStyles.ipPillBase, 'group/priv']">
                    <svg @click="node.privateIp ? handleCopy(node.id + '-priv', node.privateIp) : null" class="w-3.5 h-3.5 shrink-0" :class="node.privateIp ? 'cursor-pointer hover:text-primary-main' : 'opacity-50'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="2" y="14" width="8" height="6" rx="2" ry="2"/><rect x="14" y="14" width="8" height="6" rx="2" ry="2"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 14v-2a2 2 0 012-2h8a2 2 0 012 2v2M12 2v8"/><rect x="8" y="2" width="8" height="6" rx="2" ry="2"/></svg>
                    <span v-if="!isEditingPrivIp" @click="enablePrivIpEdit" class="font-mono truncate cursor-edit border-b border-dashed border-transparent hover:border-primary-main" :class="copiedStates[node.id + '-priv'] ? 'text-success-main' : ''" :title="copiedStates[node.id + '-priv'] ? 'Copied!' : 'Click to edit Private IP'">{{ copiedStates[node.id + '-priv'] ? 'Copied!' : (node.privateIp || 'Unknown') }}</span>
                    <input v-else ref="privIpInputRef" v-model="privIpValue" @keyup.enter="savePrivIp" @keyup.esc="cancelPrivIpEdit" @blur="savePrivIp" :class="[nodeHeaderStyles.ipInput, 'truncate']" placeholder="0.0.0.0" />
                </div>
            </div>

            <div :class="fleetWidgetStyles.tagRow">
                <span v-for="(tag, index) in node.tags" :key="tag" :class="nodeHeaderStyles.tagPill">
                    {{ tag }}
                    <button @click.stop="removeTag(index)" class="opacity-0 group-hover/tag:opacity-100 text-text-m hover:text-danger-main transition-all outline-none focus:opacity-100">
                        <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                    </button>
                </span>
                <div v-if="isEditingTag" class="relative flex items-center">
                    <span class="absolute left-2 text-text-m text-sm pointer-events-none">+</span>
                    <input ref="tagInputRef" v-model="newTagValue" @keyup.enter="saveTag" @keyup.esc="cancelTag" @blur="cancelTag" :class="fleetWidgetStyles.tagInput" placeholder="tag name..." />
                </div>
                <button v-else @click.stop="enableTagEdit" :class="nodeHeaderStyles.addTagButton">+ Tag</button>
            </div>

            <div v-if="!node.isAwaitingCheckIn" :class="fleetWidgetStyles.sensorBox">
                <div class="flex items-center justify-between mb-2">
                    <span class="text-sm font-normal text-text-h">Deployed Sensors</span>
                    <span class="text-sm text-text-m">{{ node.onlineSensors }} / {{ node.totalSensors }} Online</span>
                </div>
                
                <div class="flex flex-wrap gap-1.5 mt-1">
                    <template v-if="isManifestLoading">
                        <div v-for="i in Math.min(node.totalSensors, 3)" :key="i" :class="fleetWidgetStyles.skeletonPill"></div>
                    </template>
                    <template v-else>
                        <div v-for="summary in node.sensorSummary" :key="summary.type" class="relative group/tooltip">
                            <span :class="fleetWidgetStyles.summaryPill"><span class="text-text-h">{{ summary.count }}</span> {{ summary.type }}</span>
                            <div :class="fleetWidgetStyles.tooltipBox">
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
            <div v-else :class="fleetWidgetStyles.awaitingBox">
                <svg class="w-6 h-6 text-text-h mb-2 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" /></svg>
                <span class="text-sm font-medium text-text-h">Awaiting Initial Check-in</span>
                <span class="text-sm text-text-m mt-1 max-w-[200px]">Deploy your first Sensor!</span>
            </div>
        </div>

        <template #footer>
            <button @click="$emit('openDetail')" :class="fleetWidgetStyles.footerButton">
                <span>{{ node.totalSensors === 0 ? 'Install First Sensor' : 'Manage Node & Sensors' }}</span>
                <svg class="w-4 h-4 transition-transform duration-normal group-hover/btn:translate-x-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
            </button>
        </template>
    </BaseWidget>
</template>
