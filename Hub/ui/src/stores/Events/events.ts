import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../../api/client'
// @ts-ignore - fleet.js is pending TypeScript migration TODO
import { useFleetStore } from '../Fleet/fleet'
import { useAppStore } from '../System/app'

// --- TYPES & INTERFACES ---

export interface EventPayload {
  id: string
  timestamp: string
  nodeId: string
  sensorId: string
  severity: 'info' | 'low' | 'medium' | 'high' | 'critical'
  eventTrigger: string
  source: string
  target: string
  isRead: boolean | number
  isArchived: boolean | number
  details?: Record<string, any>
}

export interface SeverityProjection {
  timeframe: string
  total: number
  critical: number
  high: number
  medium: number
  low: number
  info: number
}

export interface ThreatVelocityProjection {
  timeframe: string
  bucketSizeMs: number
  generatedAt: number
  bucketTimestamps: number[]
  labels: string[]
  exactTimes: string[]
  series: Record<string, number[]>
  recentEventCount: number
}

export interface EventsState {
  events: EventPayload[]
  unreadCount: number
  activeEvent: EventPayload | null
  isFetching: boolean
  severityProjection: SeverityProjection | null
  threatVelocityProjection: ThreatVelocityProjection | null
  isFetchingThreatVelocityProjection: boolean
  lastVelocityInvalidation: number | null
}

let severityAbortController: AbortController | null = null
let velocityAbortController: AbortController | null = null

export const useEventsStore = defineStore('events', () => {
  // ==========================================
  // SINGLE STATE TREE
  // ==========================================
  const state = ref<EventsState>({
    events: [],
    unreadCount: 0,
    activeEvent: null,
    isFetching: false,
    severityProjection: null,
    threatVelocityProjection: null,
    isFetchingThreatVelocityProjection: false,
    lastVelocityInvalidation: null
  })

  const fleetStore = useFleetStore()
  const appStore = useAppStore()

  // ==========================================
  // ENCAPSULATED GETTERS (Public API)
  // ==========================================
  const events = computed<EventPayload[]>(() => state.value.events)
  const unreadCount = computed<number>(() => state.value.unreadCount)
  const activeEvent = computed<EventPayload | null>(() => state.value.activeEvent)
  const isFetching = computed<boolean>(() => state.value.isFetching)
  const severityProjection = computed<SeverityProjection | null>(() => state.value.severityProjection)
  const threatVelocityProjection = computed<ThreatVelocityProjection | null>(() => state.value.threatVelocityProjection)
  const isFetchingThreatVelocityProjection = computed<boolean>(() => state.value.isFetchingThreatVelocityProjection)
  const lastVelocityInvalidation = computed<number | null>(() => state.value.lastVelocityInvalidation)

  /**
   * Return events filtered by the fleet store's active selection.
   * Note: We enforce composite key validation for sensors!
   */
  const filteredEvents = computed<EventPayload[]>(() => {
    const selectedNode = fleetStore.selectedNode
    const selectedSensor = fleetStore.selectedSensor
    const isArchiveView = !!appStore.viewingArchive

    let currentEvents = state.value.events.filter(e => {
        const isArchived = e.isArchived === true || e.isArchived === 1
        return isArchiveView ? isArchived : !isArchived
    })

    if (!selectedNode && !selectedSensor) {
      return currentEvents
    }

    if (selectedNode && !selectedSensor) {
      return currentEvents.filter(e => e.nodeId === selectedNode?.id)
    }

    if (selectedNode && selectedSensor) {
      return currentEvents.filter(
        e => e.nodeId === selectedNode?.id && e.sensorId === selectedSensor?.sensorId
      )
    }

    return currentEvents
  })

  // ==========================================
  // ACTIONS & GATEKEEPERS
  // ==========================================

  const refreshUnreadCount = async (): Promise<void> => {
    try {
      const res = await api.get('/api/v1/events/unread')
      const data = await res.json()
      state.value.unreadCount = data.count
    } catch {
      state.value.unreadCount = 0
    }
  }

  const fetchEvents = async (isArchived: boolean = false, nodeId: string | null = null, sensorId: string | null = null): Promise<void> => {
    try {
      state.value.isFetching = true
      const params = new URLSearchParams()
      params.append('archived', isArchived ? 'true' : 'false')
      if (nodeId) params.append('nodeId', nodeId)
      if (sensorId) params.append('sensorId', sensorId)

      const res = await api.get(`/api/v1/events?${params.toString()}`)
      state.value.events = await res.json() as EventPayload[]

      await refreshUnreadCount()
    } catch (e) {
      console.error('Failed to fetch events', e)
    } finally {
      state.value.isFetching = false
    }
  }

  const fetchSeverityProjection = async (timeframe: string = 'alltime', nodeId: string | null = null, sensorId: string | null = null): Promise<void> => {
    if (severityAbortController) severityAbortController.abort()
    severityAbortController = new AbortController()

    try {
      const params = new URLSearchParams({ timeframe, viewingArchive: appStore.viewingArchive ? 'true' : 'false' })
      if (nodeId) params.append('node', nodeId)
      if (sensorId) params.append('sensor', sensorId)

      const response = await api.get(`/api/v1/events/severity?${params.toString()}`, { signal: severityAbortController.signal })
      state.value.severityProjection = (await response.json()) as SeverityProjection
    } catch (e: any) {
      if (e.name !== 'AbortError') console.error('Failed to fetch severity projection', e)
    }
  }

  const fetchThreatVelocityProjection = async (timeframe: string = '24H', nodeId: string | null = null, sensorId: string | null = null, viewingArchive: boolean = false): Promise<void> => {
    if (velocityAbortController) velocityAbortController.abort()
    velocityAbortController = new AbortController()

    try {
      state.value.isFetchingThreatVelocityProjection = true
      const params = new URLSearchParams({ timeframe, archived: viewingArchive ? 'true' : 'false' })
      if (nodeId) params.append('nodeId', nodeId)
      if (sensorId) params.append('sensorId', sensorId)

      const response = await api.get(`/api/v1/events/velocity?${params.toString()}`, { signal: velocityAbortController.signal })
      state.value.threatVelocityProjection = (await response.json()) as ThreatVelocityProjection
    } catch (e: any) {
      if (e.name !== 'AbortError') console.error('Velocity fetch failed:', e)
    } finally {
      state.value.isFetchingThreatVelocityProjection = false
    }
  }

  const invalidateThreatVelocityProjection = (): void => {
    state.value.lastVelocityInvalidation = Date.now()
  }

  const markAllRead = async (): Promise<void> => {
    try {
      await api.patch('/api/v1/events/read')
      state.value.events.forEach(e => (e.isRead = true))
      state.value.unreadCount = 0
    } catch (err) {
      console.error('Failed to mark all events as read:', err)
      alert('Failed to mark events as read. Please try again.')
    }
  }

  const markEventRead = async (eventId: string): Promise<void> => {
    const ev = state.value.events.find(e => e.id === eventId)
    if (!ev || ev.isRead) return

    const wasRead = ev.isRead
    ev.isRead = true
    state.value.unreadCount = Math.max(0, state.value.unreadCount - 1)

    try {
      await api.patch(`/api/v1/events/${eventId}/read`)
    } catch (err) {
      ev.isRead = wasRead
      state.value.unreadCount = Math.max(0, state.value.unreadCount + 1)
      console.error('Failed to mark event as read:', err)
    }
  }

  const archiveEvent = async (eventId: string): Promise<void> => {
    const originalEvents = [...state.value.events]
    try {
      await api.patch(`/api/v1/events/${eventId}/archive`)
      state.value.events = state.value.events.filter(e => e.id !== eventId)
      state.value.activeEvent = null
      await refreshUnreadCount()
    } catch (err) {
      console.error('Failed to archive event:', err)
      state.value.events = originalEvents
      alert('Failed to archive event. Please try again.')
    }
  }

  const archiveAll = async (): Promise<void> => {
    if (!confirm('Archive all currently active events?')) return
    try {
      await api.patch('/api/v1/events/archive-all')
      await fetchEvents(false, fleetStore.selectedNode?.id, fleetStore.selectedSensor?.sensorId)
    } catch (err) {
      console.error('Failed to archive all events:', err)
      alert('Failed to archive events. Please try again.')
    }
  }

  const purgeEvents = (): void => {
    state.value.events = []
    state.value.unreadCount = 0
  }

  const handleWsEvent = (payload: EventPayload): void => {
    const selectedNode = fleetStore.selectedNode
    const selectedSensor = fleetStore.selectedSensor

    state.value.unreadCount++

    const noFilter = !selectedNode && !selectedSensor
    const nodeOnlyMatch = selectedNode && !selectedSensor && payload.nodeId === selectedNode?.id
    const sensorMatch = selectedSensor && selectedNode && payload.nodeId === selectedNode?.id && payload.sensorId === selectedSensor?.sensorId

    if (noFilter || nodeOnlyMatch || sensorMatch) {
      state.value.events.unshift(payload)
    }

    const affectsCurrentView = 
      (!appStore.viewingArchive) &&
      (!selectedNode || selectedNode?.id === payload.nodeId) &&
      (!selectedSensor || selectedSensor?.sensorId === payload.sensorId)

    if (affectsCurrentView) {
      fetchSeverityProjection('alltime', selectedNode?.id, selectedSensor?.sensorId)
      invalidateThreatVelocityProjection()
    }
  }

  return {
    events,
    unreadCount,
    activeEvent,
    isFetching,
    severityProjection,
    threatVelocityProjection,
    isFetchingThreatVelocityProjection,
    lastVelocityInvalidation,
    filteredEvents,
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