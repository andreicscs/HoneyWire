import { ref, computed } from 'vue'

// --- GLOBAL STATE ---
// Defined outside the function so it acts as a global singleton store
const events = ref([])
const fleet = ref([])
const uptimeData = ref([])
const isArmed = ref(true)
const version = ref('1.0.0')
const viewingArchive = ref(false)
const selectedSensor = ref(null)
const activeTimeframe = ref('30D')
const activeEvent = ref(null)

export function useSentinel() {
    
    // --- COMPUTED PROPERTIES ---
    const unreadCount = computed(() => events.value.filter(e => !e.is_read).length)
    
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
            const [ev, sn, st, ver, uptimeResult] = await Promise.all([
                fetch(`/api/v1/events?archived=${viewingArchive.value}`).then(r => r.json()).catch(() => []),
                fetch('/api/v1/sensors').then(r => r.json()).catch(() => []),
                fetch('/api/v1/system/state').then(r => r.json()).catch(() => ({is_armed: isArmed.value})),
                fetch('/api/v1/version').then(r => r.json()).catch(() => ({version: version.value})),
                fetch(`/api/v1/uptime?timeframe=${activeTimeframe.value}`).then(r => r.json()).catch(() => [])
            ])
            
            if (ev.length || sn.length) {
                events.value = ev
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

    const toggleSilence = async (sensorId) => {
        if (!sensorId) return
        const sensor = fleet.value.find(s => s.sensor_id === sensorId)
        if (!sensor) return

        const nextStateBool = !sensor.is_silenced

        // Optimistic UI Update + 5 Second Lock (matches your monolith logic)
        fleet.value = fleet.value.map(s => 
            s.sensor_id === sensorId ? { ...s, is_silenced: nextStateBool, _lockedUntil: Date.now() + 5000 } : s
        )

        try {
            const response = await fetch(`/api/v1/sensors/${sensorId}/silence`, {
                method: 'PATCH',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({ is_silenced: nextStateBool })
            })
            if (!response.ok) throw new Error('Server rejected update')
        } catch(err) { 
            console.error("Silence failed:", err)
            fleet.value = fleet.value.map(s => s.sensor_id === sensorId ? { ...s, _lockedUntil: 0 } : s)
        }
    }

    return {
        // State
        events, fleet, uptimeData, isArmed, version, viewingArchive, selectedSensor, activeTimeframe,
        // Computed
        unreadCount, filteredEvents, overallUptime,
        // Actions
        update, startPolling, toggleArmed, markAllRead, archiveAll, archiveEvent, toggleSilence,
        // UI State
        activeEvent, isActiveSensorSilenced
    }
}