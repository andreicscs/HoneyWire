import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../../api/client'
// --- TYPES & INTERFACES ---
export interface RawSensorPayload {
  sensorId: string
  customName?: string
  status?: string
  isSilenced?: boolean
  envVars?: Record<string, any>
  metadata?: Record<string, any>
  deployedVersion?: string
  updateAvailable?: boolean
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
  hasUpdateAvailable?: boolean
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
  envVars: Record<string, any>
  metadata: Record<string, any>
  deployedVersion?: string
  updateAvailable?: boolean
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
  totalSensors?: number
  onlineSensors?: number
  isSilenced?: boolean
  hasUpdateAvailable?: boolean
  sensorSummary?: { type: string; count: number; sensors: any[] }[]
  isAwaitingCheckIn?: boolean
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
  const DEFAULT_SENSOR_ICON = 'M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z M3.27 6.96L12 12.01L20.73 6.96 M12 22.08V12'
  const manifests = computed(() => state.value.manifests.map(m => ({ ...m, icon_svg: m.icon_svg || DEFAULT_SENSOR_ICON })))
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

  const enrichedNodes = computed(() => {
    const manifestMap = new Map()
    for (const s of manifests.value) {
      manifestMap.set(s.id, s)
      manifestMap.set(s.sensorId, s)
      manifestMap.set(s.name, s)
    }

    return nodes.value.filter(n => !n.id.startsWith('__pending_')).map(node => {
      const sensorsList = node.installedSensors || []

      const enrichedSensors = sensorsList.map(sensor => {
        const manifest = manifestMap.get(sensor.id) || manifestMap.get(sensor.name) || manifestMap.get(sensor.sensorId)
        return {
          ...sensor,
          display: manifest?.name || sensor.display || sensor.name || '',
          icon: manifest?.icon_svg || sensor.metadata?.icon || DEFAULT_SENSOR_ICON,
          osi: manifest?.osi_layer || sensor.metadata?.osi || 'Other',
          status: (node.status === 'down' && sensor.status === 'pending') ? 'down' : sensor.status
        }
      })

      const totalSensors = enrichedSensors.length
      const onlineSensors = enrichedSensors.filter(s => ['up', 'online'].includes((s.status || '').toLowerCase())).length
      const isSilenced = totalSensors > 0 && enrichedSensors.every(s => s.isSilenced)

      const osiGroups = new Map()
      for (const sensor of enrichedSensors) {
        if (!osiGroups.has(sensor.osi)) osiGroups.set(sensor.osi, [])
        osiGroups.get(sensor.osi).push({ name: sensor.display, status: sensor.status })
      }

      const osiOrder = ['Physical', 'Data Link', 'Network', 'Transport', 'Session', 'Presentation', 'Application', 'Other']
      const sensorSummary = totalSensors > 0
        ? Array.from(osiGroups.entries())
            .map(([type, groupSensors]) => ({ type, count: groupSensors.length, sensors: groupSensors }))
            .sort((a, b) => {
              const aIdx = osiOrder.indexOf(a.type), bIdx = osiOrder.indexOf(b.type)
              if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx
              if (aIdx !== -1) return -1
              if (bIdx !== -1) return 1
              return a.type.localeCompare(b.type)
            })
        : []

      return {
        ...node, 
        installedSensors: enrichedSensors,
        totalSensors, onlineSensors, isSilenced, sensorSummary,
        isAwaitingCheckIn: node.status === 'pending' || (!node.lastHeartbeat && totalSensors === 0)
      }
    })
  })

  const hydratedUptimeGroups = computed(() => {
    const groups = state.value.uptimeData?.groups || []
    return groups.map(group => ({
      ...group,
      sensors: (group.sensors || []).map(sensor => {
        let liveStatus = sensor.status || 'unknown'
        const liveSensor = getSensor(group.nodeId, sensor.sensorId)
        if (liveSensor && liveSensor.status) liveStatus = liveSensor.status

        const isLiveOnline = ['online', 'alive', 'up'].includes(liveStatus.toLowerCase())
        const isPending = liveStatus.toLowerCase() === 'pending'

        let blocks = [...(sensor.blocks || [])]
        let worstStatus: string | null = null

        if (blocks.length > 0) {
          if (isPending) {
            blocks = blocks.map(b => b.status !== 'nodata' ? { ...b, status: 'pending', label: 'Awaiting Initial Check-in' } : b)
          } else {
            const lastIdx = blocks.length - 1
            const lastBlock = { ...blocks[lastIdx] }
            if (!isLiveOnline) {
              lastBlock.status = 'down'
              lastBlock.label = 'Offline'
            } else if (['down', 'nodata', 'pending'].includes(lastBlock.status)) {
              lastBlock.status = 'up'
              lastBlock.label = 'Online'
            }
            blocks[lastIdx] = lastBlock
          }

          for (let i = 0; i < blocks.length; i++) {
            if (blocks[i].status === 'down') worstStatus = 'down'
            else if (blocks[i].status === 'degraded' && worstStatus !== 'down') worstStatus = 'degraded'
          }
        }
        
        const isSilenced = liveSensor ? !!liveSensor.isSilenced : false

        return { ...sensor, nodeId: group.nodeId, liveStatus: liveStatus.toLowerCase(), blocks, worstStatus, isSilenced }
      })
    }))
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
      hasUpdateAvailable: raw.hasUpdateAvailable || false,
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
      deployedVersion: raw.deployedVersion || '',
      updateAvailable: raw.updateAvailable || false,
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
      patchNode(payload.nodeId, { lastEvent: 'Just now' })
      return
    }

    if (type === 'SILENCE_SENSOR') {
      const compositeSensorId = `${payload.nodeId}:${payload.sensorId}`
      if (payload.isSilenced !== undefined) {
        patchSensor(compositeSensorId, { isSilenced: payload.isSilenced })
      }
      return
    }

    if (type === 'CATALOG_UPDATED') {
      fetchFleet()
      fetchManifests()
      return
    }

    if (['NEW_SENSOR', 'UPDATE_SENSOR', 'DELETE_SENSOR', 'NODE_SYNCED', 'UPDATE_NODE'].includes(type)) {
      fetchNodeDetails(payload.nodeId)
      fetchManifests()
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
    try {
      const res = await api.get('/api/v1/manifests')
      const data = await res.json()
      // The API returns the raw index.json which has a top-level { sensors: [] } wrapper
      const sensorsArray = Array.isArray(data) ? data : (data?.sensors || [])
      state.value.manifests = sensorsArray
      return sensorsArray
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

      // TODO REMOVE DEBUG After the first node is successfully created, disable the "first startup" UI hints.
      localStorage.removeItem('DEBUG_FIRST_STARTUP')

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

  const upgradeNode = async (nodeId: string): Promise<void> => {
    try {
      await api.post(`/api/v1/nodes/${nodeId}/upgrade`)
      fetchNodeDetails(nodeId)
    } catch (err: any) {
      console.error('Node Upgrade Failed:', err.message || 'Failed to upgrade node')
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

  const upgradeSensor = async (nodeId: string, rawSensorId: string) => {
    if (!nodeId || !rawSensorId) throw new Error('Missing required parameters')
    
    try {
      await api.post(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors/${encodeURIComponent(rawSensorId)}/upgrade`)
      fetchNodeDetails(nodeId)
    } catch (err: any) {
      console.error('Sensor Upgrade Failed:', err.message || 'Failed to upgrade sensor')
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
    getNode, getSensor, hydratedUptimeGroups, enrichedNodes,
    isNodeActionPending, isNodeSilenced,
    fetchFleet, fetchNodeDetails, fetchUptime, fetchManifests,
    selectTarget, clearSelection, setActiveTimeframe,
    createNode, deleteNode, updateNode, upgradeNode,
    addSensor, updateSensor, upgradeSensor, removeSensor, toggleSilence, silenceNode,
    fetchCompose, generateCompose, syncNode,
    handleWsUpdate,
  }
})