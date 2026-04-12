<script setup>
    import { ref, computed, nextTick, watch } from 'vue'

    const props = defineProps({
        events: { type: Array, required: true },
        viewingArchive: { type: Boolean, required: true }
    })

    const emit = defineEmits(['archive-all', 'archive-event', 'mark-read'])

    const sortCol = ref('timestamp')
    const sortDesc = ref(true)
    const expandedRows = ref(new Set())

    // --- PAGINATION STATE ---
    const currentPage = ref(1)
    const itemsPerPage = ref(50)

    watch([() => props.viewingArchive, sortCol, sortDesc], () => {
        currentPage.value = 1
        expandedRows.value = new Set()
    })

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
            const eventTarget = props.events.find(e => e.id === id)
            if (eventTarget && !eventTarget.is_read) {
                eventTarget.is_read = true 
                emit('mark-read', id)
            }
        } else {
            newSet.delete(id)
        }
        expandedRows.value = newSet

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

    // --- PAGINATION COMPUTATIONS ---
    const totalPages = computed(() => Math.ceil(sortedEvents.value.length / itemsPerPage.value))

    const paginatedEvents = computed(() => {
        const start = (currentPage.value - 1) * itemsPerPage.value
        const end = start + itemsPerPage.value
        return sortedEvents.value.slice(start, end)
    })

    const visiblePages = computed(() => {
        const total = totalPages.value;
        const current = currentPage.value;
        
        if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
        
        if (current <= 4) return [1, 2, 3, 4, 5, '...', total];
        if (current >= total - 3) return [1, '...', total - 4, total - 3, total - 2, total - 1, total];
        
        return [1, '...', current - 1, current, current + 1, '...', total];
    });

    const prevPage = () => {
        if (currentPage.value > 1) currentPage.value--
    }

    const nextPage = () => {
        if (currentPage.value < totalPages.value) currentPage.value++
    }

    // --- PAYLOAD INSPECTOR HELPERS ---
    const formatEventType = (type) => type ? type.replace(/_/g, ' ') : ''
    const formatString = (str) => str ? str.replace(/_/g, ' ') : ''
    
    const formatJson = (val) => {
        if (val === null) return 'null'
        if (val === undefined) return 'undefined'
        return typeof val === 'object' ? JSON.stringify(val, null, 2) : String(val)
    }

    const getDataType = (val) => {
        if (val === null || val === undefined) return 'primitive'
        if (Array.isArray(val)) {
            if (val.length > 0 && typeof val[0] === 'object' && val[0] !== null) return 'object_array'
            return 'primitive_array'
        }
        if (typeof val === 'object') return 'object'
        return 'primitive'
    }

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

        <div class="overflow-x-auto overflow-y-auto custom-scroll max-h-[600px] lg:max-h-[700px] flex-1">
            <table class="w-full text-left border-separate border-spacing-0">
                <thead class="text-xs font-semibold text-slate-500 dark:text-zinc-400 sticky top-0 bg-slate-50 dark:bg-[#151518] z-30 shadow-[0_1px_0_0_#e2e8f0] dark:shadow-[0_1px_0_0_#27272a] select-none">
                    <tr>
                        <th class="px-3 py-3 w-8"></th>
                        <th @click="toggleSort('severity')" class="px-3 py-3 cursor-pointer hover:text-slate-800 dark:hover:text-zinc-200 transition-colors group">
                            <div class="flex items-center gap-1.5">Threat
                                <svg class="w-3 h-3 transition-transform duration-200" :class="[sortCol === 'severity' ? 'opacity-100 text-blue-500 dark:text-zinc-300' : 'opacity-0 group-hover:opacity-50 text-slate-400 dark:text-zinc-500', isDownArrow('severity') ? 'rotate-180' : '']" viewBox="0 0 384 512" fill="currentColor"><path d="M214.6 41.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 141.2V448c0 17.7 14.3 32 32 32s32-14.3 32-32V141.2L329.4 246.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z"></path></svg>
                            </div>
                        </th>
                        <th @click="toggleSort('event_trigger')" class="px-4 py-3 cursor-pointer hover:text-slate-800 dark:hover:text-zinc-200 transition-colors group">
                            <div class="flex items-center gap-1.5">Event Trigger
                                <svg class="w-3 h-3 transition-transform duration-200" :class="[sortCol === 'event_trigger' ? 'opacity-100 text-blue-500 dark:text-zinc-300' : 'opacity-0 group-hover:opacity-50 text-slate-400 dark:text-zinc-500', isDownArrow('event_trigger') ? 'rotate-180' : '']" viewBox="0 0 384 512" fill="currentColor"><path d="M214.6 41.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 141.2V448c0 17.7 14.3 32 32 32s32-14.3 32-32V141.2L329.4 246.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z"></path></svg>
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
                    <tr v-if="paginatedEvents.length === 0">
                        <td :colspan="viewingArchive ? 7 : 8" class="px-5 py-8 border-b border-slate-200 dark:border-zinc-800/50 text-center text-slate-500 dark:text-zinc-500 text-sm">No events detected matching criteria.</td>
                    </tr>
                    
                    <template v-for="event in paginatedEvents" :key="event.id">
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
                            
                            <td class="px-4 py-3 text-sm text-slate-900 dark:text-zinc-100 capitalize max-w-[140px] md:max-w-[200px] lg:max-w-md xl:max-w-2xl 2xl:max-w-none truncate" :class="expandedRows.has(event.id) ? 'border-b border-transparent' : 'border-b border-slate-200 dark:border-zinc-800/50'" :title="formatEventType(event.event_trigger)">{{ formatEventType(event.event_trigger) }}</td>
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

                                        <div class="flex flex-wrap gap-x-6 gap-y-6">
                                            
                                            <div v-for="(val, key) in event.details" :key="key" class="flex flex-col group w-fit min-w-[120px] max-w-full">
                                                
                                                <div class="flex items-center gap-1.5 mb-2">
                                                    <span class="w-1 h-1 rounded-full bg-slate-300 dark:bg-zinc-600 transition-colors group-hover:bg-blue-500 dark:group-hover:bg-slate-100 shrink-0"></span>
                                                    <span class="text-[10px] font-bold text-slate-500 dark:text-zinc-500 uppercase tracking-widest truncate">{{ formatString(key) }}</span>
                                                </div>
                                                
                                                <div v-if="getDataType(val) === 'primitive_array'" class="flex flex-wrap gap-2">
                                                    <span v-for="(item, i) in val" :key="i"
                                                          class="px-2 py-1 bg-slate-100 dark:bg-zinc-800/40 border border-slate-200 dark:border-zinc-700/50 rounded text-[11px] text-slate-700 dark:text-zinc-300 mono break-all shadow-sm">
                                                        {{ String(item) }}
                                                    </span>
                                                </div>

                                                <div v-else-if="getDataType(val) === 'object_array'" class="flex flex-col gap-2 w-full">
                                                    <div v-for="(obj, i) in val" :key="i" class="bg-slate-50 dark:bg-[#121214] border border-slate-200 dark:border-zinc-800/60 rounded p-3 text-[11px] mono shadow-inner overflow-x-auto w-full">
                                                        <div class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1.5">
                                                            <template v-for="(subVal, subKey) in obj" :key="subKey">
                                                                <div class="text-slate-500 dark:text-zinc-500 font-medium whitespace-nowrap">{{ subKey }}:</div>
                                                                <div class="text-slate-800 dark:text-zinc-300 break-words whitespace-pre-wrap">{{ formatJson(subVal) }}</div>
                                                            </template>
                                                        </div>
                                                    </div>
                                                </div>

                                                <div v-else-if="getDataType(val) === 'object'" class="bg-slate-50 dark:bg-[#121214] border border-slate-200 dark:border-zinc-800/60 rounded p-3 text-[11px] mono shadow-inner overflow-x-auto w-full">
                                                    <div class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2">
                                                        <template v-for="(subVal, subKey) in val" :key="subKey">
                                                            <div class="text-slate-500 dark:text-zinc-500 font-medium whitespace-nowrap border-b border-slate-200/50 dark:border-zinc-800/50 pb-1.5">{{ subKey }}</div>
                                                            <div class="text-slate-800 dark:text-zinc-300 break-words whitespace-pre-wrap border-b border-slate-200/50 dark:border-zinc-800/50 pb-1.5">{{ formatJson(subVal) }}</div>
                                                        </template>
                                                    </div>
                                                </div>

                                                <div v-else class="bg-slate-50 dark:bg-[#121214] border border-slate-200 dark:border-zinc-800/60 rounded px-3 py-2 text-[11px] text-slate-800 dark:text-zinc-300 mono whitespace-pre-wrap break-words shadow-inner w-fit max-w-full inline-block">
                                                    {{ String(val) }}
                                                </div>

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

        <div v-if="sortedEvents.length > itemsPerPage" class="flex items-center justify-between border-t border-slate-200 dark:border-zinc-800 bg-white dark:bg-[#151518] px-4 py-3 sm:px-5 shrink-0">
            
            <div class="flex flex-1 justify-between sm:hidden">
                <button @click="prevPage" :disabled="currentPage === 1" 
                        class="relative inline-flex items-center rounded-md border border-slate-200 dark:border-zinc-700 bg-white dark:bg-[#1f1f22] px-4 py-2 text-sm font-medium text-slate-700 dark:text-zinc-300 hover:bg-slate-50 dark:hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors">
                    Previous
                </button>
                <button @click="nextPage" :disabled="currentPage === totalPages" 
                        class="relative ml-3 inline-flex items-center rounded-md border border-slate-200 dark:border-zinc-700 bg-white dark:bg-[#1f1f22] px-4 py-2 text-sm font-medium text-slate-700 dark:text-zinc-300 hover:bg-slate-50 dark:hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors">
                    Next
                </button>
            </div>

            <div class="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
                <div>
                    <p class="text-[11px] font-medium text-slate-500 dark:text-zinc-400">
                        Showing <span class="font-bold text-slate-700 dark:text-zinc-300">{{ (currentPage - 1) * itemsPerPage + 1 }}</span> to <span class="font-bold text-slate-700 dark:text-zinc-300">{{ Math.min(currentPage * itemsPerPage, sortedEvents.length) }}</span> of <span class="font-bold text-slate-700 dark:text-zinc-300">{{ sortedEvents.length }}</span> events
                    </p>
                </div>
                <div>
                    <nav class="isolate inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
                        
                        <button @click="prevPage" :disabled="currentPage === 1" 
                                class="relative inline-flex items-center rounded-l-md px-2 py-1.5 border border-slate-200 dark:border-zinc-700 bg-white dark:bg-[#18181b] text-slate-400 dark:text-zinc-500 hover:bg-slate-50 dark:hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors focus:z-20 outline-none">
                            <span class="sr-only">Previous</span>
                            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M11.78 5.22a.75.75 0 0 1 0 1.06L8.06 10l3.72 3.72a.75.75 0 1 1-1.06 1.06l-4.25-4.25a.75.75 0 0 1 0-1.06l4.25-4.25a.75.75 0 0 1 1.06 0Z" clip-rule="evenodd" /></svg>
                        </button>
                        
                        <template v-for="(page, idx) in visiblePages" :key="idx">
                            <span v-if="page === '...'" 
                                  class="relative inline-flex items-center px-3.5 py-1.5 text-xs font-semibold text-slate-400 dark:text-zinc-500 border border-slate-200 dark:border-zinc-700 bg-white dark:bg-[#18181b]">
                                ...
                            </span>
                            <button v-else @click="currentPage = page"
                                    class="relative inline-flex items-center px-3.5 py-1.5 text-xs font-semibold border border-slate-200 dark:border-zinc-700 transition-colors focus:z-20 outline-none"
                                    :class="currentPage === page ? 'z-10 bg-slate-100 dark:bg-zinc-800 text-slate-900 dark:text-zinc-100 shadow-inner' : 'bg-white dark:bg-[#18181b] text-slate-600 dark:text-zinc-400 hover:bg-slate-50 dark:hover:bg-zinc-800/80'">
                                {{ page }}
                            </button>
                        </template>

                        <button @click="nextPage" :disabled="currentPage === totalPages" 
                                class="relative inline-flex items-center rounded-r-md px-2 py-1.5 border border-slate-200 dark:border-zinc-700 bg-white dark:bg-[#18181b] text-slate-400 dark:text-zinc-500 hover:bg-slate-50 dark:hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors focus:z-20 outline-none">
                            <span class="sr-only">Next</span>
                            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" /></svg>
                        </button>

                    </nav>
                </div>
            </div>
        </div>

    </div>
</template>