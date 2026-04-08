<script setup>
defineProps({
    currentView: String,
    isArmed: Boolean,
    unreadCount: Number
})

defineEmits(['toggle-sidebar', 'toggle-theme', 'toggle-armed', 'mark-all-read'])
</script>

<template>
    <header class="h-14 bg-slate-50/90 dark:bg-zinc-950/90 border-b border-slate-200 dark:border-zinc-800 flex items-center justify-between px-4 sm:px-6 shrink-0 backdrop-blur-sm">
        
        <div class="flex items-center gap-4">
            <button @click="$emit('toggle-sidebar')" class="text-slate-400 hover:text-slate-700 dark:text-zinc-500 dark:hover:text-zinc-200 transition-colors">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path></svg>
            </button>
            <h2 class="text-sm font-semibold text-slate-800 dark:text-white capitalize hidden sm:block">{{ currentView.replace('-', ' ') }}</h2>
        </div>
        
        <div class="flex items-center gap-3">
            
            <div class="hidden sm:flex items-center gap-2 pr-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse dark:shadow-[0_0_8px_rgba(16,185,129,0.8)]"></span>
                <span class="text-[11px] font-bold uppercase tracking-widest text-slate-500 dark:text-zinc-400">Live</span>
            </div>

            <button v-show="unreadCount > 0" @click="$emit('mark-all-read')"
                    class="hidden sm:flex items-center gap-2 px-2.5 py-1 rounded-md bg-rose-100 dark:bg-rose-900/30 text-rose-700 dark:text-rose-400 text-xs font-semibold mr-1 border border-rose-200 dark:border-rose-800/30">
                <span class="w-1.5 h-1.5 rounded-full bg-rose-500"></span>
                <span>{{ unreadCount }} Unread</span>
            </button>

            <button @click="$emit('toggle-armed')" 
                    class="px-3 py-1.5 rounded-md text-xs font-semibold transition-colors border"
                    :class="isArmed ? 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800/50' : 'bg-slate-200 dark:bg-zinc-800 text-slate-600 dark:text-zinc-400 border-slate-300 dark:border-zinc-700'">
                <span>{{ isArmed ? 'Armed' : 'Passive' }}</span>
            </button>

            <button @click="$emit('toggle-theme')" class="p-1.5 rounded-md bg-slate-200 dark:bg-zinc-800 border border-slate-300 dark:border-zinc-700 text-slate-600 dark:text-zinc-300 hover:bg-slate-300 dark:hover:bg-zinc-700 transition-colors">
                <svg class="w-4 h-4 hidden dark:block" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path></svg>
                <svg class="w-4 h-4 block dark:hidden" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path></svg>
            </button>
            
            <div class="w-px h-4 bg-slate-300 dark:bg-zinc-700 mx-1"></div>

            <a href="/logout" class="text-xs font-semibold text-slate-500 hover:text-slate-800 dark:text-zinc-400 dark:hover:text-white transition-colors">Exit</a>
        </div>
    </header>
</template>