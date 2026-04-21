import { reactive, readonly } from 'vue'

const state = reactive({
    isLoaded: false,
    hubEndpoint: '',
    hubKey: '',
    autoArchiveDays: 0,
    autoPurgeDays: 0,
    webhookType: 'ntfy',
    webhookUrl: '',
    webhookEvents: [],
    siemAddress: '',
    siemProtocol: 'tcp'
})

export function useConfig() {
    const fetchConfig = async () => {
        try {
            const res = await fetch('/api/v1/config')
            if (res.ok) {
                const data = await res.json()
                state.hubEndpoint = data.hub_endpoint || window.location.origin
                state.hubKey = data.hub_key || ''
                state.autoArchiveDays = data.auto_archive_days || 0
                state.autoPurgeDays = data.auto_purge_days || 0
                state.webhookType = data.webhook_type || 'ntfy'
                state.webhookUrl = data.webhook_url || ''
                state.webhookEvents = data.webhook_events || []
                state.siemAddress = data.siem_address || ''
                state.siemProtocol = data.siem_protocol || 'tcp'
                state.isLoaded = true
            }
        } catch (error) {
            console.error("Failed to load config", error)
        }
    }

    const patchConfig = async (updates) => {
        try {
            const payload = {}
            if (updates.hubEndpoint !== undefined) payload.hub_endpoint = updates.hubEndpoint
            if (updates.hubKey !== undefined) payload.hub_key = updates.hubKey
            if (updates.autoArchiveDays !== undefined) payload.auto_archive_days = updates.autoArchiveDays
            if (updates.autoPurgeDays !== undefined) payload.auto_purge_days = updates.autoPurgeDays
            if (updates.webhookType !== undefined) payload.webhook_type = updates.webhookType
            if (updates.webhookUrl !== undefined) payload.webhook_url = updates.webhookUrl
            if (updates.webhookEvents !== undefined) payload.webhook_events = updates.webhookEvents
            if (updates.siemAddress !== undefined) payload.siem_address = updates.siemAddress
            if (updates.siemProtocol !== undefined) payload.siem_protocol = updates.siemProtocol

            const res = await fetch('/api/v1/config', {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            })

            if (res.ok) {
                Object.assign(state, updates)
                return true
            } else {
                const errText = await res.text()
                console.error(`Config patch failed: ${res.status} - ${errText}`)
                return false
            }
        } catch (error) {
            console.error('Config patch network error:', error)
            return false
        }
    }

    return {
        config: readonly(state), 
        fetchConfig,
        patchConfig
    }
}