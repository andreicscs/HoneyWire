<script setup>
defineProps({
    event: { type: Object, required: true },
    isSilenced: { type: Boolean, required: true },
    viewingArchive: { type: Boolean, required: true }
})

defineEmits(['close', 'toggle-silence', 'archive-event'])

const formatString = (str) => str ? str.replace(/_/g, ' ') : ''
const formatJson = (val) => typeof val === 'object' ? JSON.stringify(val, null, 2) : val
</script>

<template>
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6">
        <div class="absolute inset-0 bg-slate-900/40 dark:bg-black/60 backdrop-blur-sm" @click="$emit('close')"></div>
        
        <div class="relative bg-slate-50 dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 w-full max-w-2xl rounded-lg shadow-2xl flex flex-col max-h-[90vh]">
            
            <div class="p-5 sm:p-6 border-b border-slate-200 dark:border-zinc-800 flex justify-between items-start shrink-0">
                <div>
                    <h2 class="text-xl font-bold text-slate-900 dark:text-white capitalize">{{ formatString(event.event_type) }}</h2>
                    <p class="text-xs text-slate-500 dark:text-zinc-500 mono mt-1">Trace: {{ event.id }}</p>
                </div>
                <span class="px-2 py-0.5 rounded border text-xs font-semibold uppercase tracking-wider bg-slate-100 dark:bg-transparent" 
                      :class="'severity-' + event.severity">{{ event.severity }}</span>
            </div>

            <div class="p-5 sm:p-6 overflow-y-auto custom-scroll flex-1 space-y-5">
                
                <div class="grid grid-cols-3 gap-4">
                    <div class="bg-slate-100/50 dark:bg-zinc-950/50 p-3 rounded-md border border-slate-200 dark:border-zinc-800/50">
                        <div class="text-xs font-medium text-slate-500 dark:text-zinc-500 mb-1">Sensor Node</div>
                        <div class="text-sm font-semibold text-slate-900 dark:text-zinc-100 mono truncate" :title="event.sensor_id">{{ event.sensor_id }}</div>
                    </div>
                    <div class="bg-slate-100/50 dark:bg-zinc-950/50 p-3 rounded-md border border-slate-200 dark:border-zinc-800/50">
                        <div class="text-xs font-medium text-slate-500 dark:text-zinc-500 mb-1">Source</div>
                        <div class="text-sm font-semibold text-slate-900 dark:text-zinc-100 mono">{{ event.source }}</div>
                    </div>
                    <div class="bg-slate-100/50 dark:bg-zinc-950/50 p-3 rounded-md border border-slate-200 dark:border-zinc-800/50">
                        <div class="text-xs font-medium text-slate-500 dark:text-zinc-500 mb-1">Target</div>
                        <div class="text-sm font-semibold text-slate-900 dark:text-zinc-100 mono">{{ event.target }}</div>
                    </div>
                </div>

                <div class="space-y-4">
                    <div v-for="(val, key) in event.details" :key="key">
                        <div class="text-xs font-semibold text-slate-700 dark:text-zinc-300 capitalize mb-2">{{ formatString(key) }}</div>
                        
                        <div v-if="Array.isArray(val)" class="space-y-2">
                            <pre v-for="(item, index) in val.slice(0, 50)" :key="index"
                                 class="bg-slate-100 dark:bg-[#0c0c0e] border border-slate-200 dark:border-[#1f1f23] rounded p-2.5 text-sm text-emerald-700 dark:text-emerald-400 mono overflow-x-auto custom-scroll">{{ formatJson(item) }}</pre>
                            <div v-show="val.length > 50" class="text-xs text-slate-500 dark:text-zinc-500 font-medium mt-2">
                                + {{ val.length - 50 }} more packets truncated
                            </div>
                        </div>
                        
                        <div v-else class="text-sm text-slate-800 dark:text-zinc-200 mono break-all bg-slate-100 dark:bg-[#0c0c0e] border border-slate-200 dark:border-[#1f1f23] p-2.5 rounded whitespace-pre-wrap">{{ formatJson(val) }}</div>
                    </div>
                </div>
            </div>

            <div class="p-5 sm:p-6 border-t border-slate-200 dark:border-zinc-800 shrink-0 flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                    
                    <button @click="$emit('toggle-silence', event.sensor_id)" 
                            class="flex items-center justify-center gap-2 py-2 px-3 border transition-colors rounded-md text-sm font-medium w-[155px] shrink-0"
                            :class="isSilenced ? 'bg-slate-100 dark:bg-zinc-800 text-slate-700 dark:text-zinc-300 border-slate-300 dark:border-zinc-700 hover:bg-slate-200 dark:hover:bg-zinc-700' : 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400 border-blue-200 dark:border-blue-800/30 hover:bg-blue-100 dark:hover:bg-blue-900/30'">
                        
                        <svg v-show="!isSilenced" class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"></path></svg>
                        <svg v-show="isSilenced" class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.73 21a2 2 0 01-3.46 0m-3.9-3.9a2.032 2.032 0 01-2.37.5L4 17h12.59l3.12 3.12M3 3l18 18M18 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341c-.5.186-.967.447-1.385.772"></path></svg>
                        <span class="whitespace-nowrap">{{ isSilenced ? 'Unsilence Sensor' : 'Silence Sensor' }}</span>
                    </button>

                    <button v-show="!viewingArchive" @click="$emit('archive-event', event.id)" 
                            class="flex items-center gap-2 py-2 px-3 border transition-colors rounded-md text-sm font-medium bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800/30 hover:bg-amber-100 dark:hover:bg-amber-900/30">
                        <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"></path></svg>
                        Archive Event
                    </button>
                </div>
                
                <button @click="$emit('close')" class="py-2 px-4 bg-slate-800 hover:bg-slate-900 dark:bg-zinc-100 dark:hover:bg-white text-white dark:text-slate-900 text-sm font-medium rounded-md transition-colors">
                    Close
                </button>
            </div>
        </div>
    </div>
</template>