<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
    label: string,
    sortKey: string,
    sortCol: string,
    sortDesc: boolean,
    align?: 'left' | 'right',
    padClass?: string,
    ariaLabel?: string,
}>(), {
    align: 'left',
    padClass: 'px-4 py-3',
    ariaLabel: '',
})

const emit = defineEmits<{ (e: 'toggle', col: string): void }>()

const isActive = computed(() => props.sortKey === props.sortCol)
// Mirrors the table's arrow convention: timestamp/severity sort descending by
// default, every other column ascending; inactive columns show a muted arrow.
const arrowDown = computed(() => {
    if (!isActive.value) return true
    return ['timestamp', 'severity'].includes(props.sortKey) ? props.sortDesc : !props.sortDesc
})
</script>

<template>
    <th @click="emit('toggle', sortKey)"
        class="cursor-pointer hover:text-text-h transition-colors group"
        :class="[padClass, align === 'right' ? 'text-right' : '']"
        role="button"
        tabindex="0"
        :aria-label="ariaLabel || `Sort by ${label.toLowerCase()}`">
        <div class="flex items-center gap-1.5" :class="align === 'right' ? 'justify-end' : ''">{{ label }}
            <svg class="w-3 h-3 transition-transform duration-normal"
                 :class="[isActive ? 'opacity-100 text-highlight-border' : 'opacity-0 group-hover:opacity-50 text-text-m', arrowDown ? 'rotate-180' : '']"
                 viewBox="0 0 384 512" fill="currentColor"><path d="M214.6 41.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 141.2V448c0 17.7 14.3 32 32 32s32-14.3 32-32V141.2L329.4 246.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z"></path></svg>
        </div>
    </th>
</template>
