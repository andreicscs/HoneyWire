import { ref, computed, watch } from 'vue'

// --- GLOBAL STATE ---
const events = ref([])
const fleet = ref([])
const uptimeData = ref([])
const isArmed = ref(true)
const version = ref('1.0.0')
const viewingArchive = ref(false)
const selectedNode = ref(null)   
const selectedSensor = ref(null) 
const activeTimeframe = ref('24H')
const velocityTimeframe = ref('24H')
const activeEvent = ref(null)
const unreadCount = ref(0)

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
            } else if (selectedNode.value) {
                url.searchParams.append('node_id', selectedNode.value)
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
    let wsRetryCount = 0;
    const maxRetries = 10;
    let wsRetryDelay = 3000;

    const connectWS = () => {
        if (isDestroyed) return;
        
        if (wsRetryCount >= maxRetries) {
            console.error('WebSocket: Max retries reached, stopping reconnection attempts');
            return;
        }
        
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/ws`);

        ws.onopen = () => {
            wsRetryCount = 0;
            wsRetryDelay = 3000;
        };

        ws.onmessage = (message) => {
            try {
                const data = JSON.parse(message.data);
                
                if (data.type === 'NEW_EVENT') {
                    unreadCount.value++; 
                    if (!viewingArchive.value) {
                        const matchNoFilter = !selectedSensor.value && !selectedNode.value;
                        const matchSensorFilter = selectedSensor.value === data.payload.sensor_id;
                        const matchNodeFilter = !selectedSensor.value && selectedNode.value === data.payload.node_id;

                        if (matchNoFilter || matchSensorFilter || matchNodeFilter) {
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
            if (!isDestroyed) {
                wsRetryCount++;
                console.warn(`WebSocket disconnected. Retry ${wsRetryCount}/${maxRetries} in ${wsRetryDelay}ms`);
                setTimeout(connectWS, wsRetryDelay);
                wsRetryDelay = Math.min(wsRetryDelay * 2, 30000);
            }
        };
    }

    const logout = async () => {
        try {
            await fetch('/logout', { method: 'POST' })
            window.location.href = '/'
        } catch(err) {
            console.error("Logout failed", err)
        }
    }

    const startRealtimeSync = async () => {
        isDestroyed = false;
        await Promise.all([fetchEvents(), fetchFleet(), fetchUptime()])
        connectWS()
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

    watch([viewingArchive, selectedSensor, selectedNode], () => fetchEvents())
    watch(activeTimeframe, () => fetchUptime())

    const toggleArmed = async () => {
        const next = !isArmed.value
        const previous = isArmed.value
        try {
            const response = await fetch('/api/v1/system/state', {
                method: 'PATCH', 
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({is_armed: next})
            })
            if (!response.ok) throw new Error(`Server error: ${response.status}`)
            isArmed.value = next
        } catch(err) {
            console.error('Failed to toggle armed state:', err)
            isArmed.value = previous
            alert(`Failed to ${next ? 'arm' : 'disarm'} system. Please try again.`)
        }
    }

    const markAllRead = async () => {
        try {
            const response = await fetch('/api/v1/events/read', {method: 'PATCH'})
            if (!response.ok) throw new Error(`Server error: ${response.status}`)
            events.value.forEach(e => e.is_read = 1) 
            unreadCount.value = 0
        } catch(err) {
            console.error('Failed to mark all events as read:', err)
            alert('Failed to mark events as read. Please try again.')
        }
    }

    const archiveAll = async () => {
        if (confirm("Archive all currently active events?")) {
            try {
                const response = await fetch('/api/v1/events/archive-all', {method: 'PATCH'})
                if (!response.ok) throw new Error(`Server error: ${response.status}`)
                fetchEvents() 
            } catch(err) {
                console.error('Failed to archive all events:', err)
                alert('Failed to archive events. Please try again.')
            }
        }
    }

    const archiveEvent = async (id) => {
        const originalEvents = [...events.value]
        try {
            const response = await fetch(`/api/v1/events/${id}/archive`, {method: 'PATCH'})
            if (!response.ok) throw new Error(`Server error: ${response.status}`)
            events.value = events.value.filter(e => e.id !== id)
            activeEvent.value = null
            await refreshUnreadCount()
        } catch(err) {
            console.error('Failed to archive event:', err)
            events.value = originalEvents
            alert('Failed to archive event. Please try again.')
        }
    }

    const forgetSensor = async (sensorId) => {
        const originalFleet = [...fleet.value]
        const originalUptime = [...uptimeData.value]
        const originalSelected = selectedSensor.value
        
        fleet.value = fleet.value.filter(s => s.sensor_id !== sensorId)
        uptimeData.value = uptimeData.value.filter(s => s.id !== sensorId)
        if (selectedSensor.value === sensorId) selectedSensor.value = null
        
        try {
            const response = await fetch(`/api/v1/sensors/${sensorId}`, { method: 'DELETE' })
            if (!response.ok) throw new Error(`Server error: ${response.status}`)
        } catch (err) {
            console.error('Failed to forget sensor:', err)
            fleet.value = originalFleet
            uptimeData.value = originalUptime
            selectedSensor.value = originalSelected
            alert('Failed to remove sensor. Please try again.')
        }
    }

    // NEW: Forget entire Node
    const forgetNode = async (nodeId) => {
        if (!confirm(`Are you sure you want to delete Node "${nodeId}" and ALL of its underlying sensors?`)) return;
        
        const nodeSensors = fleet.value.filter(s => s.node_id === nodeId);
        for (const s of nodeSensors) {
            await forgetSensor(s.sensor_id);
        }
        
        if (selectedNode.value === nodeId) {
            selectedNode.value = null;
        }
    }

    const toggleSilence = async (sensorId) => {
        const sensor = fleet.value.find(s => s.sensor_id === sensorId)
        if (sensor) {
            const previousState = sensor.is_silenced
            sensor.is_silenced = !sensor.is_silenced
            try {
                const response = await fetch(`/api/v1/sensors/${sensorId}/silence`, {
                    method: 'PATCH',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ is_silenced: sensor.is_silenced })
                })
                if (!response.ok) throw new Error(`Server error: ${response.status}`)
            } catch (err) {
                sensor.is_silenced = previousState
                console.error('Failed to toggle sensor silence:', err)
                alert(`Failed to ${previousState ? 'unsilence' : 'silence'} sensor. Please try again.`)
            }
        }
    }

    // NEW: Silence entire Node
    const silenceNode = async (nodeId) => {
        const nodeSensors = fleet.value.filter(s => s.node_id === nodeId);
        if (!nodeSensors.length) return;
        
        // If all are silenced, target is Unsilence. Otherwise, Silence all.
        const allSilenced = nodeSensors.every(s => s.is_silenced);
        const targetState = !allSilenced;
        
        for (const s of nodeSensors) {
            if (s.is_silenced !== targetState) {
                await toggleSilence(s.sensor_id);
            }
        }
    }

    const markEventRead = async (eventId) => {
        const ev = events.value.find(e => e.id === eventId)
        if (!ev || ev.is_read) return
        const wasRead = ev.is_read
        ev.is_read = true
        unreadCount.value = Math.max(0, unreadCount.value - 1)
        try {
            const response = await fetch(`/api/v1/events/${eventId}/read`, { method: 'PATCH' })
            if (!response.ok) throw new Error(`Server error: ${response.status}`)
        } catch (err) {
            ev.is_read = wasRead
            unreadCount.value = Math.max(0, unreadCount.value + 1)
            console.error('Failed to mark event as read:', err)
        }
    }

    const purgeEvents = () => {
        events.value = []
        unreadCount.value = 0
    }
    
    return {
        events, fleet, uptimeData, isArmed, version, viewingArchive, selectedNode, selectedSensor, activeTimeframe, velocityTimeframe,
        unreadCount, overallUptime, isFetching,
        logout, startRealtimeSync, stopRealtimeSync, toggleArmed, markAllRead, archiveAll, archiveEvent, toggleSilence, forgetSensor, markEventRead,
        activeEvent, isActiveSensorSilenced, purgeEvents, silenceNode, forgetNode // <-- Exposed
    }
}