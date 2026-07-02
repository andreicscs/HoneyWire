// src/composables/useDropdown.js
import { ref } from 'vue'

// This exists exactly ONCE in the entire application's memory
export const globalActiveMenuId = ref<string | null>(null)

let listenersAttached: boolean = false

export const initGlobalDropdownListeners = (): void => {
    if (listenersAttached || typeof window === 'undefined') return
    
    window.addEventListener('click', (e: Event) => {
        const target = e.target as Element | null;
        if (target && !target.closest('.meatball-trigger') && !target.closest('.meatball-dropdown')) {
            globalActiveMenuId.value = null
        }
    }, true)
    
    window.addEventListener('scroll', () => {
        globalActiveMenuId.value = null
    }, true)
    
    listenersAttached = true
}