<script>
import { ref } from 'vue'

// --- SHARED GLOBAL STATE ---
const activeMenuId = ref(null)
let listenersAttached = false

const handleGlobalClick = (e) => {
    if (!e.target.closest('.meatball-trigger') && !e.target.closest('.meatball-dropdown')) {
        activeMenuId.value = null
    }
}
const handleGlobalScroll = () => {
    activeMenuId.value = null
}

// Tells Vue not to guess where classes go, preventing the warning
export default {
    inheritAttrs: false
}
</script>

<script setup>
import { computed, onMounted, ref as vueRef } from 'vue'

const props = defineProps({
    id: { type: String, default: () => Math.random().toString(36).substr(2, 9) },
    inverted: { type: Boolean, default: false }
})

const triggerRef = vueRef(null)
const menuPos = vueRef({ top: '0px', left: '0px' })

const isOpen = computed(() => activeMenuId.value === props.id)

const toggle = () => {
    if (isOpen.value) {
        activeMenuId.value = null
    } else {
        const rect = triggerRef.value.getBoundingClientRect()
        const dropdownWidth = 144 // Tailwind's w-36 is 9rem = 144px
        const windowWidth = window.innerWidth

        // Check if expanding to the right would go off-screen
        const fitsRight = rect.left + dropdownWidth <= windowWidth

        let calculatedLeft = rect.left
        if (!fitsRight) {
            // Align the right edge of the dropdown with the right edge of the trigger
            calculatedLeft = rect.right - dropdownWidth
            
            // Safety check: if it's too wide for the screen altogether, pin it to the left edge (0px)
            if (calculatedLeft < 0) {
                calculatedLeft = 0
            }
        }

        menuPos.value = { 
            top: rect.bottom + 6 + 'px', 
            left: calculatedLeft + 'px' 
        }
        activeMenuId.value = props.id
    }
}

onMounted(() => {
    if (!listenersAttached && typeof window !== 'undefined') {
        window.addEventListener('click', handleGlobalClick, true)
        window.addEventListener('scroll', handleGlobalScroll, true)
        listenersAttached = true
    }
})
</script>

<template>
    <div ref="triggerRef" 
         v-bind="$attrs"
         @click.stop="toggle"
         class="meatball-trigger w-5 h-5 rounded flex items-center justify-center transition-colors cursor-pointer shrink-0"
         :class="[
             inverted 
                ? (isOpen ? 'text-primary-text bg-primary-text/20' : 'text-primary-text/70 hover:text-primary-text hover:bg-primary-text/15')
                : (isOpen ? 'text-text-h bg-border-default' : 'text-text-l hover:text-text-h hover:bg-border-default/60')
         ]">
        <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/>
        </svg>
    </div>

    <Teleport to="body">
        <transition enter-active-class="transition ease-out duration-[var(--duration-fast)]" 
                    enter-from-class="transform opacity-0 scale-95" 
                    enter-to-class="transform opacity-100 scale-100" 
                    leave-active-class="transition ease-in duration-[var(--duration-fast)]" 
                    leave-from-class="transform opacity-100 scale-100" 
                    leave-to-class="transform opacity-0 scale-95">
            
            <div v-if="isOpen" 
                 :style="{ top: menuPos.top, left: menuPos.left }"
                 class="meatball-dropdown fixed w-36 rounded-md shadow-lg bg-bg-surface border border-border-default z-[100] py-1 overflow-hidden"
                 @click.stop="activeMenuId = null">
                <slot />
            </div>
            
        </transition>
    </Teleport>
</template>