import { ref, computed, watch } from 'vue'

// --- GLOBAL STATE ---
// Defined outside the function so it acts as a global singleton store
const events = ref([])
const fleet = ref([])
const uptimeData = ref([])
const isArmed = ref(true)
const version = ref('1.0.0')
const viewingArchive = ref(false)
const selectedSensor = ref(null)
const activeTimeframe = ref('24H')
const activeEvent = ref(null)
const unreadCount = ref(0)

export function useSentinel() {
    
    // --- COMPUTED PROPERTIES ---    
    const filteredEvents = computed(() => {
        return selectedSensor.value 
            ? events.value.filter(e => e.sensor_id === selectedSensor.value) 
            : events.value
    })

    const overallUptime = computed(() => {
        if (!uptimeData.value || uptimeData.value.length === 0) return '0.0%'
        let validBlocks = 0
        let upBlocks = 0
        
        uptimeData.value.forEach(sensor => {
            sensor.blocks.forEach(block => {
                if (block.status !== 'nodata') {
                    validBlocks++
                    if (block.status === 'up') upBlocks += 1
                    else if (block.status === 'degraded') upBlocks += 0.8
                }
            })
        })
        
        if (validBlocks === 0) return '100.0%'
        return ((upBlocks / validBlocks) * 100).toFixed(1) + '%'
    })

    const isActiveSensorSilenced = computed(() => {
        if (!activeEvent.value) return false
        return fleet.value.find(s => s.sensor_id === activeEvent.value.sensor_id)?.is_silenced || false
    })

    // --- API POLLING LOOP ---
    const update = async () => {
        try {
            // ALWAYS fetch active events to keep the unread badge accurate, 
            // then conditionally fetch archived events if the user is in that view
            const [activeEv, currentViewEv, sn, st, ver, uptimeResult] = await Promise.all([
                fetch(`/api/v1/events?archived=false`).then(r => r.json()).catch(() => []),
                viewingArchive.value ? fetch(`/api/v1/events?archived=true`).then(r => r.json()).catch(() => null) : Promise.resolve(null),
                fetch('/api/v1/sensors').then(r => r.json()).catch(() => []),
                fetch('/api/v1/system/state').then(r => r.json()).catch(() => ({is_armed: isArmed.value})),
                fetch('/api/v1/version').then(r => r.json()).catch(() => ({version: version.value})),
                fetch(`/api/v1/uptime?timeframe=${activeTimeframe.value}`).then(r => r.json()).catch(() => [])
            ])
            
            // 1. Maintain accurate unread count from active events
            if (activeEv) {
                unreadCount.value = activeEv.filter(e => !e.is_read).length
            }

            // 2. Assign the table data based on the current view
            const displayEvents = viewingArchive.value && currentViewEv ? currentViewEv : activeEv
            if (displayEvents) {
                events.value = displayEvents
            }

            if (sn && sn.length) {
                fleet.value = sn.map(newSensor => {
                    const normalizedSensor = { ...newSensor, is_silenced: !!newSensor.is_silenced }
                    const current = fleet.value.find(f => f.sensor_id === normalizedSensor.sensor_id)
                    
                    if (current && current._lockedUntil && Date.now() < current._lockedUntil) {
                        return { ...normalizedSensor, is_silenced: current.is_silenced, _lockedUntil: current._lockedUntil }
                    }
                    return normalizedSensor
                })
            }

            if (uptimeResult && uptimeResult.length) uptimeData.value = uptimeResult
            isArmed.value = st.is_armed !== undefined ? st.is_armed : isArmed.value
            version.value = ver.version || version.value

        } catch(e) { 
            console.error('Polling failed', e) 
        }
    }

    // --- WATCHERS ---
    // Instantly fetch new data when the server-side filters change
    watch([activeTimeframe, viewingArchive], () => {
        update()
    })

    // --- ACTIONS ---
    const startPolling = () => {
        update()
        setInterval(update, 5000)
    }

    const toggleArmed = async () => {
        const next = !isArmed.value
        try {
            await fetch('/api/v1/system/state', {
                method: 'PATCH', 
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({is_armed: next})
            })
            isArmed.value = next
        } catch(err) {}
    }

    const markAllRead = async () => {
        try {
            await fetch('/api/v1/events/read', {method: 'PATCH'})
            events.value.forEach(e => e.is_read = 1) 
            unreadCount.value = 0 // Optimistic UI update
        } catch(err) {}
    }

    const archiveAll = async () => {
        if (confirm("Archive all currently active events?")) {
            try {
                await fetch('/api/v1/events/archive-all', {method: 'PATCH'})
                update() // Immediately refresh to clear the table
            } catch(err) { console.error(err) }
        }
    }

    const archiveEvent = async (id) => {
        try {
            await fetch(`/api/v1/events/${id}/archive`, {method: 'PATCH'})
            events.value = events.value.filter(e => e.id !== id)
            activeEvent.value = null // Close the modal
        } catch(err) { console.error(err) }
    }

    const forgetSensor = async (sensorId) => {
        // 1. Optimistic Update: Immediately remove it from the local UI state
        fleet.value = fleet.value.filter(s => s.sensor_id !== sensorId)
        
        // If the user had this sensor selected, reset their view to 'All Traffic'
        if (selectedSensor.value === sensorId) {
            selectedSensor.value = null
        }

        try {
            // 2. Perform the actual network request in the background
            await fetch(`/api/v1/sensors/${sensorId}`, { method: 'DELETE' })
            
            update() 
        } catch (err) {
            console.error("Failed to delete sensor, state may be out of sync", err)
            // If it fails, you might want to call update() to fetch the true state back
        }
    }

    const toggleSilence = async (sensorId) => {
        // 1. Optimistic Update: Immediately flip the boolean in the local UI state
        const sensor = fleet.value.find(s => s.sensor_id === sensorId)
        if (sensor) {
            sensor.is_silenced = !sensor.is_silenced
            
            try {
                // 2. Perform the background network request
                await fetch(`/api/v1/sensors/${sensorId}/silence`, {
                    method: 'PATCH',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ is_silenced: sensor.is_silenced })
                })
            } catch (err) {
                console.error("Failed to silence sensor", err)
                // Revert optimistic update on failure
                sensor.is_silenced = !sensor.is_silenced
            }
        }
    }

    const markEventRead = async (eventId) => {
        // 1. Optimistic Update: Instantly recalculate the global badge count
        // since EventTable.vue just flipped this event's is_read property to true.
        unreadCount.value = events.value.filter(e => !e.is_read).length

        try {
            // 2. Fire the background network request silently
            await fetch(`/api/v1/events/${eventId}/read`, { method: 'PATCH' })
        } catch (err) {
            console.error("Failed to mark event as read", err)
            
            // Optional: Revert state if the network request fails
            const ev = events.value.find(e => e.id === eventId)
            if (ev) ev.is_read = false
            unreadCount.value = events.value.filter(e => !e.is_read).length
        }
    }
    
    return {
        // State
        events, fleet, uptimeData, isArmed, version, viewingArchive, selectedSensor, activeTimeframe,
        // Computed
        unreadCount, filteredEvents, overallUptime,
        // Actions
        update, startPolling, toggleArmed, markAllRead, archiveAll, archiveEvent, toggleSilence, forgetSensor, markEventRead,
        // UI State
        activeEvent, isActiveSensorSilenced
    }
}