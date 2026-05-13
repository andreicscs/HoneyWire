// src/composables/useDropdown.js
import { ref } from 'vue'

// This exists exactly ONCE in the entire application's memory
export const globalActiveMenuId = ref(null)

let listenersAttached = false

export const initGlobalDropdownListeners = () => {
    if (listenersAttached || typeof window === 'undefined') return
    
    window.addEventListener('click', (e) => {
        if (!e.target.closest('.meatball-trigger') && !e.target.closest('.meatball-dropdown')) {
            globalActiveMenuId.value = null
        }
    }, true)
    
    window.addEventListener('scroll', () => {
        globalActiveMenuId.value = null
    }, true)
    
    listenersAttached = true
}