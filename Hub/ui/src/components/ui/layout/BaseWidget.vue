<script setup>
// Base layout component for dashboard widgets
</script>

<template>
    <div class="bg-bg-surface border border-border-default rounded-lg flex flex-col shadow-sm h-full w-full overflow-hidden relative group">
        
        <!-- Main Content Area -->
        <div class="p-4 sm:p-5 pb-0 sm:pb-0 flex flex-col flex-1 min-h-0">
            <slot name="header"></slot>
            <slot></slot>
        </div>

        <!-- Footer Area -->
        <div v-if="$slots.footer" class="px-4 py-2.5 sm:px-5 sm:py-3 shrink-0 mt-auto">
            <slot name="footer"></slot>
        </div>

    </div>
</template>