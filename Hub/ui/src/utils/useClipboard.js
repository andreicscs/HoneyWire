import { ref } from 'vue'

export function useClipboard() {
    const copiedStates = ref({})

    const handleCopy = async (id, text) => {
        if (!text) return
        try {
            if (navigator.clipboard && window.isSecureContext) {
                await navigator.clipboard.writeText(text)
            } else {
                // Fallback for non-secure contexts (e.g., HTTP via IP address)
                const textArea = document.createElement('textarea')
                textArea.value = text
                textArea.style.position = 'fixed' // Prevent page scroll
                textArea.style.opacity = '0'
                document.body.appendChild(textArea)
                textArea.focus()
                textArea.select()
                const successful = document.execCommand('copy')
                document.body.removeChild(textArea)
                if (!successful) throw new Error('Fallback copy failed')
            }
            copiedStates.value = { ...copiedStates.value, [id]: true }
            setTimeout(() => { 
                copiedStates.value = { ...copiedStates.value, [id]: false }
            }, 2000)
        } catch (err) {
            console.error('Failed to copy text', err)
            alert('Your browser blocked clipboard access due to an insecure HTTP context. Please copy manually.')
        }
    }

    return { copiedStates, handleCopy }
}