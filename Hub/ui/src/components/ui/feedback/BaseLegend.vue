<script setup>
defineProps({
    items: {
        type: Array,
        required: true,
    }
})
</script>

<template>
    <div class="flex flex-wrap items-center justify-center gap-3 sm:gap-4 text-sm font-medium text-text-l shrink-0">
        <div v-for="(item, index) in items" :key="index" class="flex items-center gap-1.5 cursor-default">
            <span class="w-2 h-2 rounded-full shrink-0" :class="item.colorClass"></span>
            <span>{{ item.label }}</span>
        </div>
    </div>
</template>