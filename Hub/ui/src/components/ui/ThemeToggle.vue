<script setup>
defineEmits(['toggle'])
</script>

<template>
    <button 
        @click="$emit('toggle')" 
        class="w-10 h-10 rounded-[var(--radius-md)] bg-secondary-main border border-secondary-border text-secondary-text hover:bg-secondary-hover hover:text-text-h transition-colors duration-normal flex items-center justify-center group overflow-hidden shadow-sm outline-none"
    >
        <svg class="w-4 h-4 transition-transform duration-normal ease-out group-hover:rotate-45 group-hover:scale-110 block dark:hidden" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="5"></circle><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"></path>
        </svg>
        <svg class="w-4 h-4 transition-transform duration-normal ease-out group-hover:-rotate-12 group-hover:scale-110 hidden dark:block" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"></path>
        </svg>
    </button>
</template>