import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../../api/client'

// --- TYPES & INTERFACES ---
export interface RawSensorPayload {
  sensorId: string
  customName?: string
  status?: string
  isSilenced?: boolean
  events24h?: number
  envVars?: Record<string, any>
  metadata?: Record<string, any>
  lastHeartbeat?: string | null
}

export interface RawNodePayload {
  nodeId: string
  alias?: string
  status?: string
  hasPendingConfig?: boolean
  lastHeartbeat?: string | null
  publicIp?: string | null
  privateIp?: string | null
  tags?: string[]
  apiKey?: string | null
  activeRevision?: string
  desiredRevision?: string
  lastEvent?: string
  installedSensors?: RawSensorPayload[]
}

export interface UptimeBlock {
  status: string
  label: string
  timeLabel: string
}

export interface UptimeSensor {
  sensorId: string
  displayName: string
  status: string
  isSilenced: boolean
  blocks: UptimeBlock[]
}

export interface UptimeGroup {
  nodeId: string
  nodeAlias: string
  worstStatus: string
  sensors: UptimeSensor[]
}

export interface InstalledSensor {
  id: string
  sensorId: string
  nodeId: string
  name: string
  display: string
  status: string
  isSilenced: boolean
  events24h: number
  envVars: Record<string, any>
  metadata: Record<string, any>
  lastHeartbeat: string | null
}

export interface FleetNode {
  id: string
  alias: string
  status: string
  hasPendingConfig: boolean
  lastHeartbeat: string | null
  publicIp: string | null
  privateIp: string | null
  tags: string[]
  apiKey: string | null
  activeRevision: string
  desiredRevision: string
  lastEvent: string
  installedSensors?: InstalledSensor[]
}

export interface UptimeSummary {
  overallUptime?: number
  overall_uptime?: number // Backend backwards compatibility fallback
}

export interface UptimeData {
  timeframe: string
  generatedAt: string
  summary: UptimeSummary
  groups: UptimeGroup[]
}

export interface FleetState {
  nodesById: Record<string, FleetNode>
  sensorsById: Record<string, InstalledSensor>
  pendingNodeActions: Map<string, Set<string>>
  pendingSensorActions: Map<string, Set<string>>
  selectedNodeId: string | null
  selectedSensorId: string | null
  activeTimeframe: string
  uptimeData: UptimeData | null
  manifests: any[]
}

export const useFleetStore = defineStore('fleet', () => {
  // ==========================================
  // SINGLE STATE TREE
  // ==========================================
  const state = ref<FleetState>({
    nodesById: {},
    sensorsById: {},
    pendingNodeActions: new Map(),
    pendingSensorActions: new Map(),
    selectedNodeId: null,
    selectedSensorId: null,
    activeTimeframe: '24H',
    uptimeData: null,
    manifests: []
  })

  const abortControllers = new Map<string, AbortController>()

  // ==========================================
  // ENCAPSULATED GETTERS (Public API)
  // ==========================================
  const nodesById = computed(() => state.value.nodesById)
  const sensorsById = computed(() => state.value.sensorsById)
  const selectedNodeId = computed(() => state.value.selectedNodeId)
  const selectedSensorId = computed(() => state.value.selectedSensorId)
  const activeTimeframe = computed(() => state.value.activeTimeframe)
  const uptimeData = computed(() => state.value.uptimeData)
  const manifests = computed(() => state.value.manifests)
  const pendingNodeActions = computed(() => state.value.pendingNodeActions)
  const pendingSensorActions = computed(() => state.value.pendingSensorActions)

  // --- DYNAMIC READ MODELS ---
  const sensorsByNodeId = computed<Record<string, InstalledSensor[]>>(() => {
    const map: Record<string, InstalledSensor[]> = {}
    for (const s of Object.values(state.value.sensorsById)) {
      if (!map[s.nodeId]) map[s.nodeId] = []
      map[s.nodeId].push(s)
    }
    return map
  })

  const nodes = computed<FleetNode[]>(() => {
    return Object.values(state.value.nodesById).map(node => ({
      ...node,
      installedSensors: sensorsByNodeId.value[node.id] || []
    }))
  })

  const getNode = (id: string | null): FleetNode | null => {
    if (!id) return null
    const node = state.value.nodesById[id]
    if (!node) return null
    return {
      ...node,
      installedSensors: sensorsByNodeId.value[node.id] || []
    }
  }

  const getSensor = (nodeId: string | null, rawSensorId: string | null): InstalledSensor | null => {
    if (!nodeId || !rawSensorId) return null
    return state.value.sensorsById[`${nodeId}:${rawSensorId}`] || null
  }

  const selectedNode = computed<FleetNode | null>(() => getNode(state.value.selectedNodeId))
  const selectedSensor = computed<InstalledSensor | null>(() => {
    if (!state.value.selectedNodeId || !state.value.selectedSensorId) return null
    return state.value.sensorsById[state.value.selectedSensorId] || null
  })

  const overallUptime = computed<string>(() => {
    const summary = state.value.uptimeData?.summary
    if (summary?.overallUptime !== undefined) return summary.overallUptime.toFixed(2) + '%'
    if (summary?.overall_uptime !== undefined) return summary.overall_uptime.toFixed(2) + '%' // Fallback
    return '0.0%'
  })

  // --- UI METADATA HELPERS ---
  const markNodeAction = (nodeId: string, action: string): void => {
    const current = state.value.pendingNodeActions.get(nodeId) || new Set<string>()
    current.add(action)
    state.value.pendingNodeActions = new Map(state.value.pendingNodeActions).set(nodeId, current)
  }

  const clearNodeAction = (nodeId: string, action: string): void => {
    const current = state.value.pendingNodeActions.get(nodeId)
    if (!current) return
    current.delete(action)
    const next = new Map(state.value.pendingNodeActions)
    if (current.size === 0) next.delete(nodeId)
    else next.set(nodeId, current)
    state.value.pendingNodeActions = next
  }

  const isNodeActionPending = (nodeId: string, action: string): boolean => {
    return state.value.pendingNodeActions.get(nodeId)?.has(action) || false
  }

  const isNodeSilenced = (nodeId: string): boolean => {
    const nodeSensors = sensorsByNodeId.value[nodeId] || []
    if (!nodeSensors.length) return false
    return nodeSensors.every(s => s.isSilenced)
  }

  // --- THE 3 MUTATION GATEKEEPERS ---
  const commitStructuralSnapshot = (nextNodes: Record<string, FleetNode>, nextSensors: Record<string, InstalledSensor>): void => {
    state.value.nodesById = nextNodes
    state.value.sensorsById = nextSensors
  }

  const patchNode = (nodeId: string, patch: Partial<FleetNode>): FleetNode | null => {
    const existing = state.value.nodesById[nodeId]
    if (!existing) return null
    const previousState = { ...existing }
    state.value.nodesById[nodeId] = { ...existing, ...patch }
    return previousState
  }

  const patchSensor = (compositeSensorId: string, patch: Partial<InstalledSensor>): InstalledSensor | null => {
    const existing = state.value.sensorsById[compositeSensorId]
    if (!existing) return null
    const previousState = { ...existing }
    state.value.sensorsById[compositeSensorId] = { ...existing, ...patch }
    return previousState
  }

  // --- NORMALIZATION ---
  const normalizeNodeData = (raw: RawNodePayload): FleetNode | null => {
    const id = raw.nodeId
    if (!id) return null
    return {
      id,
      alias: raw.alias || 'Unnamed Node', 
      status: raw.status || 'unknown',
      hasPendingConfig: raw.hasPendingConfig ?? false,
      lastHeartbeat: raw.lastHeartbeat || null,
      publicIp: raw.publicIp || null,
      privateIp: raw.privateIp || null,
      tags: raw.tags || [],
      apiKey: raw.apiKey || null,
      activeRevision: raw.activeRevision || '',
      desiredRevision: raw.desiredRevision || '',
      lastEvent: raw.lastEvent || 'Never',
    }
  }

  const normalizeSensorData = (raw: RawSensorPayload, nodeId: string): InstalledSensor | null => {
    const rawId = raw.sensorId
    if (!rawId) return null
    const id = `${nodeId}:${rawId}`
    return {
      id,
      sensorId: rawId,
      nodeId,
      name: raw.customName || rawId,
      display: raw.customName || rawId,
      status: raw.status || 'down',
      isSilenced: raw.isSilenced ?? false,
      events24h: raw.events24h ?? 0,
      envVars: raw.envVars || {},
      metadata: raw.metadata || {},
      lastHeartbeat: raw.lastHeartbeat || null,
    }
  }

  // --- ACTIONS: STRUCTURAL INGESTION ---
  const fetchFleet = async (): Promise<void> => {
    try {
      const res = await api.get('/api/v1/nodes')
      const raw = await res.json() || []
      const incoming = Array.isArray(raw) ? raw : raw.nodes || []
  
      const nextNodes: Record<string, FleetNode> = {}
      const nextSensors: Record<string, InstalledSensor> = {}
  
      for (const rawNode of incoming) {
        const node = normalizeNodeData(rawNode)
        if (!node) continue
        nextNodes[node.id] = node
  
        const sensors = rawNode.installedSensors || []
  
        for (const rs of sensors) {
          const sensor = normalizeSensorData(rs, node.id)
          if (sensor) nextSensors[sensor.id] = sensor
        }
      }
  
      commitStructuralSnapshot(nextNodes, nextSensors)
    } catch (e) {
      console.error('Failed to fetch fleet data', e)
    }
  }

  const fetchNodeDetails = async (nodeId: string): Promise<void> => {
    if (abortControllers.has(nodeId)) abortControllers.get(nodeId)?.abort()
    const controller = new AbortController()
    abortControllers.set(nodeId, controller)

    try {
      const res = await api.get(`/api/v1/nodes/${encodeURIComponent(nodeId)}`, { signal: controller.signal })
      const rawNode = await res.json()
      const node = normalizeNodeData(rawNode)
      if (!node) return

      const nextNodes = { ...state.value.nodesById, [node.id]: node }
      const nextSensors = { ...state.value.sensorsById }

      const oldSensors = sensorsByNodeId.value[node.id] || []
      for (const s of oldSensors) delete nextSensors[s.id]

      const rawSensors = rawNode.installedSensors || []
      for (const rs of rawSensors) {
        const sensor = normalizeSensorData(rs, node.id)
        if (sensor) nextSensors[sensor.id] = sensor
      }

      commitStructuralSnapshot(nextNodes, nextSensors)
    } catch (e: any) {
      if (e.name !== 'AbortError') console.error('Failed to fetch node subgraph:', e)
    } finally {
      if (abortControllers.get(nodeId) === controller) abortControllers.delete(nodeId)
    }
  }

  // --- HIGH-FREQUENCY RUNTIME PATCHES ---
  const handleWsUpdate = (type: string, payload: any): void => {
    if (type === 'SENSOR_HEARTBEAT') {
      const compositeSensorId = `${payload.nodeId}:${payload.sensorId}`
      patchNode(payload.nodeId, { lastHeartbeat: payload.timestamp, status: 'up' })
      patchSensor(compositeSensorId, { lastHeartbeat: payload.timestamp, status: 'up' })
      return
    }

    if (type === 'NEW_EVENT') {
      const compositeSensorId = `${payload.nodeId}:${payload.sensorId}`
      patchNode(payload.nodeId, { lastEvent: 'Just now' })
      
      if (payload.events24h !== undefined) {
        patchSensor(compositeSensorId, { events24h: payload.events24h })
      }
      return
    }

    if (type === 'SILENCE_SENSOR') {
      const compositeSensorId = `${payload.nodeId}:${payload.sensorId}`
      if (payload.isSilenced !== undefined) {
        patchSensor(compositeSensorId, { isSilenced: payload.isSilenced })
      }
      return
    }

    if (['NEW_SENSOR', 'UPDATE_SENSOR', 'DELETE_SENSOR', 'NODE_SYNCED', 'UPDATE_NODE'].includes(type)) {
      fetchNodeDetails(payload.nodeId)
      return
    }
    
    if (type === 'NEW_NODE' || type === 'DELETE_NODE') {
      fetchFleet()
      return
    }
  }

  // --- ACTIONS: SELECTION ---
  const clearSelection = (): void => {
    state.value.selectedNodeId = null
    state.value.selectedSensorId = null
  }

  const selectTarget = (nodeId: string | null, rawSensorId: string | null = null, toggle: boolean = true): void => {
    const compositeId = rawSensorId && nodeId ? `${nodeId}:${rawSensorId}` : null

    const sameNode = state.value.selectedNodeId === nodeId
    const sameSensor = state.value.selectedSensorId === compositeId

    if (toggle && sameNode && sameSensor) {
      clearSelection()
      return
    }

    state.value.selectedNodeId = nodeId
    state.value.selectedSensorId = compositeId
  }

  const setActiveTimeframe = (timeframe: string): void => {
    state.value.activeTimeframe = timeframe
  }

  // --- ACTIONS: UPTIME & MANIFESTS ---
  const fetchUptime = async (timeframe?: string): Promise<void> => {
    const target = timeframe || state.value.activeTimeframe
    try {
      const res = await api.get(`/api/v1/uptime?timeframe=${target}`)
      state.value.uptimeData = (await res.json()) as UptimeData
    } catch (e) {
      console.error('Failed to fetch uptime', e)
    }
  }

  const fetchManifests = async (): Promise<any[]> => {
    if (state.value.manifests.length > 0) {
      return state.value.manifests
    }
    try {
      const res = await api.get('/api/v1/manifests')
      const data = await res.json()
      state.value.manifests = data
      return data
    } catch (err) {
      console.error('Failed to fetch manifests:', err)
      throw err
    }
  }
  
  // --- ACTIONS: COMPOSE / SYNC ---
  const fetchCompose = async (apiKey: string): Promise<string> => {
    try {
      const res = await api.request('/api/v1/nodes/compose', {
        headers: { Authorization: `Bearer ${apiKey}` },
      })
      return await res.text()
    } catch (err) {
      console.error('Failed to fetch compose:', err)
      throw err
    }
  }

  const generateCompose = async (payload: any): Promise<string> => {
    try {
      const res = await api.post('/api/v1/compose/generate', payload)
      return await res.text()
    } catch (err) {
      console.error('Failed to generate compose:', err)
      throw err
    }
  }

  const syncNode = async (nodeId: string): Promise<{ success: boolean; yaml?: string; error?: string }> => {
    const node = getNode(nodeId)
    if (!node) return { success: false, error: `Unable to sync: node ${nodeId} not found` }
    if (!node.apiKey) return { success: false, error: 'Unable to sync: missing node API key' }

    try {
      const composeYaml = await fetchCompose(node.apiKey)
      patchNode(nodeId, { hasPendingConfig: false })
      await fetchNodeDetails(nodeId)
      return { success: true, yaml: composeYaml }
    } catch (err: any) {
      return { success: false, error: err.message || 'Failed to sync node.' }
    }
  }

  // --- ACTIONS: NODE MUTATIONS ---
  const createNode = async (alias: string, tags: string[] = []): Promise<{ nodeId: string, apiKey: string }> => {
    try {
      const res = await api.post('/api/v1/nodes', { alias, tags })
      const data = await res.json()
      await fetchFleet()
      return { nodeId: data.nodeId, apiKey: data.apiKey }
    } catch (err) {
      console.error('Failed to create node:', err)
      throw err
    }
  }

  const updateNode = async (nodeId: string, payload: Partial<FleetNode>): Promise<void> => {
    let previousState: FleetNode | null = null
    try {
      previousState = patchNode(nodeId, payload)
      await api.patch(`/api/v1/nodes/${nodeId}`, payload)
    } catch (err) {
      if (previousState) patchNode(nodeId, previousState)
      console.error('Failed to update node:', err)
      throw err
    }
  }

  const deleteNode = async (nodeId: string): Promise<{ success: boolean; error?: string }> => {
    markNodeAction(nodeId, 'deleting')

    try {
      await api.delete(`/api/v1/nodes/${nodeId}`)
      if (state.value.selectedNodeId === nodeId) {
        state.value.selectedNodeId = null
        state.value.selectedSensorId = null
      }
      await fetchFleet()
      await fetchUptime()
      return { success: true }
    } catch (err) {
      console.error('Failed to delete node:', err)
      return { success: false, error: 'Failed to delete node.' }
    } finally {
      clearNodeAction(nodeId, 'deleting')
    }
  }

  // --- ACTIONS: SENSOR MUTATIONS ---
  const addSensor = async (nodeId: string, { sensorId: rawSensorId, customName, configValues }: { sensorId: string, customName?: string, configValues?: Record<string, string> }): Promise<void> => {
    try {
      await api.post(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors`, {
        sensorId: rawSensorId,
        customName: customName || rawSensorId,
        configValues: configValues,
      })
      fetchNodeDetails(nodeId)
    } catch (err) {
      console.error('Failed to add sensor:', err)
      throw err
    }
  }

  const updateSensor = async (nodeId: string, rawSensorId: string, { customName, configValues }: { customName?: string, configValues?: Record<string, string> }): Promise<void> => {
    const sensor = getSensor(nodeId, rawSensorId)
    if (!sensor) return
    try {
      await api.put(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors/${encodeURIComponent(rawSensorId)}`, {
        customName: customName || rawSensorId,
        configValues: configValues,
      })
      fetchNodeDetails(nodeId)
    } catch (err) {
      console.error('Failed to update sensor:', err)
      throw err
    }
  }

  const removeSensor = async (nodeId: string, rawSensorId: string): Promise<{ success: boolean; error?: string }> => {
    const sensor = getSensor(nodeId, rawSensorId)
    if (!sensor) return { success: false, error: 'Sensor not found.' }

    try {
      await api.delete(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors/${encodeURIComponent(rawSensorId)}`)
      await fetchNodeDetails(nodeId)
      const compositeId = `${nodeId}:${rawSensorId}`
      if (state.value.selectedSensorId === compositeId) state.value.selectedSensorId = null
      await fetchUptime()
      return { success: true }
    } catch (err) {
      console.error('Failed to remove sensor:', err)
      return { success: false, error: 'Failed to remove sensor.' }
    }
  }

  const toggleSilence = async (nodeId: string, rawSensorId: string, targetState: boolean): Promise<void> => {
    const sensor = getSensor(nodeId, rawSensorId)
    if (!sensor) return

    const previousState = patchSensor(sensor.id, { isSilenced: targetState })

    try {
      await api.patch(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors/${encodeURIComponent(rawSensorId)}/silence`, {
        isSilenced: targetState,
      })
    } catch (err) {
      if (previousState) patchSensor(sensor.id, previousState)
      console.error('Failed to toggle sensor silence:', err)
      throw err
    }
  }

  const silenceNode = async (nodeId: string): Promise<void> => {
    const nodeSensors = sensorsByNodeId.value[nodeId] || []
    if (!nodeSensors.length) return

    const allSilenced = nodeSensors.every(s => s.isSilenced)
    const targetState = !allSilenced

    const prevStates = new Map<string, InstalledSensor | null>()
    nodeSensors.forEach(s => {
      const prev = patchSensor(s.id, { isSilenced: targetState })
      prevStates.set(s.id, prev)
    })

    try {
      await Promise.all(nodeSensors.map(s =>
        api.patch(`/api/v1/nodes/${nodeId}/sensors/${encodeURIComponent(s.sensorId)}/silence`, {
          isSilenced: targetState,
        })
      ))
    } catch (err) {
      nodeSensors.forEach(s => {
        const prev = prevStates.get(s.id)
        if (prev) patchSensor(s.id, prev)
      })
      console.error('Failed to silence node:', err)
      throw err
    }
  }

  return {
    nodesById, sensorsById,
    selectedNodeId, selectedSensorId, activeTimeframe, uptimeData,
    pendingNodeActions, pendingSensorActions,
    selectedNode, selectedSensor,
    nodes, sensorsByNodeId, overallUptime, manifests,
    getNode, getSensor,
    isNodeActionPending, isNodeSilenced,
    fetchFleet, fetchNodeDetails, fetchUptime, fetchManifests,
    selectTarget, clearSelection, setActiveTimeframe,
    createNode, deleteNode, updateNode,
    addSensor, updateSensor, removeSensor, toggleSilence, silenceNode,
    fetchCompose, generateCompose, syncNode,
    handleWsUpdate,
  }
})