import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useFleetStore } from './fleet'
import { useAppStore } from './app'

/**
 * Events Store (Telemetry State)
 * 
 * Manages event data, filtering by fleet selection, and real-time updates.
 * 
 * CRITICAL: filteredEvents and handleWsEvent use composite keys:
 * node_id AND sensor_id must be checked together.
 */

let severityAbortController = null;
let velocityAbortController = null;

export const useEventsStore = defineStore('events', () => {
  // --- STATE ---
  const events = ref([])
  const unreadCount = ref(0)
  const activeEvent = ref(null)
  const isFetching = ref(false)
  const severityProjection = ref(null)

  const threatVelocityProjection = ref(null)
  const isFetchingThreatVelocityProjection = ref(false)
  const lastVelocityInvalidation = ref(null)

  // --- GETTERS ---
  /**
   * Return events filtered by the fleet store's selectedNode and selectedSensor
   * Composite Key filtering: if both are selected, filter by both.
   */
  const filteredEvents = computed(() => {
    const fleetStore = useFleetStore()
    const appStore = useAppStore()
    const { selectedNode, selectedSensor } = fleetStore
    const isArchiveView = appStore.viewingArchive

    // 1. First, filter by Archive State
    // (Assuming the backend returns a mix, or 'is_archived' property exists)
    // If your backend only returns one type based on the fetch URL, skip this local filter 
    // and let the URL param handle it. But doing it locally ensures safety.
    let currentEvents = events.value.filter(e => 
        isArchiveView ? e.is_archived === true || e.is_archived === 1 : e.is_archived === false || e.is_archived === 0 || e.is_archived === undefined
    )

    // 2. No node/sensor filter: return events
    if (!selectedNode && !selectedSensor) {
      return currentEvents
    }

    // 3. Node selected but no sensor: filter by node only
    if (selectedNode && !selectedSensor) {
      return currentEvents.filter(e => e.node_id === selectedNode)
    }

    // 4. Both selected
    if (selectedNode && selectedSensor) {
      return currentEvents.filter(
        e => e.node_id === selectedNode && e.sensor_id === selectedSensor
      )
    }

    return currentEvents
  })

  // --- PRIVATE HELPERS ---
  /**
   * Refresh the unread count from the server
   * @private
   */
  const refreshUnreadCount = async () => {
    const res = await fetch('/api/v1/events/unread')
      .then(r => r.json())
      .catch(() => ({ count: 0 }))
    unreadCount.value = res.count
  }

  // --- ACTIONS ---

  /**
   * Fetch events from the backend with optional filters
   * @param {boolean} isArchived - Include archived events
   * @param {string} nodeId - Optional node filter
   * @param {string} sensorId - Optional sensor filter
   */
  const fetchEvents = async (isArchived = false, nodeId = null, sensorId = null) => {
    try {
      isFetching.value = true

      const url = new URL('/api/v1/events', window.location.origin)
      url.searchParams.append('archived', isArchived)

      if (nodeId) {
        url.searchParams.append('node_id', nodeId)
      }

      if (sensorId) {
        url.searchParams.append('sensor_id', sensorId)
      }

      const res = await fetch(url.toString()).then(r => r.json())
      events.value = res || []

      await refreshUnreadCount()
    } catch (e) {
      console.error('Failed to fetch events', e)
    } finally {
      isFetching.value = false
    }
  }

  const fetchSeverityProjection = async (timeframe = 'alltime', nodeId = null, sensorId = null) => {
    if (severityAbortController) {
      severityAbortController.abort();
    }
    severityAbortController = new AbortController();

    try {
      const appStore = useAppStore()
      const viewingArchive = appStore.viewingArchive
      const params = new URLSearchParams({ timeframe });
      if (nodeId) params.append('node', nodeId);
      if (sensorId) params.append('sensor', sensorId);
      params.append('viewingArchive', viewingArchive);

      const response = await fetch(`/api/v1/events/severity?${params.toString()}`, {
        signal: severityAbortController.signal
      });
      if (!response.ok) {
        throw new Error(`Server error: ${response.status}`)
      }
      const data = await response.json()
      severityProjection.value = data
    } catch (e) {
      if (e.name !== 'AbortError') console.error('Failed to fetch severity projection', e)
    }
  }

  const fetchThreatVelocityProjection = async (timeframe = '24H', nodeId = null, sensorId = null, viewingArchive = false) => {
    if (velocityAbortController) {
      velocityAbortController.abort();
    }
    velocityAbortController = new AbortController();

    try {
      isFetchingThreatVelocityProjection.value = true;
      const params = new URLSearchParams({ timeframe, archived: viewingArchive ? 'true' : 'false' });
      if (nodeId) params.append('node_id', nodeId);
      if (sensorId) params.append('sensor_id', sensorId);

      const response = await fetch(`/api/v1/events/velocity?${params.toString()}`, {
        signal: velocityAbortController.signal
      });
      
      if (!response.ok) {
        throw new Error(`Server error: ${response.status}`)
      }
      
      threatVelocityProjection.value = await response.json(); 
    } catch (e) {
      if (e.name !== 'AbortError') console.error('Velocity fetch failed:', e);
    } finally {
      isFetchingThreatVelocityProjection.value = false;
    }
  }

  const invalidateThreatVelocityProjection = () => {
    lastVelocityInvalidation.value = Date.now();
  }

  const markAllRead = async () => {
    try {
      const response = await fetch('/api/v1/events/read', { method: 'PATCH' })
      if (!response.ok) throw new Error(`Server error: ${response.status}`)

      events.value.forEach(e => (e.is_read = 1))
      unreadCount.value = 0
    } catch (err) {
      console.error('Failed to mark all events as read:', err)
      alert('Failed to mark events as read. Please try again.')
    }
  }

  /**
   * Mark a specific event as read
   * @param {string} eventId
   */
  const markEventRead = async (eventId) => {
    const ev = events.value.find(e => e.id === eventId)
    if (!ev || ev.is_read) return

    const wasRead = ev.is_read
    ev.is_read = true
    unreadCount.value = Math.max(0, unreadCount.value - 1)

    try {
      const response = await fetch(`/api/v1/events/${eventId}/read`, {
        method: 'PATCH',
      })
      if (!response.ok) throw new Error(`Server error: ${response.status}`)
    } catch (err) {
      ev.is_read = wasRead
      unreadCount.value = Math.max(0, unreadCount.value + 1)
      console.error('Failed to mark event as read:', err)
    }
  }

  /**
   * Archive a specific event
   * @param {string} eventId
   */
  const archiveEvent = async (eventId) => {
    const originalEvents = [...events.value]

    try {
      const response = await fetch(`/api/v1/events/${eventId}/archive`, {
        method: 'PATCH',
      })
      if (!response.ok) throw new Error(`Server error: ${response.status}`)

      events.value = events.value.filter(e => e.id !== eventId)
      activeEvent.value = null

      await refreshUnreadCount()
    } catch (err) {
      console.error('Failed to archive event:', err)
      events.value = originalEvents
      alert('Failed to archive event. Please try again.')
    }
  }

  /**
   * Archive all currently active events
   */
  const archiveAll = async () => {
    if (!confirm('Archive all currently active events?')) {
      return
    }

    try {
      const response = await fetch('/api/v1/events/archive-all', {
        method: 'PATCH',
      })
      if (!response.ok) throw new Error(`Server error: ${response.status}`)

      // Refetch to update the list
      const fleetStore = useFleetStore()
      await fetchEvents(false, fleetStore.selectedNode, fleetStore.selectedSensor)
    } catch (err) {
      console.error('Failed to archive all events:', err)
      alert('Failed to archive events. Please try again.')
    }
  }

  /**
   * Clear all events locally (no backend call)
   */
  const purgeEvents = () => {
    events.value = []
    unreadCount.value = 0
  }

  /**
   * Handle a new event from WebSocket
   * Prepends event and increments unread count only if it matches current filters
   * COMPOSITE KEY enforcement: node_id AND sensor_id must both match for sensor filter
   * @param {object} payload - Event payload from WebSocket
   */
  const handleWsEvent = (payload) => {
    const fleetStore = useFleetStore()
    const { selectedNode, selectedSensor } = fleetStore

    // Always increment unread count (event came from backend regardless)
    unreadCount.value++

    // Determine if the event should be added to the visible list
    // 1. If no filter is active, show everything
    const noFilter = !selectedNode && !selectedSensor

    // 2. If node is selected but no sensor, match node only
    const nodeOnlyMatch =
      selectedNode && !selectedSensor && payload.node_id === selectedNode

    // 3. COMPOSITE KEY: If specific sensor is selected, match BOTH node AND sensor
    const sensorMatch =
      selectedSensor &&
      selectedNode &&
      payload.node_id === selectedNode &&
      payload.sensor_id === selectedSensor

    // Add event to the front if it matches the current filter
    if (noFilter || nodeOnlyMatch || sensorMatch) {
      events.value.unshift(payload)
    }

    const appStore = useAppStore();
    const affectsCurrentView = 
      (!appStore.viewingArchive) &&
      (!selectedNode || selectedNode === payload.node_id) &&
      (!selectedSensor || selectedSensor === payload.sensor_id);

    if (affectsCurrentView) {
      fetchSeverityProjection('alltime', selectedNode, selectedSensor);
      invalidateThreatVelocityProjection();
    }
  }

  return {
    // State
    events,
    unreadCount,
    activeEvent,
    isFetching,
    severityProjection,
    threatVelocityProjection,
    isFetchingThreatVelocityProjection,
    lastVelocityInvalidation,
    filteredEvents,
    // Actions
    fetchEvents,
    fetchSeverityProjection,
    fetchThreatVelocityProjection,
    invalidateThreatVelocityProjection,
    markAllRead,
    markEventRead,
    archiveEvent,
    archiveAll,
    purgeEvents,
    handleWsEvent,
  }
})
