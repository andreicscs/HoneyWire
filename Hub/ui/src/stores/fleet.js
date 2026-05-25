import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'

export const useFleetStore = defineStore('fleet', () => {
  // --- 1. AUTHORITATIVE BACKEND SNAPSHOT STATE ---
  const nodesById = ref({})
  const sensorsById = ref({})
  
  // --- 2. FRONTEND OPERATIONAL UI STATE ---
  const pendingNodeActions = ref(new Map())
  const pendingSensorActions = ref(new Map())
  const selectedNodeId = ref(null)
  const selectedSensorId = ref(null)
  const activeTimeframe = ref('24H')
  
  // Existing state for uptime
  const uptimeData = ref([])

  // --- 3. TRANSPORT CONTROL (Race Condition Protection) ---
  const abortControllers = new Map()

  // --- UI METADATA HELPERS ---
  const markNodeAction = (nodeId, action) => {
    const current = pendingNodeActions.value.get(nodeId) || new Set()
    current.add(action)
    pendingNodeActions.value = new Map(pendingNodeActions.value).set(nodeId, current)
  }

  const clearNodeAction = (nodeId, action) => {
    const current = pendingNodeActions.value.get(nodeId)
    if (!current) return
    current.delete(action)
    const next = new Map(pendingNodeActions.value)
    if (current.size === 0) next.delete(nodeId)
    else next.set(nodeId, current)
    pendingNodeActions.value = next
  }

  const isNodeActionPending = (nodeId, action) => {
    return pendingNodeActions.value.get(nodeId)?.has(action) || false
  }

  // --- THE 3 MUTATION GATEKEEPERS ---
  const commitStructuralSnapshot = (nextNodes, nextSensors) => {
    nodesById.value = nextNodes
    sensorsById.value = nextSensors
  }

  const patchNode = (nodeId, patch) => {
    const existing = nodesById.value[nodeId]
    if (!existing) return null
    const previousState = { ...existing }
    nodesById.value[nodeId] = { ...existing, ...patch }
    return previousState
  }

  const patchSensor = (compositeSensorId, patch) => {
    const existing = sensorsById.value[compositeSensorId]
    if (!existing) return null
    const previousState = { ...existing }
    sensorsById.value[compositeSensorId] = { ...existing, ...patch }
    return previousState
  }

  // --- DYNAMIC READ MODELS (Selectors) ---
  const sensorsByNodeId = computed(() => {
    const map = {}
    for (const s of Object.values(sensorsById.value)) {
      (map[s.nodeId] ||= []).push(s)
    }
    return map
  })

  const nodes = computed(() => {
    return Object.values(nodesById.value).map(node => ({
      ...node,
      installedSensors: sensorsByNodeId.value[node.id] || []
    }))
  })

  const getNode = (id) => {
    const node = nodesById.value[id]
    if (!node) return null
    return {
      ...node,
      installedSensors: sensorsByNodeId.value[node.id] || []
    }
  }

  const getSensor = (nodeId, rawSensorId) => {
    if (!nodeId || !rawSensorId) return null
    return sensorsById.value[`${nodeId}:${rawSensorId}`] || null
  }

  // Computed properties for selected targets - Exported as Objects (Option A)
  const selectedNode = computed(() => getNode(selectedNodeId.value))
  const selectedSensor = computed(() => {
    if (!selectedNodeId.value || !selectedSensorId.value) return null;
    return sensorsById.value[selectedSensorId.value] || null;
  })

  // --- NORMALIZATION ---
  const normalizeNodeData = (raw) => {
    const id = raw.node_id ?? raw.id
    if (!id) return null
    return {
      id,
      alias: raw.alias || raw.name || 'Unnamed Node', 
      status: raw.status || 'unknown',
      hasPendingConfig: raw.hasPendingConfig ?? raw.pending_config ?? raw.has_pending_config ?? false,
      lastHeartbeat: raw.lastHeartbeat || raw.last_heartbeat || null,
      publicIp: raw.publicIp || raw.public_ip || null,
      privateIp: raw.privateIp || raw.private_ip || null,
      tags: raw.tags || [],
      apiKey: raw.apiKey || raw.api_key || null,
      activeRevision: raw.activeRevision || raw.active_revision || '',
      desiredRevision: raw.desiredRevision || raw.desired_revision || '',
      lastEvent: raw.lastEvent || raw.last_event || 'Never',
    }
  }

  const normalizeSensorData = (raw, nodeId) => {
    const rawId = raw.sensor_id ?? raw.id
    if (!rawId) return null
    const id = `${nodeId}:${rawId}`
    return {
      id,
      sensorId: rawId,
      nodeId,
      name: raw.name || raw.sensor_id || rawId,
      display: raw.display || raw.custom_name || raw.name || raw.sensor_id || rawId,
      status: raw.status || 'down',
      isSilenced: raw.isSilenced ?? raw.is_silenced ?? false,
      events24h: raw.events24h ?? raw.events_24h ?? 0,
      osi: raw.osi_layer || raw.osi || 'Sensor',
      icon: raw.icon || raw.icon_svg || 'M12 12h0',
      envVars: raw.envVars || raw.env_vars || {},
      metadata: raw.metadata || {},
      lastHeartbeat: raw.lastHeartbeat || raw.last_heartbeat || null,
    }
  }

  // --- ACTIONS: STRUCTURAL INGESTION ---
  const fetchFleet = async () => {
    try {
      const res = await api.get('/api/v1/nodes')
      const raw = await res.json() || []
      const incoming = Array.isArray(raw) ? raw : raw.nodes || []
  
      const nextNodes = {}
      const nextSensors = {}
  
      for (const rawNode of incoming) {
        const node = normalizeNodeData(rawNode)
        if (!node) continue
        nextNodes[node.id] = node
  
        const sensors = rawNode.installedSensors || rawNode.installed_sensors || []
  
        for (const rs of sensors) {
          const sensor = normalizeSensorData(rs, node.id)
          if (sensor) {
            nextSensors[sensor.id] = sensor
          }
        }
      }
  
      commitStructuralSnapshot(nextNodes, nextSensors)
    } catch (e) {
      console.error('Failed to fetch fleet data', e)
    }
  }

  const fetchNodeDetails = async (nodeId) => {
    if (abortControllers.has(nodeId)) abortControllers.get(nodeId).abort()
    const controller = new AbortController()
    abortControllers.set(nodeId, controller)

    try {
      const res = await api.get(`/api/v1/nodes/${encodeURIComponent(nodeId)}`, { signal: controller.signal })
      const rawNode = await res.json()
      const node = normalizeNodeData(rawNode)
      if (!node) return

      const nextNodes = { ...nodesById.value, [node.id]: node }
      const nextSensors = { ...sensorsById.value }

      // O(K) Garbage Collection using the computed index
      const oldSensors = sensorsByNodeId.value[node.id] || []
      for (const s of oldSensors) delete nextSensors[s.id]

      const rawSensors = rawNode.installedSensors || rawNode.installed_sensors || []
      for (const rs of rawSensors) {
        const sensor = normalizeSensorData(rs, node.id)
        if (sensor) nextSensors[sensor.id] = sensor
      }

      commitStructuralSnapshot(nextNodes, nextSensors)
    } catch (e) {
      if (e.name !== 'AbortError') console.error('Failed to fetch node subgraph:', e)
    } finally {
      if (abortControllers.get(nodeId) === controller) abortControllers.delete(nodeId)
    }
  }

  // --- HIGH-FREQUENCY RUNTIME PATCHES ---
  const handleWsUpdate = (type, payload) => {
    if (type === 'SENSOR_HEARTBEAT') {
      const compositeSensorId = `${payload.node_id}:${payload.sensor_id}`
      patchNode(payload.node_id, { lastHeartbeat: payload.timestamp, status: 'up' })
      patchSensor(compositeSensorId, { lastHeartbeat: payload.timestamp, status: 'up' })
      return
    }

    if (type === 'NEW_EVENT') {
      const compositeSensorId = `${payload.node_id}:${payload.sensor_id}`
      patchNode(payload.node_id, { lastEvent: 'Just now' })
      
      const events = payload.events24h ?? payload.events_24h
      if (events !== undefined) {
        patchSensor(compositeSensorId, { events24h: events })
      }
      return
    }

    if (type === 'SILENCE_SENSOR') {
      const compositeSensorId = `${payload.node_id}:${payload.sensor_id}`
      const targetState = payload.is_silenced ?? payload.isSilenced
      if (targetState !== undefined) {
        patchSensor(compositeSensorId, { isSilenced: targetState })
      }
      return
    }

    // Structural boundaries trigger refetches, NEVER local mapping mutations
    if (type === 'NEW_SENSOR' || type === 'UPDATE_SENSOR' || type === 'DELETE_SENSOR' || type === 'NODE_SYNCED' || type === 'UPDATE_NODE') {
      fetchNodeDetails(payload.node_id || payload.id)
      return
    }
    
    if (type === 'NEW_NODE' || type === 'DELETE_NODE') {
      fetchFleet()
      return
    }
  }

  const deleteNode = async (nodeId) => {
    if (!confirm(`Delete Node "${nodeId}" and ALL of its underlying sensors?`)) return
    markNodeAction(nodeId, 'deleting')

    try {
      await api.delete(`/api/v1/nodes/${nodeId}`)
      if (selectedNodeId.value === nodeId) {
        selectedNodeId.value = null
        selectedSensorId.value = null
      }
      await fetchFleet()
    } catch (err) {
      console.error('Failed to delete node:', err)
    } finally {
      clearNodeAction(nodeId, 'deleting')
    }
  }

  // --- ACTIONS: SELECTION ---
  const clearSelection = () => {
    selectedNodeId.value = null
    selectedSensorId.value = null
  }

  const selectTarget = (nodeId, rawSensorId = null) => {
    const compositeId = rawSensorId ? `${nodeId}:${rawSensorId}` : null

    const sameNode = selectedNodeId.value === nodeId
    const sameSensor = selectedSensorId.value === compositeId

    if (sameNode && sameSensor) {
      clearSelection()
      return
    }

    selectedNodeId.value = nodeId
    selectedSensorId.value = compositeId
  }

  // --- ACTIONS: UPTIME & MANIFESTS ---
  const fetchUptime = async (timeframe) => {
    const target = timeframe || activeTimeframe.value
    try {
      const res = await api.get(`/api/v1/uptime?timeframe=${target}`)
      uptimeData.value = await res.json() || []
    } catch (e) {
      console.error('Failed to fetch uptime', e)
    }
  }

  const overallUptime = computed(() => {
    if (uptimeData.value && uptimeData.value.summary && typeof uptimeData.value.summary.overall_uptime === 'number') {
      return uptimeData.value.summary.overall_uptime.toFixed(2) + '%'
    }
    return '0.0%'
  })

  const fetchManifests = async () => {
    try {
      const res = await api.get('/api/v1/manifests')
      return await res.json()
    } catch (err) {
      console.error('Failed to fetch manifests:', err)
      throw err
    }
  }
  
  // --- ACTIONS: COMPOSE / SYNC ---
  const fetchCompose = async (apiKey) => {
    try {
      const res = await api.request('/api/v1/nodes/compose', {
        headers: { Authorization: `Bearer ${apiKey}` },
      })
      if (!res.ok) {
        throw new Error(`Failed to fetch compose YAML: ${res.statusText}`)
      }
      return await res.text()
    } catch (err) {
      console.error('Failed to fetch compose:', err)
      throw err
    }
  }

  const generateCompose = async (payload) => {
    try {
      const res = await api.post('/api/v1/compose/generate', payload)
      return await res.text()
    } catch (err) {
      console.error('Failed to generate compose:', err)
      throw err
    }
  }

  const syncNode = async (nodeId) => {
    const node = getNode(nodeId)
    if (!node) {
      throw new Error(`Unable to sync: node ${nodeId} not found`)
    }
    if (!node.apiKey) {
      throw new Error('Unable to sync: missing node API key')
    }

    const composeYaml = await fetchCompose(node.apiKey)
    patchNode(nodeId, { hasPendingConfig: false })
    fetchNodeDetails(nodeId)

    return composeYaml
  }

  // --- ACTIONS: NODE MUTATIONS ---
  const createNode = async (alias, tags = []) => {
    try {
      const res = await api.post('/api/v1/nodes', { alias, tags })
      const data = await res.json()
      
      const nodeId = data.node_id || data.nodeId || data.id
      const apiKey = data.api_key || data.apiKey || data.key
      
      await fetchFleet()
      
      return {
        nodeId,
        apiKey,
      }
    } catch (err) {
      console.error('Failed to create node:', err)
      throw err
    }
  }

  const updateNode = async (nodeId, payload) => {
    try {
      // Optimistic update using rollback cache
      const previousState = patchNode(nodeId, {
        alias: payload.alias !== undefined ? payload.alias : undefined,
        tags: payload.tags !== undefined ? payload.tags : undefined,
        publicIp: payload.publicIp !== undefined ? payload.publicIp : undefined,
        privateIp: payload.privateIp !== undefined ? payload.privateIp : undefined,
      })

      await api.patch(`/api/v1/nodes/${nodeId}`, payload)
    } catch (err) {
      if (previousState) patchNode(nodeId, previousState)
      console.error('Failed to update node:', err)
      throw err
    }
  }

  // --- ACTIONS: SENSOR MUTATIONS ---
  const addSensor = async (nodeId, { sensorId: rawSensorId, customName, configValues }) => {
    try {
      await api.post(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors`, {
        sensor_id: rawSensorId,
        custom_name: customName || rawSensorId,
        config_values: configValues,
      })
      fetchNodeDetails(nodeId)
    } catch (err) {
      console.error('Failed to add sensor:', err)
      throw err
    }
  }

  const updateSensor = async (nodeId, rawSensorId, { customName, configValues }) => {
    const sensor = getSensor(nodeId, rawSensorId)
    if (!sensor) return
    try {
      await api.put(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors/${encodeURIComponent(rawSensorId)}`, {
        custom_name: customName || rawSensorId,
        config_values: configValues,
      })
      fetchNodeDetails(nodeId)
    } catch (err) {
      console.error('Failed to update sensor:', err)
      throw err
    }
  }

  const removeSensor = async (nodeId, rawSensorId) => {
    if (!confirm('Remove this sensor? The node will be marked for deployment sync.')) return
    const sensor = getSensor(nodeId, rawSensorId)
    if (!sensor) return

    try {
      await api.delete(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors/${encodeURIComponent(rawSensorId)}`)
      fetchNodeDetails(nodeId)
      const compositeId = `${nodeId}:${rawSensorId}`
      if (selectedSensorId.value === compositeId) selectedSensorId.value = null
    } catch (err) {
      console.error('Failed to remove sensor:', err)
      throw err
    }
  }

  const toggleSilence = async (nodeId, rawSensorId, targetState) => {
    const sensor = getSensor(nodeId, rawSensorId)
    if (!sensor) return

    const previousState = patchSensor(sensor.id, { isSilenced: targetState })

    try {
      await api.patch(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors/${encodeURIComponent(rawSensorId)}/silence`, {
        is_silenced: targetState,
      })
    } catch (err) {
      if (previousState) patchSensor(sensor.id, previousState)
      console.error('Failed to toggle sensor silence:', err)
      throw err
    }
  }

  const silenceNode = async (nodeId) => {
    const nodeSensors = sensorsByNodeId.value[nodeId] || []
    if (!nodeSensors.length) return

    const allSilenced = nodeSensors.every(s => s.isSilenced)
    const targetState = !allSilenced

    const prevStates = new Map()
    nodeSensors.forEach(s => {
      const prev = patchSensor(s.id, { isSilenced: targetState })
      prevStates.set(s.id, prev)
    })

    try {
      await Promise.all(nodeSensors.map(s =>
        api.patch(`/api/v1/nodes/${nodeId}/sensors/${encodeURIComponent(s.sensorId)}/silence`, {
          is_silenced: targetState,
        })
      ))
    } catch (err) {
      nodeSensors.forEach(s => {
        patchSensor(s.id, prevStates.get(s.id))
      })
      console.error('Failed to silence node:', err)
      throw err
    }
  }

  // --- Phase 8: Exports ---
  return {
    // State
    nodesById, sensorsById,
    selectedNodeId, selectedSensorId, activeTimeframe, uptimeData,
    pendingNodeActions, pendingSensorActions,
    // Aliases
    selectedNode, selectedSensor,
    
    // Computed / Projection
    nodes, sensorsByNodeId, overallUptime,
    
    // Getters
    getNode, getSensor,
    isNodeActionPending,

    // Actions
    fetchFleet, fetchNodeDetails, fetchUptime, fetchManifests,
    selectTarget, clearSelection,
    createNode, deleteNode, updateNode,
    addSensor, updateSensor, removeSensor, toggleSilence, silenceNode,
    fetchCompose, generateCompose, syncNode,
    handleWsUpdate,
    normalizeSensorData, normalizeNodeData,
  }
})
