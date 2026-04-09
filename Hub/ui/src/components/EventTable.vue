<script setup>
import { ref, computed, nextTick } from 'vue'

const props = defineProps({
    events: { type: Array, required: true },
    viewingArchive: { type: Boolean, required: true }
})

// Added 'mark-read' to the emits
const emit = defineEmits(['archive-all', 'archive-event', 'mark-read'])

const sortCol = ref('timestamp')
const sortDesc = ref(true)
const expandedRows = ref(new Set())

const toggleSort = (col) => {
    if (sortCol.value === col) {
        sortDesc.value = !sortDesc.value
    } else {
        sortCol.value = col
        sortDesc.value = ['timestamp', 'severity'].includes(col)
    }
}

const toggleRow = async (id) => {
    const newSet = new Set(expandedRows.value)
    const isExpanding = !newSet.has(id)
    
    if (isExpanding) {
        newSet.add(id)
        
        // BUG FIX: Optimistically mark as read and notify parent
        const eventTarget = props.events.find(e => e.id === id)
        if (eventTarget && !eventTarget.is_read) {
            eventTarget.is_read = true 
            emit('mark-read', id)
        }
    } else {
        newSet.delete(id)
    }
    expandedRows.value = newSet

    // Gentle Auto-Scroll
    if (isExpanding) {
        await nextTick()
        const detailsRow = document.getElementById(`details-${id}`)
        if (detailsRow) {
            detailsRow.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        }
    }
}

const isDownArrow = (col) => {
    if (sortCol.value !== col) return true; 
    return ['timestamp', 'severity'].includes(col) ? sortDesc.value : !sortDesc.value;
}

const getSeverityColor = (sev) => {
    const colors = { critical: '#f43f5e', high: '#fb923c', medium: '#eab308', low: '#3b82f6', info: '#64748b' }
    return colors[sev?.toLowerCase()] || 'transparent'
}

const sortedEvents = computed(() => {
    return [...props.events].sort((a, b) => {
        let valA = a[sortCol.value] || ''
        let valB = b[sortCol.value] || ''
        
        if (sortCol.value === 'severity') {
            const scores = { critical: 5, high: 4, medium: 3, low: 2, info: 1 }
            valA = scores[valA.toLowerCase()] || 0
            valB = scores[valB.toLowerCase()] || 0
        }
        
        if (valA < valB) return sortDesc.value ? 1 : -1
        if (valA > valB) return sortDesc.value ? -1 : 1
        return 0
    })
})

const formatEventType = (type) => type ? type.replace(/_/g, ' ') : ''
const formatString = (str) => str ? str.replace(/_/g, ' ') : ''
const formatJson = (val) => typeof val === 'object' ? JSON.stringify(val, null, 2) : val

const formatTime = (timestamp) => {
    if (!timestamp) return ''
    const dateObj = new Date(timestamp.replace(' ', 'T') + 'Z')
    const today = new Date()
    const isToday = dateObj.getDate() === today.getDate() && dateObj.getMonth() === today.getMonth() && dateObj.getFullYear() === today.getFullYear()
    
    const timeStr = timestamp.split(' ')[1]
    const dateStr = timestamp.split(' ')[0]
    return isToday ? timeStr : `${dateStr} ${timeStr}`
}
</script>

<template>
    <div class="bg-white dark:bg-[#0f0f11] border border-slate-200 dark:border-zinc-800 rounded-lg overflow-hidden flex flex-col shadow-sm w-full relative z-0">
        
        <div class="px-5 py-3 border-b border-slate-200 dark:border-zinc-800 flex justify-between items-center bg-white dark:bg-[#151518] shrink-0">
            <h3 class="text-sm font-semibold text-slate-800 dark:text-zinc-200">
                {{ viewingArchive ? 'Archived Events' : 'Active Threat Queue' }}
            </h3>
            <div class="justify-between items-center flex gap-2">
                <div v-show="!viewingArchive" class="hidden sm:flex items-center gap-2 pr-1">
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse dark:shadow-[0_0_8px_rgba(16,185,129,0.8)]"></span>
                    <span class="text-[11px] font-bold uppercase tracking-widest text-slate-500 dark:text-zinc-400">Live</span>
                </div>
                
                <button v-show="!viewingArchive && events.length > 0" @click="$emit('archive-all')"
                        class="px-2.5 py-1 rounded-md text-xs font-semibold text-slate-600 dark:text-zinc-400 bg-slate-100 dark:bg-zinc-800 hover:bg-amber-50 dark:hover:bg-amber-900/20 hover:text-amber-700 dark:hover:text-amber-400 transition-colors border border-slate-200 dark:border-zinc-700 hover:border-amber-300 dark:hover:border-amber-800/50 shadow-sm">
                    Archive All
                </button>
            </div>
        </div>

        <div class="overflow-x-auto custom-scroll max-h-[600px] lg:max-h-[700px]">
            <table class="w-full text-left border-separate border-spacing-0">
                <thead class="text-xs font-semibold text-slate-500 dark:text-zinc-400 sticky top-0 bg-slate-50 dark:bg-[#151518] z-30 shadow-[0_1px_0_0_#e2e8f0] dark:shadow-[0_1px_0_0_#27272a] select-none">
                    <tr>
                        <th class="px-3 py-3 w-8"></th>
                        <th @click="toggleSort('severity')" class="px-3 py-3 cursor-pointer hover:text-slate-800 dark:hover:text-zinc-200 transition-colors group">
                            <div class="flex items-center gap-1.5">Threat
                                <svg class="w-3 h-3 transition-transform duration-200" :class="[sortCol === 'severity' ? 'opacity-100 text-blue-500 dark:text-zinc-300' : 'opacity-0 group-hover:opacity-50 text-slate-400 dark:text-zinc-500', isDownArrow('severity') ? 'rotate-180' : '']" viewBox="0 0 384 512" fill="currentColor"><path d="M214.6 41.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 141.2V448c0 17.7 14.3 32 32 32s32-14.3 32-32V141.2L329.4 246.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z"></path></svg>
                            </div>
                        </th>
                        <th @click="toggleSort('event_type')" class="px-4 py-3 cursor-pointer hover:text-slate-800 dark:hover:text-zinc-200 transition-colors group">
                            <div class="flex items-center gap-1.5">Event Trigger
                                <svg class="w-3 h-3 transition-transform duration-200" :class="[sortCol === 'event_type' ? 'opacity-100 text-blue-500 dark:text-zinc-300' : 'opacity-0 group-hover:opacity-50 text-slate-400 dark:text-zinc-500', isDownArrow('event_type') ? 'rotate-180' : '']" viewBox="0 0 384 512" fill="currentColor"><path d="M214.6 41.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 141.2V448c0 17.7 14.3 32 32 32s32-14.3 32-32V141.2L329.4 246.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z"></path></svg>
                            </div>
                        </th>
                        <th @click="toggleSort('source')" class="px-4 py-3 cursor-pointer hover:text-slate-800 dark:hover:text-zinc-200 transition-colors group">
                            <div class="flex items-center gap-1.5">Source
                                <svg class="w-3 h-3 transition-transform duration-200" :class="[sortCol === 'source' ? 'opacity-100 text-blue-500 dark:text-zinc-300' : 'opacity-0 group-hover:opacity-50 text-slate-400 dark:text-zinc-500', isDownArrow('source') ? 'rotate-180' : '']" viewBox="0 0 384 512" fill="currentColor"><path d="M214.6 41.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 141.2V448c0 17.7 14.3 32 32 32s32-14.3 32-32V141.2L329.4 246.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z"></path></svg>
                            </div>
                        </th>
                        <th @click="toggleSort('target')" class="px-4 py-3 cursor-pointer hover:text-slate-800 dark:hover:text-zinc-200 transition-colors group">
                            <div class="flex items-center gap-1.5">Target
                                <svg class="w-3 h-3 transition-transform duration-200" :class="[sortCol === 'target' ? 'opacity-100 text-blue-500 dark:text-zinc-300' : 'opacity-0 group-hover:opacity-50 text-slate-400 dark:text-zinc-500', isDownArrow('target') ? 'rotate-180' : '']" viewBox="0 0 384 512" fill="currentColor"><path d="M214.6 41.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 141.2V448c0 17.7 14.3 32 32 32s32-14.3 32-32V141.2L329.4 246.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z"></path></svg>
                            </div>
                        </th>
                        <th @click="toggleSort('sensor_id')" class="px-4 py-3 text-right cursor-pointer hover:text-slate-800 dark:hover:text-zinc-200 transition-colors group">
                            <div class="flex items-center justify-end gap-1.5">Node
                                <svg class="w-3 h-3 transition-transform duration-200" :class="[sortCol === 'sensor_id' ? 'opacity-100 text-blue-500 dark:text-zinc-300' : 'opacity-0 group-hover:opacity-50 text-slate-400 dark:text-zinc-500', isDownArrow('sensor_id') ? 'rotate-180' : '']" viewBox="0 0 384 512" fill="currentColor"><path d="M214.6 41.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 141.2V448c0 17.7 14.3 32 32 32s32-14.3 32-32V141.2L329.4 246.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z"></path></svg>
                            </div>
                        </th>
                        <th @click="toggleSort('timestamp')" class="px-5 py-3 text-right cursor-pointer hover:text-slate-800 dark:hover:text-zinc-200 transition-colors group">
                            <div class="flex items-center justify-end gap-1.5">Time
                                <svg class="w-3 h-3 transition-transform duration-200" :class="[sortCol === 'timestamp' ? 'opacity-100 text-blue-500 dark:text-zinc-300' : 'opacity-0 group-hover:opacity-50 text-slate-400 dark:text-zinc-500', isDownArrow('timestamp') ? 'rotate-180' : '']" viewBox="0 0 384 512" fill="currentColor"><path d="M214.6 41.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 141.2V448c0 17.7 14.3 32 32 32s32-14.3 32-32V141.2L329.4 246.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z"></path></svg>
                            </div>
                        </th>
                        <th v-if="!viewingArchive" class="px-4 py-3 w-16"></th>
                    </tr>
                </thead>
                <tbody class="relative z-0">
                    <tr v-if="sortedEvents.length === 0">
                        <td :colspan="viewingArchive ? 7 : 8" class="px-5 py-8 border-b border-slate-200 dark:border-zinc-800/50 text-center text-slate-500 dark:text-zinc-500 text-sm">No events detected matching criteria.</td>
                    </tr>
                    
                    <template v-for="event in sortedEvents" :key="event.id">
                        <tr class="hover:bg-slate-50 dark:hover:bg-[#18181b] cursor-pointer transition-colors relative z-0 group"
                            :class="[ 'bleed-' + event.severity, expandedRows.has(event.id) ? 'bg-white dark:bg-[#18181b]' : '' ]"
                            @click="toggleRow(event.id)">
                            
                            <td class="px-3 py-3 border-l-[3px] text-slate-400 dark:text-zinc-500 transition-all duration-200" 
                                :class="expandedRows.has(event.id) ? 'border-b border-transparent' : 'border-b border-slate-300 dark:border-zinc-800/50'"
                                :style="{ borderLeftColor: getSeverityColor(event.severity) }">
                                <svg class="w-4 h-4 transition-transform duration-200" :class="expandedRows.has(event.id) ? 'rotate-90 text-slate-600 dark:text-zinc-300' : 'group-hover:text-slate-600 dark:group-hover:text-zinc-300'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
                            </td>

                            <td class="px-3 py-3 flex items-center gap-3" :class="expandedRows.has(event.id) ? 'border-b border-transparent' : 'border-b border-slate-200 dark:border-zinc-800/50'">
                                <div v-show="!event.is_read" class="w-1.5 h-1.5 rounded-full bg-rose-500 shrink-0 animate-pulse"></div>
                                <span class="px-2 py-0.5 rounded border text-[11px] font-semibold uppercase tracking-wider bg-slate-50 dark:bg-transparent whitespace-nowrap" :class="'severity-' + event.severity">{{ event.severity }}</span>
                            </td>
                            
                            <td class="px-4 py-3 text-sm text-slate-900 dark:text-zinc-100 capitalize max-w-[140px] md:max-w-[200px] lg:max-w-md xl:max-w-2xl 2xl:max-w-none truncate" :class="expandedRows.has(event.id) ? 'border-b border-transparent' : 'border-b border-slate-200 dark:border-zinc-800/50'" :title="formatEventType(event.event_type)">{{ formatEventType(event.event_type) }}</td>
                            <td class="px-4 py-3 text-sm text-slate-600 dark:text-zinc-400 mono max-w-[120px] md:max-w-[180px] lg:max-w-sm xl:max-w-xl 2xl:max-w-none truncate" :class="expandedRows.has(event.id) ? 'border-b border-transparent' : 'border-b border-slate-200 dark:border-zinc-800/50'" :title="event.source">{{ event.source }}</td>
                            <td class="px-4 py-3 text-sm text-slate-600 dark:text-zinc-400 mono max-w-[120px] md:max-w-[180px] lg:max-w-sm xl:max-w-xl 2xl:max-w-none truncate" :class="expandedRows.has(event.id) ? 'border-b border-transparent' : 'border-b border-slate-200 dark:border-zinc-800/50'" :title="event.target">{{ event.target }}</td>
                            <td class="px-4 py-3 text-sm text-right text-blue-600 dark:text-zinc-300 font-medium mono max-w-[140px] md:max-w-[200px] lg:max-w-sm xl:max-w-xl 2xl:max-w-none truncate" :class="expandedRows.has(event.id) ? 'border-b border-transparent' : 'border-b border-slate-200 dark:border-zinc-800/50'" :title="event.sensor_id">{{ event.sensor_id }}</td>
                            <td class="px-5 py-3 text-sm text-right text-slate-500 dark:text-zinc-500 mono whitespace-nowrap" :class="expandedRows.has(event.id) ? 'border-b border-transparent' : 'border-b border-slate-200 dark:border-zinc-800/50'" :title="event.timestamp">{{ formatTime(event.timestamp) }}</td>
                            
                            <td v-if="!viewingArchive" class="px-4 py-2 text-right w-16" :class="expandedRows.has(event.id) ? 'border-b border-transparent' : 'border-b border-slate-200 dark:border-zinc-800/50'">
                                <button @click.stop="$emit('archive-event', event.id)"
                                        class="flex items-center justify-center w-6 h-6 ml-auto rounded-md bg-white dark:bg-[#1f1f22] border border-slate-200 dark:border-zinc-700 text-slate-500 dark:text-zinc-400 hover:bg-amber-50 dark:hover:bg-amber-900/20 hover:border-amber-300 dark:hover:border-amber-700/50 hover:text-amber-600 dark:hover:text-amber-400 transition-all duration-200 shadow-sm active:scale-95"
                                        title="Archive Event">
                                    <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <path d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"></path>
                                    </svg>
                                </button>
                            </td>
                        </tr>

                        <tr v-if="expandedRows.has(event.id)" :id="'details-' + event.id">
                            <td :colspan="viewingArchive ? 7 : 8" class="p-0 border-b border-slate-300 dark:border-zinc-950 bg-slate-50 dark:bg-[#18181b]">
                                <div class="pl-11 pr-5 pb-5 pt-2">
                                    <div class="px-6 py-5 bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-zinc-800/80 rounded-lg shadow-sm relative overflow-hidden">
                                        
                                        <div class="absolute left-0 top-0 bottom-0 w-1" :style="{ backgroundColor: getSeverityColor(event.severity) }"></div>


                                        
                                        <div class="flex flex-wrap gap-x-8 gap-y-6">
                                            <div v-for="(val, key) in event.details" :key="key" class="flex flex-col group max-w-full">
                                                
                                                <div class="flex items-center gap-1.5 mb-1.5">
                                                    <span class="w-1 h-1 rounded-full bg-slate-300 dark:bg-zinc-600 transition-colors group-hover:bg-blue-500 dark:group-hover:bg-slate-100 shrink-0"></span>
                                                    <span class="text-[10px] font-bold text-slate-500 dark:text-zinc-500 uppercase tracking-widest truncate">{{ formatString(key) }}</span>
                                                </div>
                                                
                                                <div v-if="Array.isArray(val)" class="space-y-1.5 w-fit min-w-[150px] max-w-full">
                                                    <pre v-for="(item, index) in val.slice(0, 5)" :key="index"
                                                         class="bg-slate-50 dark:bg-[#121214] border border-slate-200 dark:border-zinc-800/60 rounded p-2.5 text-[11px] text-emerald-700 dark:text-emerald-400 mono overflow-x-auto custom-scroll whitespace-pre-wrap break-all shadow-inner w-fit min-w-[150px] max-w-full">{{ formatJson(item) }}</pre>
                                                    <div v-show="val.length > 5" class="text-[10px] text-slate-400 dark:text-zinc-600 font-medium pt-1 italic">+ {{ val.length - 5 }} items truncated</div>
                                                </div>
                                                
                                                <div v-else class="text-[11px] text-slate-800 dark:text-zinc-300 mono break-all bg-slate-50 dark:bg-[#121214] border border-slate-200 dark:border-zinc-800/60 p-2.5 rounded whitespace-pre-wrap max-h-40 overflow-y-auto custom-scroll shadow-inner w-fit min-w-[150px] max-w-[600px] xl:max-w-[800px]">{{ formatJson(val) }}</div>
                                            </div>
                                        </div>
                                        
                                        <div class="mt-5 pt-3 border-t border-slate-100 dark:border-zinc-800/50 flex justify-between items-center text-[10px] text-slate-400 dark:text-zinc-500 mono">
                                            <div class="flex items-center gap-2">
                                                <span>Trace ID: {{ event.id }}</span>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </td>
                        </tr>
                    </template>
                </tbody>
            </table>
        </div>
    </div>
</template>