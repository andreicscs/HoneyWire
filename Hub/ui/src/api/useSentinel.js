import { ref, computed, watch } from 'vue'

// --- GLOBAL STATE ---
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

    // --- FETCHERS ---
    const fetchEvents = async () => {
        try {
            if (viewingArchive.value) {
                const archiveEv = await fetch(`/api/v1/events?archived=true`).then(r => r.json()).catch(() => [])
                events.value = archiveEv 
                
                const activeEv = await fetch(`/api/v1/events?archived=false`).then(r => r.json()).catch(() => [])
                if (activeEv) unreadCount.value = activeEv.filter(e => !e.is_read).length
            } else {
                const activeEv = await fetch(`/api/v1/events?archived=false`).then(r => r.json()).catch(() => [])
                events.value = activeEv 
                if (activeEv) unreadCount.value = activeEv.filter(e => !e.is_read).length
            }
        } catch(e) {
            console.error('Failed to fetch events', e)
        }
    }

    const fetchUptime = async () => {
        try {
            const data = await fetch(`/api/v1/uptime?timeframe=${activeTimeframe.value}`).then(r => r.json())
            uptimeData.value = data || []
        } catch (e) {
            console.error('Failed to fetch uptime', e)
        }
    }

    const fetchFleet = async () => {
        try {
            const [sn, st, ver] = await Promise.all([
                fetch('/api/v1/sensors').then(r => r.json()).catch(() => []),
                fetch('/api/v1/system/state').then(r => r.json()).catch(() => ({is_armed: isArmed.value})),
                fetch('/api/v1/version').then(r => r.json()).catch(() => ({version: version.value}))
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

            isArmed.value = st.is_armed !== undefined ? st.is_armed : isArmed.value
            version.value = ver.version || version.value

        } catch(e) {
            console.error('Failed to fetch fleet data', e)
        }
    }

    // --- WEBSOCKET ENGINE ---
    let ws = null;
    let pollInterval = null;
    let isDestroyed = false; // Prevents memory leaks if unmounted

    const connectWS = () => {
        if (isDestroyed) return;

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/ws`);

        ws.onopen = () => console.log('🟢 WebSockets Connected. Real-time active.');

        ws.onmessage = (message) => {
            try {
                const data = JSON.parse(message.data);
                
                if (data.type === 'NEW_EVENT') {
                    unreadCount.value++; 

                    if (!viewingArchive.value) {
                        events.value.unshift(data.payload);
                        if (events.value.length > 1000) events.value.pop();
                    }
                }
            } catch (e) {
                console.error("WS Parse error", e);
            }
        };

        ws.onclose = () => {
            if (!isDestroyed) {
                console.warn('🔴 WebSockets Disconnected. Reconnecting in 3s...');
                setTimeout(connectWS, 3000); 
            }
        };
    }

    // --- ACTIONS ---
    const startPolling = async () => {
        isDestroyed = false;
        
        await Promise.all([fetchEvents(), fetchFleet(), fetchUptime()])
        connectWS()

        pollInterval = setInterval(() => {
            fetchFleet()
            fetchUptime()
        }, 30000)
    }

    const stopPolling = () => {
        isDestroyed = true;
        if (pollInterval) clearInterval(pollInterval);
        if (ws) ws.close();
    }

    // --- WATCHERS ---
    watch(viewingArchive, () => fetchEvents())
    watch(activeTimeframe, () => fetchUptime())

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
                fetchEvents() 
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
        if (selectedSensor.value === sensorId) selectedSensor.value = null

        try {
            await fetch(`/api/v1/sensors/${sensorId}`, { method: 'DELETE' })
            fetchFleet() 
        } catch (err) {}
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
        const ev = events.value.find(e => e.id === eventId)
        if (!ev || ev.is_read) return
        
        // Optimistic update
        ev.is_read = true
        unreadCount.value = Math.max(0, unreadCount.value - 1)

        try {
            await fetch(`/api/v1/events/${eventId}/read`, { method: 'PATCH' })
        } catch (err) {
            // Rollback on failure
            ev.is_read = false
            unreadCount.value = events.value.filter(e => !e.is_read).length
        }
    }
    
    return {
        events, fleet, uptimeData, isArmed, version, viewingArchive, selectedSensor, activeTimeframe,
        unreadCount, filteredEvents, overallUptime,
        startPolling, stopPolling, toggleArmed, markAllRead, archiveAll, archiveEvent, toggleSilence, forgetSensor, markEventRead,
        activeEvent, isActiveSensorSilenced
    }
}