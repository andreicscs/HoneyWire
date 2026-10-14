<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppStore } from '../../stores/System/app.ts'
import { useEventsStore } from '../../stores/Events/events.ts'
import { useFleetStore } from '../../stores/Fleet/fleet.ts'
import { formatSensorId } from '../../utils/formatSensorId'
import { formatEventType } from '../../utils/formatEventType'
import EventSortHeader from './EventSortHeader.vue'
import { eventTableStyles } from './dashboardWidgetsStyles.ts'


const appStore = useAppStore()
const eventsStore = useEventsStore()
const fleetStore = useFleetStore()

const { viewingArchive } = storeToRefs(appStore)
const { filteredEvents: rawEvents, isFetching } = storeToRefs(eventsStore)
const { nodes: fleet } = storeToRefs(fleetStore)

const sortCol = ref('timestamp')
const sortDesc = ref(true)
const expandedRows = ref<Set<string>>(new Set())

const currentPage = ref(1)
const itemsPerPage = ref(50)

const displayEvents = ref<any[]>([])

watch([rawEvents, isFetching], ([newEvents, fetching]) => {
    // Freeze the UI on the old data if the filter yields an empty array while fetching
    if (fetching && newEvents.length === 0 && displayEvents.value.length > 0) {
        return 
    }
    displayEvents.value = newEvents
}, { immediate: true })

// Resolve nodeId → alias for display
const getNodeAlias = (nodeId: string | null) => {
    if (!nodeId) return 'Unassigned'
    const node = fleet.value.find(n => n.id === nodeId)
    return node?.alias || nodeId
}

watch([viewingArchive, sortCol, sortDesc, () => displayEvents.value.length], () => {
    if (displayEvents.value.length === 0) currentPage.value = 1
    expandedRows.value = new Set()
})

const toggleSort = (col: string) => {
    if (sortCol.value === col) sortDesc.value = !sortDesc.value
    else { sortCol.value = col; sortDesc.value = ['timestamp', 'severity'].includes(col) }
}

const rowBorderClass = (eventId: string) =>
    expandedRows.value.has(eventId) ? eventTableStyles.rowBorderOpen : eventTableStyles.rowBorderClosed

const toggleRow = async (id: string) => {
    const newSet = new Set(expandedRows.value)
    const isExpanding = !newSet.has(id)
    
    if (isExpanding) {
        newSet.add(id)
        const eventTarget = displayEvents.value.find((e: any) => e.id === id)
        if (eventTarget && !eventTarget.isRead) eventsStore.markEventRead(id)
    } else {
        newSet.delete(id)
    }
    expandedRows.value = newSet

    if (isExpanding) {
        await nextTick()
        const detailsRow = document.getElementById(`details-${id}`)
        if (detailsRow) detailsRow.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    }
}

const sortedEvents = computed(() => {
    return [...displayEvents.value].sort((a, b) => {
        let valA: any = (a as any)[sortCol.value] || ''
        let valB: any = (b as any)[sortCol.value] || ''
        if (sortCol.value === 'severity') {
            const scores = { critical: 5, high: 4, medium: 3, low: 2, info: 1 }
            valA = scores[valA.toLowerCase() as keyof typeof scores] || 0
            valB = scores[valB.toLowerCase() as keyof typeof scores] || 0
        }
        if (sortCol.value === 'nodeId') {
            // Sort by alias for display consistency, fallback to nodeId
            valA = getNodeAlias(a.nodeId)
            valB = getNodeAlias(b.nodeId)
        }
        if (valA < valB) return sortDesc.value ? 1 : -1
        if (valA > valB) return sortDesc.value ? -1 : 1
        return 0
    })
})

const totalPages = computed(() => Math.ceil(sortedEvents.value.length / itemsPerPage.value) || 1)

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

const prevPage = () => { if (currentPage.value > 1) currentPage.value-- }
const nextPage = () => { if (currentPage.value < totalPages.value) currentPage.value++ }

const goToPage = (page: number | string) => {
    if (typeof page === 'number') {
        currentPage.value = page
    }
}

const handleArchiveAll = async () => {
    if (!confirm('Archive all currently active events?')) return
    const res = await eventsStore.archiveAll()
    if (!res.success) alert(res.error)
}

const handleArchiveEvent = async (eventId: string) => {
    const res = await eventsStore.archiveEvent(eventId)
    if (!res.success) alert(res.error)
}

// Formatters
const formatString = (str: string | number) => str ? String(str).replace(/_/g, ' ') : ''
const formatJson = (val: any) => {
    if (val === null) return 'null'
    if (val === undefined) return 'undefined'
    return typeof val === 'object' ? JSON.stringify(val, null, 2) : String(val)
}
const getDataType = (val: any) => {
    if (val === null || val === undefined) return 'primitive'
    if (Array.isArray(val)) {
        if (val.length > 0 && typeof val[0] === 'object' && val[0] !== null) return 'object_array'
        return 'primitive_array'
    }
    if (typeof val === 'object') return 'object'
    return 'primitive'
}
const handleExport = () => {
    const url = new URL('/api/v2/events/export', window.location.origin)
    url.searchParams.append('archived', viewingArchive.value ? 'true' : 'false')
    if (fleetStore.selectedNodeId) url.searchParams.append('nodeId', fleetStore.selectedNodeId)
    if (fleetStore.selectedSensorId) url.searchParams.append('sensorId', fleetStore.selectedSensorId)
    const a = document.createElement('a')
    a.href = url.toString()
    a.download = 'honeywire_events.json'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
}

const formatTime = (timestamp: string) => {
    if (!timestamp) return ''
    const dateObj = new Date(timestamp)
    const now = new Date()

    const isToday = 
        dateObj.getDate() === now.getDate() && 
        dateObj.getMonth() === now.getMonth() && 
        dateObj.getFullYear() === now.getFullYear()

    if (isToday) {
        return new Intl.DateTimeFormat('default', {
            hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false 
        }).format(dateObj)
    }

    return new Intl.DateTimeFormat('default', {
        month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false 
    }).format(dateObj)
}
</script>

<template>
    <div :class="eventTableStyles.card">
        
        <div :class="eventTableStyles.headBar">
            <div class="flex items-center gap-3">
                <h3 class="text-base font-medium text-text-h">
                    {{ viewingArchive ? 'Archived Events' : 'Active Threat Queue' }}
                </h3>
                <div v-show="!viewingArchive" class="hidden sm:flex items-center gap-2">
                    <span class="w-1.5 h-1.5 rounded-full bg-success-main animate-pulse shadow-[0_0_8px_var(--color-success-main)]"></span>
                    <span class="text-sm tracking-wide text-text-m">Live</span>
                </div>
            </div>
            
            <div class="flex items-center gap-2">
                <button v-show="displayEvents.length > 0" @click="handleExport"
                        type="button"
                        title="Export JSON"
                        aria-label="Export all events as JSON"
                        :class="eventTableStyles.exportButton">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path></svg>
                </button>
                <button v-show="!viewingArchive && displayEvents.length > 0" @click="handleArchiveAll"
                        type="button"
                        aria-label="Archive all active events"
                        :class="eventTableStyles.archiveAllButton">
                    Archive All
                </button>
            </div>
        </div>

        <div :class="eventTableStyles.tableScroll">
            <table class="min-w-full text-left border-separate border-spacing-0">
                <thead :class="eventTableStyles.tableHead">
                    <tr>
                        <th class="px-3 py-3 w-8"></th>
                        <EventSortHeader label="Threat" sort-key="severity" aria-label="Sort by threat severity" pad-class="px-3 py-3" :sort-col="sortCol" :sort-desc="sortDesc" @toggle="toggleSort" />
                        <EventSortHeader label="Event Trigger" sort-key="eventTrigger" aria-label="Sort by event trigger type" :sort-col="sortCol" :sort-desc="sortDesc" @toggle="toggleSort" />
                        <EventSortHeader label="Source" sort-key="source" :sort-col="sortCol" :sort-desc="sortDesc" @toggle="toggleSort" />
                        <EventSortHeader label="Target" sort-key="target" :sort-col="sortCol" :sort-desc="sortDesc" @toggle="toggleSort" />
                        
                        <EventSortHeader label="Sensor" sort-key="sensorId" aria-label="Sort by sensor id" :sort-col="sortCol" :sort-desc="sortDesc" @toggle="toggleSort" />

                        <EventSortHeader label="Node" sort-key="nodeId" align="right" :sort-col="sortCol" :sort-desc="sortDesc" @toggle="toggleSort" />

                        <EventSortHeader label="Time" sort-key="timestamp" aria-label="Sort by timestamp" align="right" pad-class="px-5 py-3" :sort-col="sortCol" :sort-desc="sortDesc" @toggle="toggleSort" />
                        <th v-if="!viewingArchive" class="px-4 py-3 w-16"></th>
                    </tr>
                </thead>
                
                <tbody class="relative z-0">
                    <tr v-if="displayEvents.length === 0 && !isFetching">
                        <td :colspan="viewingArchive ? 8 : 9" class="px-5 py-8 border-b border-border-default text-center text-text-m text-base">No events.</td>
                    </tr>
                    
                    <template v-for="event in paginatedEvents" :key="event.id">
                        <tr data-testid="event-row" :data-event-id="event.id" :class="[eventTableStyles.rowBase, 'bleed-' + event.severity.toLowerCase(), expandedRows.has(event.id) ? eventTableStyles.rowExpanded : '']"
                            @click="toggleRow(event.id)">
                            
                            <td class="px-3 py-3 border-l-[3px] text-text-m transition-all duration-[var(--duration-fast)]" 
                                :class="rowBorderClass(event.id)"
                                :style="{ borderLeftColor: event.severity ? `var(--color-${event.severity.toLowerCase()})` : 'transparent' }">
                                <svg class="w-4 h-4 transition-transform duration-normal" :class="expandedRows.has(event.id) ? 'rotate-90 text-text-h' : 'group-hover:text-text-h'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
                            </td>

                            <td class="px-3 py-3 flex items-center gap-3" :class="rowBorderClass(event.id)">
                                <div v-show="!event.isRead" class="w-1.5 h-1.5 rounded-full bg-danger-main shrink-0 animate-pulse"></div>
                                <span class="px-2 py-0.5 rounded border text-base bg-bg-surface whitespace-nowrap capitalize" 
                                      :style="{ borderColor: `var(--color-${event.severity.toLowerCase()})`, color: `var(--color-${event.severity.toLowerCase()})` }">
                                    {{ event.severity }}
                                </span>
                            </td>
                            
                            <td class="px-4 py-3 text-base font-base text-text-h capitalize max-w-[140px] md:max-w-[200px] lg:max-w-md xl:max-w-2xl 2xl:max-w-none truncate" :class="rowBorderClass(event.id)" :title="formatEventType(event.eventTrigger)">{{ formatEventType(event.eventTrigger) }}</td>
                            
                            <td class="px-4 py-3 text-base text-text-m font-mono max-w-[120px] md:max-w-[180px] lg:max-w-sm xl:max-w-xl 2xl:max-w-none truncate" :class="rowBorderClass(event.id)" :title="event.source">{{ event.source }}</td>
                            <td class="px-4 py-3 text-base text-text-m font-mono max-w-[120px] md:max-w-[180px] lg:max-w-sm xl:max-w-xl 2xl:max-w-none truncate" :class="rowBorderClass(event.id)" :title="event.target">{{ event.target }}</td>
                            
                            <td class="px-4 py-3 text-base text-text-h font-mono max-w-[140px] md:max-w-[200px] lg:max-w-sm xl:max-w-xl 2xl:max-w-none truncate" :class="rowBorderClass(event.id)" :title="event.sensorId">{{ formatSensorId(event.sensorId) }}</td>
                            
                            <td class="px-4 py-3 text-base font-medium text-right text-text-h font-mono w-[160px] min-w-[160px] max-w-[160px] truncate" :class="rowBorderClass(event.id)" :title="event.nodeId || 'Unassigned'">{{ getNodeAlias(event.nodeId) }}</td>
                            
                            <td class="px-5 py-3 text-base text-right text-text-m font-mono whitespace-nowrap" :class="rowBorderClass(event.id)" :title="event.timestamp">{{ formatTime(event.timestamp) }}</td>
                            
                            <td v-if="!viewingArchive" class="px-4 py-2 text-right w-16" :class="rowBorderClass(event.id)">
                                <button @click.stop="handleArchiveEvent(event.id)"                                     
                                        type="button"
                                        aria-label="Archive this event"                                        
                                        :class="eventTableStyles.archiveRowButton"
                                        title="Archive Event">
                                    <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                        <path d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"></path>
                                    </svg>
                                </button>
                            </td>
                        </tr>

                       <tr v-if="expandedRows.has(event.id)" :id="'details-' + event.id" data-testid="event-details">
                            <td :colspan="viewingArchive ? 8 : 9" class="p-0 border-b border-border-default bg-bg-base">
                                <div class="pl-11 pr-5 pb-5 pt-2">
                                    
                                    <div class="px-6 py-5 bg-bg-surface border border-border-default rounded-lg shadow-sm relative overflow-hidden w-fit max-w-full">
                                        <div class="absolute left-0 top-0 bottom-0 w-1" :style="{ backgroundColor: event.severity ? `var(--color-${event.severity.toLowerCase()})` : 'transparent' }"></div>
                                        
                                        <div class="flex flex-wrap gap-x-6 gap-y-6">
                                            
                                            <div v-for="(val, key) in event.details" :key="key" class="flex flex-col group w-fit min-w-[120px] max-w-full">
                                                <div class="flex items-center gap-1.5 mb-2">
                                                    <span class="w-1 h-1 rounded-full bg-border-default transition-colors group-hover:bg-highlight-border shrink-0"></span>
                                                    <span class="text-sm font-medium text-text-m capitalize truncate">{{ formatString(key) }}</span>
                                                </div>
                                                
                                                <div v-if="getDataType(val) === 'primitive_array'" class="flex flex-wrap gap-2">
                                                    <span v-for="(item, i) in val" :key="i" class="px-2 py-1 bg-bg-inset border border-border-default rounded text-base text-text-h font-mono break-all shadow-sm">
                                                        {{ String(item) }}
                                                    </span>
                                                </div>
                                                
                                                <div v-else-if="getDataType(val) === 'object_array'" class="flex flex-col gap-2">
                                                    <div v-for="(obj, i) in val" :key="i" class="bg-bg-inset border border-border-default rounded p-3 text-base font-mono shadow-inner overflow-x-auto w-fit max-w-full">
                                                        <div class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1.5">
                                                            <template v-for="(subVal, subKey) in obj" :key="subKey">
                                                                <div class="text-text-m whitespace-nowrap">{{ subKey }}:</div>
                                                                <div class="text-text-h break-words whitespace-pre-wrap">{{ formatJson(subVal) }}</div>
                                                            </template>
                                                        </div>
                                                    </div>
                                                </div>
                                                
                                                <div v-else-if="getDataType(val) === 'object'" class="bg-bg-inset border border-border-default rounded p-3 text-base font-mono shadow-inner overflow-x-auto w-fit max-w-full">
                                                    <div class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2">
                                                        <template v-for="(subVal, subKey) in val" :key="subKey">
                                                            <div class="text-text-m whitespace-nowrap border-b border-border-default pb-1.5">{{ subKey }}</div>
                                                            <div class="text-text-h break-words whitespace-pre-wrap border-b border-border-default pb-1.5">{{ formatJson(subVal) }}</div>
                                                        </template>
                                                    </div>
                                                </div>
                                                
                                                <div v-else class="bg-bg-inset border border-border-default rounded px-3 py-2 text-base text-text-h font-mono whitespace-pre-wrap break-words shadow-inner w-fit max-w-full inline-block">
                                                    {{ String(val) }}
                                                </div>
                                            </div>
                                            
                                        </div>
                                        <div class="mt-5 pt-3 border-t border-border-default flex justify-between items-center text-sm text-text-m font-mono">
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

        <div v-if="sortedEvents.length > itemsPerPage" :class="eventTableStyles.pageBar">
            <div class="flex flex-1 justify-between sm:hidden">
                <button @click="prevPage" :disabled="currentPage === 1"
                        type="button"
                        :class="eventTableStyles.pageButtonMobile">
                    Previous
                </button>
                <button @click="nextPage" :disabled="currentPage === totalPages"
                        type="button"
                        :class="[eventTableStyles.pageButtonMobile, 'ml-3']">
                    Next
                </button>
            </div>
            <div class="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
                <div>
                    <p class="text-base text-text-m">
                        Showing <span class="font-medium text-text-h">{{ (currentPage - 1) * itemsPerPage + 1 }}</span> to <span class="font-medium text-text-h">{{ Math.min(currentPage * itemsPerPage, sortedEvents.length) }}</span> of <span class="font-medium text-text-h">{{ sortedEvents.length }}</span> events
                    </p>
                </div>
                <div>
                    <nav class="isolate inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
                        <button @click="prevPage" :disabled="currentPage === 1" 
                                :class="[eventTableStyles.pageArrow, 'rounded-l-md']">
                            <span class="sr-only">Previous</span>
                            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M11.78 5.22a.75.75 0 0 1 0 1.06L8.06 10l3.72 3.72a.75.75 0 1 1-1.06 1.06l-4.25-4.25a.75.75 0 0 1 0-1.06l4.25-4.25a.75.75 0 0 1 1.06 0Z" clip-rule="evenodd" /></svg>
                        </button>
                        <template v-for="(page, idx) in visiblePages" :key="idx">
                            <span v-if="page === '...'" 
                                  class="relative inline-flex items-center px-3.5 py-1.5 text-base text-text-m border border-border-default bg-bg-surface">
                                ...
                            </span>
                            <button v-else @click="goToPage(page)"
                                    :class="[eventTableStyles.pageNumber, currentPage === page ? eventTableStyles.pageActive : eventTableStyles.pageInactive]">
                                {{ page }}
                            </button>
                        </template>
                        <button @click="nextPage" :disabled="currentPage === totalPages" 
                                :class="[eventTableStyles.pageArrow, 'rounded-r-md']">
                            <span class="sr-only">Next</span>
                            <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                                <path fill-rule="evenodd" d="M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" />
                            </svg>
                        </button>
                    </nav>
                </div>
            </div>
        </div>
    </div>
</template>