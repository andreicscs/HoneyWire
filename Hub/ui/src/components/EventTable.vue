<script setup>
defineProps({
    events: { type: Array, required: true },
    viewingArchive: { type: Boolean, required: true }
})

defineEmits(['open-event', 'archive-all'])

// Helper functions to format the raw database strings
const formatEventType = (type) => type ? type.replace(/_/g, ' ') : ''
const formatTime = (timestamp) => timestamp ? timestamp.split(' ')[1] : ''
</script>

<template>
    <div class="bg-slate-50 dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 rounded-lg overflow-hidden flex flex-col backdrop-blur-sm w-full">
        
        <div class="px-5 py-3 border-b border-slate-200 dark:border-zinc-800 flex justify-between items-center bg-slate-100/50 dark:bg-zinc-950/50 shrink-0">
            <h3 class="text-sm font-semibold text-slate-800 dark:text-zinc-200">
                {{ viewingArchive ? 'Archived Events' : 'Active Threat Queue' }}
            </h3>
            
            <button v-show="!viewingArchive && events.length > 0" @click="$emit('archive-all')"
                    class="px-2.5 py-1 rounded-md text-xs font-semibold text-slate-600 dark:text-zinc-400 bg-white dark:bg-zinc-800 hover:bg-slate-100 dark:hover:bg-zinc-700 transition-colors border border-slate-300 dark:border-zinc-700 shadow-sm">
                Archive All
            </button>
        </div>

        <div class="overflow-x-auto custom-scroll max-h-[450px]">
            <table class="w-full text-left border-collapse">
                <thead class="text-xs font-semibold text-slate-500 dark:text-zinc-400 border-b border-slate-200 dark:border-zinc-800 sticky top-0 bg-slate-50 dark:bg-zinc-900 z-10">
                    <tr>
                        <th class="px-5 py-3">Threat</th>
                        <th class="px-4 py-3">Event Trigger</th>
                        <th class="px-4 py-3">Source</th>
                        <th class="px-4 py-3">Target</th>
                        <th class="px-4 py-3 text-right min-w-[180px]">Node</th>
                        <th class="px-5 py-3 text-right">Time</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-slate-200 dark:divide-zinc-800/50">
                    
                    <tr v-if="events.length === 0">
                        <td colspan="6" class="px-5 py-8 text-center text-slate-500 dark:text-zinc-500 text-sm">
                            No events detected matching criteria.
                        </td>
                    </tr>
                    
                    <tr v-for="event in events" :key="event.id"
                        class="hover:bg-slate-100/50 dark:hover:bg-zinc-800/30 cursor-pointer border-l-[3px] border-transparent transition-colors"
                        :class="'bleed-' + event.severity"
                        @click="$emit('open-event', event)">
                        
                        <td class="px-5 py-3 flex items-center gap-3">
                            <div v-show="!event.is_read" class="w-1.5 h-1.5 rounded-full bg-rose-500 shrink-0"></div>
                            <span class="px-2 py-0.5 rounded border text-[11px] font-semibold uppercase tracking-wider bg-slate-50 dark:bg-transparent whitespace-nowrap" 
                                  :class="'severity-' + event.severity">{{ event.severity }}</span>
                        </td>
                        <td class="px-4 py-3 text-sm text-slate-900 dark:text-zinc-100 capitalize">{{ formatEventType(event.event_type) }}</td>
                        <td class="px-4 py-3 text-sm text-slate-600 dark:text-zinc-400 mono">{{ event.source }}</td>
                        <td class="px-4 py-3 text-sm text-slate-600 dark:text-zinc-400 mono">{{ event.target }}</td>
                        <td class="px-4 py-3 text-sm text-right text-slate-500 dark:text-zinc-500 mono min-w-[180px] max-w-[200px] truncate" :title="event.sensor_id">{{ event.sensor_id }}</td>
                        <td class="px-5 py-3 text-sm text-right text-slate-500 dark:text-zinc-500 mono whitespace-nowrap">{{ formatTime(event.timestamp) }}</td>
                    </tr>

                </tbody>
            </table>
        </div>
    </div>
</template>