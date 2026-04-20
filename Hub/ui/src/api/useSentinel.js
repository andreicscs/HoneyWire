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
const velocityTimeframe = ref('24H')
const activeEvent = ref(null)
const unreadCount = ref(0)

// --- PAGINATION STATE ---
const isFetching = ref(false)

export function useSentinel() {
    
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
    const refreshUnreadCount = async () => {
        const res = await fetch('/api/v1/events/unread').then(r => r.json()).catch(() => ({count: 0}))
        unreadCount.value = res.count
    }

    const fetchEvents = async () => {
        try {
            isFetching.value = true
            const url = new URL('/api/v1/events', window.location.origin)
            url.searchParams.append('archived', viewingArchive.value)
            
            if (selectedSensor.value) {
                url.searchParams.append('sensor_id', selectedSensor.value)
            }

            const res = await fetch(url.toString()).then(r => r.json())
            
            events.value = res || []
            await refreshUnreadCount()
        } catch(e) {
            console.error('Failed to fetch events', e)
        } finally {
            isFetching.value = false
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
                    return { ...newSensor, is_silenced: !!newSensor.is_silenced }
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
    let healthSyncInterval = null;
    let isDestroyed = false; 

const connectWS = () => {
        if (isDestroyed) return;
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/ws`);

        ws.onmessage = (message) => {
            try {
                const data = JSON.parse(message.data);
                
                if (data.type === 'NEW_EVENT') {
                    unreadCount.value++; 
                    if (!viewingArchive.value) {
                        if (!selectedSensor.value || selectedSensor.value === data.payload.sensor_id) {
                            events.value.unshift(data.payload);
                        }
                    }
                } else if (data.type === 'NEW_SENSOR') {
                    fetchFleet();
                    fetchUptime();
                } else if (data.type === 'DELETE_SENSOR') {
                    fleet.value = fleet.value.filter(s => s.sensor_id !== data.payload.sensor_id);
                    uptimeData.value = uptimeData.value.filter(s => s.id !== data.payload.sensor_id);
                    if (selectedSensor.value === data.payload.sensor_id) selectedSensor.value = null;
                } else if (data.type === 'SILENCE_SENSOR') {
                    const s = fleet.value.find(s => s.sensor_id === data.payload.sensor_id);
                    if (s) s.is_silenced = data.payload.is_silenced;
                }
            } catch (e) {
                console.error("WS Parse error", e);
            }
        };

        ws.onclose = () => {
            if (!isDestroyed) setTimeout(connectWS, 3000); 
        };
    }

    // --- ACTIONS ---
    const logout = async () => {
        try {
            await fetch('/logout', { method: 'POST' })
            // Hard redirect to clear frontend state and force re-auth
            window.location.href = '/'
        } catch(err) {
            console.error("Logout failed", err)
        }
    }

    const startRealtimeSync = async () => {
        isDestroyed = false;
        
        // Initial data load
        await Promise.all([fetchEvents(), fetchFleet(), fetchUptime()])
        
        // 1. Establish the WebSocket for instant Push Notifications (Events, Silence toggles, etc.)
        connectWS()
        
        // 2. Establish the Health Sync loop for fleet and uptime data.
        healthSyncInterval = setInterval(() => { 
            fetchFleet(); 
            fetchUptime() 
        }, 30000)
    }

    const stopRealtimeSync = () => {
        isDestroyed = true;
        if (healthSyncInterval) clearInterval(healthSyncInterval);
        if (ws) ws.close();
    }

    // --- WATCHERS ---
    watch([viewingArchive, selectedSensor], () => fetchEvents())
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
            await refreshUnreadCount()
        } catch(err) { console.error(err) }
    }

    const forgetSensor = async (sensorId) => {
        fleet.value = fleet.value.filter(s => s.sensor_id !== sensorId)
        uptimeData.value = uptimeData.value.filter(s => s.id !== sensorId)
        if (selectedSensor.value === sensorId) selectedSensor.value = null
        try { await fetch(`/api/v1/sensors/${sensorId}`, { method: 'DELETE' }) } catch (err) {}
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
        ev.is_read = true
        unreadCount.value = Math.max(0, unreadCount.value - 1)
        try {
            await fetch(`/api/v1/events/${eventId}/read`, { method: 'PATCH' })
        } catch (err) {
            ev.is_read = false
            await refreshUnreadCount()
        }
    }
    
    return {
        events, fleet, uptimeData, isArmed, version, viewingArchive, selectedSensor, activeTimeframe, velocityTimeframe,
        unreadCount, overallUptime, isFetching,
        logout, startRealtimeSync, stopRealtimeSync, toggleArmed, markAllRead, archiveAll, archiveEvent, toggleSilence, forgetSensor, markEventRead,
        activeEvent, isActiveSensorSilenced
    }
}