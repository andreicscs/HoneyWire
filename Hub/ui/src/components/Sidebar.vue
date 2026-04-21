<script setup>
defineProps({
    isOpen: Boolean,
    currentView: String,
    version: String,
    viewingArchive: Boolean
})

defineEmits(['change-view', 'toggle-archive', 'clear-logs', 'toggle-sidebar'])
</script>

<template>
    <aside class="flex flex-col bg-slate-50 dark:bg-zinc-950 border-r border-slate-200 dark:border-zinc-800 z-10 transition-all duration-300 ease-in-out shrink-0"
           :class="isOpen ? 'w-[240px]' : 'w-[68px]'">
        
        <div class="h-14 flex items-center px-[22px] shrink-0 border-b border-slate-200 dark:border-zinc-800/50 mb-2">
            <button @click="$emit('toggle-sidebar')" 
                    type="button"
                    aria-label="Toggle sidebar"
                    class="text-slate-400 hover:text-slate-700 dark:text-zinc-500 dark:hover:text-zinc-200 transition-colors">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path></svg>
            </button>
        </div>

        <nav class="flex-1 space-y-2 overflow-y-auto custom-scroll overflow-x-hidden px-3">
            
            <div class="px-3 text-xs font-semibold text-slate-400 dark:text-zinc-500 transition-all duration-300 overflow-hidden whitespace-nowrap"
                 :class="isOpen ? 'max-h-6 opacity-100 mb-1' : 'max-h-0 opacity-0 mb-0'">Menu</div>
            
            <button @click="$emit('change-view', 'dashboard')" 
                    type="button"
                    class="w-full flex items-center px-3 py-2.5 rounded-md text-sm font-medium transition-colors"
                    :class="currentView === 'dashboard' ? 'bg-slate-200 dark:bg-zinc-800 text-slate-900 dark:text-zinc-100' : 'text-slate-600 dark:text-zinc-400 hover:bg-slate-200/50 dark:hover:bg-zinc-800/50'"
                    :title="!isOpen ? 'Dashboard' : ''">
                <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"></path></svg>
                <div class="overflow-hidden transition-all duration-300 ease-in-out whitespace-nowrap flex items-center"
                     :class="isOpen ? 'max-w-[150px] ml-3 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>Dashboard</span>
                </div>
            </button>
            
            <button @click="$emit('change-view', 'store')" 
                    type="button"
                    class="w-full flex items-center px-3 py-2.5 rounded-md text-sm font-medium transition-colors"
                    :class="currentView === 'store' ? 'bg-slate-200 dark:bg-zinc-800 text-slate-900 dark:text-zinc-100' : 'text-slate-600 dark:text-zinc-400 hover:bg-slate-200/50 dark:hover:bg-zinc-800/50'"
                    :title="!isOpen ? 'Sensor Store' : ''">
                <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path></svg>
                <div class="overflow-hidden transition-all duration-300 ease-in-out whitespace-nowrap flex items-center"
                     :class="isOpen ? 'max-w-[150px] ml-3 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>Sensor Store</span>
                </div>
            </button>

            <button @click="$emit('change-view', 'settings')" 
                    type="button"
                    class="w-full flex items-center px-3 py-2.5 rounded-md text-sm font-medium transition-colors"
                    :class="currentView === 'settings' ? 'bg-slate-200 dark:bg-zinc-800 text-slate-900 dark:text-zinc-100' : 'text-slate-600 dark:text-zinc-400 hover:bg-slate-200/50 dark:hover:bg-zinc-800/50'"
                    :title="!isOpen ? 'Settings' : ''">
                <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>
                <div class="overflow-hidden transition-all duration-300 ease-in-out whitespace-nowrap flex items-center"
                     :class="isOpen ? 'max-w-[150px] ml-3 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>Settings</span>
                </div>
            </button>
        </nav>

        <div class="p-3 border-t border-slate-200 dark:border-zinc-800 shrink-0 space-y-2">
            <button @click="$emit('toggle-archive')" 
                    type="button"
                    :aria-label="viewingArchive ? 'View active events' : 'View archived events'"
                    class="w-full flex items-center justify-center py-2 px-3 rounded-md text-xs font-bold transition-colors border"
                    :class="viewingArchive ? 'bg-amber-100 dark:bg-amber-500/20 text-amber-800 dark:text-amber-400 border-amber-300 dark:border-amber-500/30 shadow-sm' : 'text-slate-600 dark:text-zinc-400 bg-slate-100 dark:bg-zinc-800 hover:bg-slate-200 dark:hover:bg-zinc-700 border-slate-300 dark:border-zinc-700'">
                <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"></path></svg>
                <div class="overflow-hidden transition-all duration-300 ease-in-out whitespace-nowrap flex items-center"
                    :class="isOpen ? 'max-w-[120px] ml-2 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>{{ viewingArchive ? 'Active Events' : 'Event Archive' }}</span>
                </div>
            </button>

            <button @click="$emit('clear-logs')" 
                    type="button"
                    aria-label="Purge system logs"
                    class="w-full flex items-center justify-center py-2 px-3 rounded-md text-xs font-bold text-rose-800 bg-rose-100 hover:bg-rose-200 dark:text-rose-400 dark:bg-rose-500/20 dark:hover:bg-rose-500/30 border border-rose-300 dark:border-rose-500/30 transition-colors shadow-sm"
                    :title="!isOpen ? 'Purge Logs' : ''">
                <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                <div class="overflow-hidden transition-all duration-300 ease-in-out whitespace-nowrap flex items-center"
                    :class="isOpen ? 'max-w-[120px] ml-2 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>Purge System Logs</span>
                </div>
            </button>
        </div>
    </aside>
</template>