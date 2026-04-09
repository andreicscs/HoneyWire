<script setup>
defineProps({
    currentView: String,
    isArmed: Boolean,
    unreadCount: Number
})

defineEmits(['toggle-theme', 'toggle-armed', 'mark-all-read'])
</script>

<template>
    <header class="h-14 bg-slate-50/90 dark:bg-zinc-950/90 border-b border-slate-200 dark:border-zinc-800 flex items-center justify-between px-4 sm:px-6 shrink-0 backdrop-blur-sm">
        
        <div class="flex items-center gap-4">
            <div class="flex items-center gap-2 pr-4 mr-2 border-r border-slate-200 dark:border-zinc-700">
                <svg class="w-10 h-10 text-slate-900 dark:text-white fill-current shrink-0" viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg">
                    <path d="M511.995 377.74q0-.166-.004-.333c-.189-68.109-26.791-132.112-74.972-180.292-48.352-48.352-112.638-74.98-181.019-74.98-68.38 0-132.667 26.628-181.02 74.981C26.628 245.468 0 309.754 0 378.135c0 5.632 4.566 10.199 10.199 10.199 1.864 0 3.606-.509 5.11-1.382l125.089-83.831 109.438 84.656a10.14 10.14 0 0 0 6.164 2.088c2.181 0 5.404-.696 7.261-2.133l109.361-84.611 121.315 81.509c2.254 1.919 5.719 3.958 8.883 3.958 5.632 0 9.179-4.822 9.179-10.454.001-.131-.004-.262-.004-.394M21.212 358.623c3.517-42.685 18.459-82.176 41.757-115.408l60.428 46.753zM131.57 270.5l-56.183-43.468a237.4 237.4 0 0 1 56.183-48.87zm20.398-103.713a233.8 233.8 0 0 1 83.541-23.352l-83.541 115.904zm93.833 192.092-75.958-58.768h75.958zm20.398.788v-59.556h76.977zm0-79.954v-57.116c0-5.632-4.567-10.199-10.199-10.199-5.633 0-10.199 4.567-10.199 10.199v57.116h-83.372l94.082-130.528 94.082 130.528zm94.853-20.375-83.471-115.806a233.8 233.8 0 0 1 83.471 23.762zm20.398-80.539a237.5 237.5 0 0 1 55.567 48.71L381.45 270.5zm8.173 111.169 59.764-46.239c22.96 32.935 37.728 71.98 41.335 114.166z"/><path d="M234.5 172.5 229 196l6 6.5c6 7.5 7 8.5 4.5 7l-8-4.5-7-3.5-3.5-18q.5-2.5-1.5-2-4.5.5-3 3l3 16.5.5 3.5 3 1.5 19 10.5-11 5-13 6.5-2 1-.5 12.5v12h2l2.5.5.5-11v-11l21-9.5-5.5 5.5-7.5 7.5-2 2.5 4 15 5 15.5q0 1 2 .5l2-1.5-4-14.5-3.5-14 7.5-8q11-12 6.5-5.5c-2 2.5-2.5 4.5-5 17v2l7 5.5 7.5 5.5 1.5-1 12.5-10c1-.5 1-1-.5-8-1.5-8.5-2-9-4-11l-1-2 8 7.5 7.5 7.5-1 3.5-6.5 26 2 .5q2 1 2-.5c2.5-6.5 8.5-29.5 8.5-30l-4.5-5q-9.5-10-9-10.5l10 5 10 4.5v21.5h1l2.5.5h1v-25l-11.5-5.5-13-6.5-1.5-1.5 10.5-5 11-6 .5-2.5 3.5-19.5-2.5-.5-2-.5-1.5 9-2 10q0 1.5-6.5 4L271 210l2.5-4 9-10.5L280 185l-5-17.5h-2.5q-2 .5-1.5 1.5l3.5 13 3 12.5-2.5 2.5-8 9.5v-2q1-1.5-1-3.5c-2.5-2.5-2.5-2.5-1-5l1-1.5-2.5-3.5q-2-4-3-4-2 1-1 4 .5 4.5-1.5 4.5h-3.5q-3 .5-2.5-1.5v-3.5q1-2-.5-3l-1-1-1.5 2-2 3.5q-2 2.5-.5 4 2.5 2.5-.5 5-2.5 1.5-1.5 4v2l-1-1-8.5-10.5-1-1 1-4.5 5.5-21.5-3.5-1.5zm25.5 30q2.5 2 1.5 3c-.5 3.5-3 8-4.5 9.5l-1.5 1-1.5-1-3-5c-2-5-2-6 0-7.5q4.5-4.5 9 0m2 26.5 3 13c0 .5-8.5 7.5-9.5 7.5-.5 0-9-7-9-8l3-13 3.5-4 3-2.5 2.5 3 3 4"/></svg>
                <span class="text-sm font-bold text-slate-800 dark:text-white leading-none tracking-wide">HoneyWire</span>
            </div>
            
            <div class="h-4 w-px bg-slate-300 dark:bg-zinc-700 mx-1 hidden sm:block"></div>
            <h2 class="text-sm font-semibold text-slate-500 dark:text-zinc-400 capitalize hidden sm:block">{{ currentView.replace('-', ' ') }}</h2>
        </div>
        
        <div class="flex items-center gap-3">
            <button v-show="unreadCount > 0" @click="$emit('mark-all-read')"
                    class="hidden sm:flex items-center gap-2 px-2.5 py-1 rounded-md bg-rose-100 dark:bg-rose-900/30 text-rose-700 dark:text-rose-400 text-xs font-semibold mr-1 border border-rose-200 dark:border-rose-800/30">
                <span class="w-1.5 h-1.5 rounded-full bg-rose-500 animate-pulse"></span>
                <span>{{ unreadCount }} Unread</span>
            </button>

            <button @click="$emit('toggle-armed')" 
                    class="px-3 py-1.5 rounded-md text-xs font-semibold transition-colors border"
                    :class="isArmed ? 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800/50' : 'bg-slate-200 dark:bg-zinc-800 text-slate-600 dark:text-zinc-400 border-slate-300 dark:border-zinc-700'">
                <span>{{ isArmed ? 'Armed' : 'Passive' }}</span>
            </button>

            <button @click="$emit('toggle-theme')" 
                    class="w-8 h-8 rounded-md bg-slate-200 dark:bg-zinc-800 border border-slate-300 dark:border-zinc-700 text-slate-600 dark:text-zinc-300 transition-colors flex items-center justify-center group overflow-hidden">
                <svg class="w-4 h-4 transition-transform duration-300 ease-out group-hover:rotate-45 group-hover:scale-110 block dark:hidden" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="5"></circle><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"></path>
                </svg>
                <svg class="w-4 h-4 transition-transform duration-300 ease-out group-hover:-rotate-12 group-hover:scale-110 hidden dark:block" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"></path>
                </svg>
            </button>
            
            <div class="w-px h-4 bg-slate-300 dark:bg-zinc-700 mx-1"></div>
            <a href="/logout" class="text-xs font-semibold text-slate-500 hover:text-slate-800 dark:text-zinc-400 dark:hover:text-white transition-colors">Exit</a>
        </div>
    </header>
</template>