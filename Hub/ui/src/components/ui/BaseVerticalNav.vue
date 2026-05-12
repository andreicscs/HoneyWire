<script setup>
defineProps({
    modelValue: { type: String, required: true },
    tabs: { type: Array, required: true } // { id, label, iconSvg, danger, divider }
})
defineEmits(['update:modelValue'])
</script>

<template>
    <nav class="w-full md:w-56 shrink-0 flex flex-col gap-2">
        <template v-for="(tab, idx) in tabs" :key="tab.id || idx">
            
            <div v-if="tab.divider" class="h-px bg-border-default/50 my-2 mx-4"></div>
            
            <button v-else @click="$emit('update:modelValue', tab.id)"
                class="w-full text-left px-4 py-2.5 rounded-[var(--radius-md)] text-base transition-all duration-fast flex items-center gap-3 border outline-none shadow-lg hover:shadow-md"
                :class="[
                    modelValue === tab.id 
                        ? (tab.danger ? 'bg-danger-hover text-danger-text !shadow-sm border-danger-border' : 'bg-secondary-selected text-text-h !shadow-sm border-secondary-border')
                        : (tab.danger ? 'bg-secondary-main text-text-m hover:bg-danger-bg hover:text-danger-text border-border-default/50' : 'bg-secondary-main text-secondary-text hover:bg-secondary-hover hover:text-text-h border-secondary-border/50')
                ]"
            >
                <span v-if="tab.iconSvg" class="w-5 h-5 shrink-0" v-html="tab.iconSvg"></span>
                {{ tab.label }}
            </button>

        </template>
    </nav>
</template>