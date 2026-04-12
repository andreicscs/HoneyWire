import { ref, computed, watch, shallowRef } from 'vue'

// --- GLOBAL STATE ---
const events = shallowRef([])
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

    // --- DECOUPLED FETCHERS ---

    // 1. Ultra-fast Event Fetcher
    const fetchEvents = async () => {
        try {
            const [eventsData, unreadData] = await Promise.all([
                fetch(`/api/v1/events?archived=${viewingArchive.value}`).then(r => r.json()).catch(() => []),
                fetch(`/api/v1/events/unread`).then(r => r.json()).catch(() => ({ count: 0 }))
            ])
            
            unreadCount.value = unreadData.count || 0
            events.value = eventsData || []
            
        } catch(e) {
            console.error('Failed to fetch events', e)
        }
    }

    // 2. Heavier Fleet & Uptime Fetcher
    const fetchFleetAndUptime = async () => {
        try {
            const [sn, st, ver, uptimeResult] = await Promise.all([
                fetch('/api/v1/sensors').then(r => r.json()).catch(() => []),
                fetch('/api/v1/system/state').then(r => r.json()).catch(() => ({is_armed: isArmed.value})),
                fetch('/api/v1/version').then(r => r.json()).catch(() => ({version: version.value})),
                fetch(`/api/v1/uptime?timeframe=${activeTimeframe.value}`).then(r => r.json()).catch(() => [])
            ])

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
            console.error('Failed to fetch fleet data', e)
        }
    }

    // The main polling loop runs both
    const update = async () => {
        await Promise.all([fetchEvents(), fetchFleetAndUptime()])
    }

    // --- WATCHERS ---
    
    // Only fetch events when toggling the archive view (Lightning fast)
    watch(viewingArchive, () => {
        fetchEvents()
    })

    // Only fetch heavy uptime data when changing the timeframe
    watch(activeTimeframe, () => {
        fetchFleetAndUptime()
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
            unreadCount.value = 0
        } catch(err) {}
    }

    const archiveAll = async () => {
        if (confirm("Archive all currently active events?")) {
            try {
                await fetch('/api/v1/events/archive-all', {method: 'PATCH'})
                fetchEvents() // Only refresh the table!
            } catch(err) { console.error(err) }
        }
    }

    const archiveEvent = async (id) => {
        try {
            await fetch(`/api/v1/events/${id}/archive`, {method: 'PATCH'})
            events.value = events.value.filter(e => e.id !== id)
            activeEvent.value = null
        } catch(err) { console.error(err) }
    }

    const forgetSensor = async (sensorId) => {
        fleet.value = fleet.value.filter(s => s.sensor_id !== sensorId)
        if (selectedSensor.value === sensorId) {
            selectedSensor.value = null
        }

        try {
            await fetch(`/api/v1/sensors/${sensorId}`, { method: 'DELETE' })
            fetchFleetAndUptime() // Refresh fleet
        } catch (err) {
            console.error("Failed to delete sensor", err)
        }
    }

    const toggleSilence = async (sensorId) => {
        const sensor = fleet.value.find(s => s.sensor_id === sensorId)
        if (sensor) {
            sensor.is_silenced = !sensor.is_silenced
            try {
                await fetch(`/api/v1/sensors/${sensorId}/silence`, {
                    method: 'PATCH',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ is_silenced: sensor.is_silenced })
                })
            } catch (err) {
                sensor.is_silenced = !sensor.is_silenced
            }
        }
    }

    const markEventRead = async (eventId) => {
        unreadCount.value = events.value.filter(e => !e.is_read).length
        try {
            await fetch(`/api/v1/events/${eventId}/read`, { method: 'PATCH' })
        } catch (err) {
            const ev = events.value.find(e => e.id === eventId)
            if (ev) ev.is_read = false
            unreadCount.value = events.value.filter(e => !e.is_read).length
        }
    }
    
    return {
        events, fleet, uptimeData, isArmed, version, viewingArchive, selectedSensor, activeTimeframe,
        unreadCount, filteredEvents, overallUptime,
        update, startPolling, toggleArmed, markAllRead, archiveAll, archiveEvent, toggleSilence, forgetSensor, markEventRead,
        activeEvent, isActiveSensorSilenced
    }
}