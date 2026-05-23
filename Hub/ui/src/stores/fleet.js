import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client'

export const useFleetStore = defineStore('fleet', () => {
  // --- STATE ---
  const nodes = ref([])
  const uptimeData = ref([])
  const selectedNode = ref(null)
  const selectedSensor = ref(null)
  const activeTimeframe = ref('24H')

  // --- NORMALIZATION ---

  const normalizeSensor = (raw) => {
    if (!raw) return null
    return {
      ...raw,
      id: raw.id || raw.sensor_id || raw.name,
      name: raw.name || raw.sensor_id || raw.id,
      display: raw.display || raw.custom_name || raw.name || raw.sensor_id || raw.id,
      status: raw.status || 'down',
      events24h: raw.events24h ?? raw.events_24h ?? 0,
      isSilenced: raw.isSilenced ?? raw.is_silenced ?? false,
      osi: raw.osi_layer || raw.osi || 'Sensor',
      icon: raw.icon || raw.icon_svg || 'M12 12h0',
      envVars: raw.envVars || raw.env_vars || {},
      metadata: raw.metadata || {},
      lastHeartbeat: raw.lastHeartbeat || raw.last_heartbeat || null,
    }
  }

  const normalizeNode = (raw) => {
    if (!raw) return null
    return {
      ...raw,
      id: raw.id || raw.node_id || raw.nodeId,
      alias: raw.alias || raw.name || 'Unnamed Node',
      status: raw.status || 'unknown',
      publicIp: raw.publicIp || raw.public_ip || null,
      privateIp: raw.privateIp || raw.private_ip || null,
      tags: raw.tags || [],
      apiKey: raw.apiKey || raw.api_key || null,
      hasPendingConfig: raw.hasPendingConfig ?? raw.pending_config ?? raw.has_pending_config ?? false,
      activeRevision: raw.activeRevision || raw.active_revision || '',
      desiredRevision: raw.desiredRevision || raw.desired_revision || '',
      lastEvent: raw.lastEvent || raw.last_event || 'Never',
      lastHeartbeat: raw.lastHeartbeat || raw.last_heartbeat || null,
      installedSensors: (raw.installedSensors || raw.installed_sensors || []).map(normalizeSensor),
    }
  }

  // --- MERGE HELPERS ---

  const mergeSensor = (existing, incoming) => {
    Object.assign(existing, incoming)
  }

  const mergeNode = (existing, incoming) => {
    const incomingSensors = incoming.installedSensors || []
    if (!existing.installedSensors) existing.installedSensors = []
    const existingSensors = existing.installedSensors

    const incomingSensorIds = new Set(incomingSensors.map(s => s.id))

    incomingSensors.forEach(newSensor => {
      const existingSensor = existingSensors.find(s => s.id === newSensor.id)
      if (existingSensor) {
        mergeSensor(existingSensor, newSensor)
      } else {
        existingSensors.push(newSensor)
      }
    })

    for (let i = existingSensors.length - 1; i >= 0; i--) {
      if (!incomingSensorIds.has(existingSensors[i].id)) {
        existingSensors.splice(i, 1)
      }
    }

    // Copy properties safely to preserve Vue 3 Proxy array identity.
    // This prevents breaking the reactive reference that components are watching.
    for (const key of Object.keys(incoming)) {
      if (key !== 'installedSensors') {
        existing[key] = incoming[key]
      }
    }
  }

  // --- INDEXED ACCESS ---

  const nodeMap = computed(() => {
    const map = {}
    for (const node of nodes.value) {
      map[node.id] = node
    }
    return map
  })

  const sensorIndex = computed(() => {
    const map = {}
    for (const node of nodes.value) {
      const sensors = {}
      for (const sensor of (node.installedSensors || [])) {
        sensors[sensor.id] = sensor
      }
      map[node.id] = sensors
    }
    return map
  })

  const getNode = (nodeId) => nodeMap.value[nodeId] || null
  const getSensor = (nodeId, sensorId) => sensorIndex.value[nodeId]?.[sensorId] || null

  // --- GETTERS ---

  const overallUptime = computed(() => {
    // Use the overall_uptime from the API response if available
    if (uptimeData.value && uptimeData.value.summary && typeof uptimeData.value.summary.overall_uptime === 'number') {
      return uptimeData.value.summary.overall_uptime.toFixed(2) + '%'
    }
    return '0.0%'
  })

  // --- ACTIONS: FETCH ---

  const fetchFleet = async () => {
    try {
      const res = await api.get('/api/v1/nodes')
      const raw = await res.json() || []
      const incoming = (Array.isArray(raw) ? raw : raw.nodes || []).map(normalizeNode)

      const incomingIds = new Set(incoming.map(n => n.id))

      incoming.forEach(newNode => {
        const existing = getNode(newNode.id)
        if (existing) {
          mergeNode(existing, newNode)
        } else {
          nodes.value.push(newNode)
        }
      })

      for (let i = nodes.value.length - 1; i >= 0; i--) {
        if (!incomingIds.has(nodes.value[i].id)) {
          nodes.value.splice(i, 1)
        }
      }
    } catch (e) {
      console.error('Failed to fetch fleet data', e)
    }
  }

  const fetchNodeDetails = async (nodeId) => {
    try {
      const res = await api.get(`/api/v1/nodes/${encodeURIComponent(nodeId)}`)
      const raw = await res.json()
      const normalized = normalizeNode(raw)

      const existing = getNode(nodeId)
      if (existing) {
        mergeNode(existing, normalized)
        return existing
      } else {
        nodes.value.push(normalized)
        return normalized
      }
    } catch (e) {
      console.error('Failed to fetch node details:', e)
      return null
    }
  }

  const fetchUptime = async (timeframe) => {
    const target = timeframe || activeTimeframe.value
    try {
      const res = await api.get(`/api/v1/uptime?timeframe=${target}`)
      uptimeData.value = await res.json() || []
    } catch (e) {
      console.error('Failed to fetch uptime', e)
    }
  }

  const fetchManifests = async () => {
    try {
      const res = await api.get('/api/v1/manifests')
      return await res.json()
    } catch (err) {
      console.error('Failed to fetch manifests:', err)
      throw err
    }
  }

  // --- ACTIONS: SELECTION ---

  const clearSelection = () => {
    selectedNode.value = null
    selectedSensor.value = null
  }

  const selectTarget = (nodeId, sensorId = null) => {
    const sameNode = selectedNode.value === nodeId
    const sameSensor = selectedSensor.value === sensorId

    if (sameNode && sameSensor) {
      clearSelection()
      return
    }

    selectedNode.value = nodeId
    selectedSensor.value = sensorId
  }

  // --- ACTIONS: NODE MUTATIONS ---

  const createNode = async (alias, tags = []) => {
    try {
      const res = await api.post('/api/v1/nodes', { alias, tags })
      const data = await res.json()

      // Optimistic add — partial node, background fetch fills details
      const partialNode = normalizeNode({
        id: data.node_id || data.nodeId || data.id,
        alias,
        tags,
        status: 'pending',
        apiKey: data.api_key || data.apiKey || data.key,
        installedSensors: [],
      })
      nodes.value.push(partialNode)

      // Background refresh for full details
      fetchNodeDetails(partialNode.id)

      return {
        nodeId: partialNode.id,
        apiKey: partialNode.apiKey,
      }
    } catch (err) {
      console.error('Failed to create node:', err)
      throw err
    }
  }

  const deleteNode = async (nodeId) => {
    if (!confirm(`Delete Node "${nodeId}" and ALL of its underlying sensors?`)) return

    const previousIdx = nodes.value.findIndex(n => n.id === nodeId)
    const previous = previousIdx !== -1 ? nodes.value[previousIdx] : null

    if (previousIdx !== -1) nodes.value.splice(previousIdx, 1)
    if (selectedNode.value === nodeId) clearSelection()

    try {
      await api.delete(`/api/v1/nodes/${nodeId}`)
    } catch (err) {
      console.error('Failed to delete node:', err)
      if (previous && previousIdx !== -1) {
        nodes.value.splice(previousIdx, 0, previous)
      }
      fetchFleet()
    }
  }

  const updateNode = async (nodeId, payload) => {
    const node = getNode(nodeId)
    if (!node) return

    const previous = {
      alias: node.alias,
      tags: [...node.tags],
      publicIp: node.publicIp,
      privateIp: node.privateIp,
    }

    if (payload.alias !== undefined) node.alias = payload.alias
    if (payload.tags !== undefined) node.tags = payload.tags
    if (payload.publicIp !== undefined) node.publicIp = payload.publicIp
    if (payload.privateIp !== undefined) node.privateIp = payload.privateIp

    try {
      await api.patch(`/api/v1/nodes/${nodeId}`, payload)
    } catch (err) {
      Object.assign(node, previous)
      console.error('Failed to update node:', err)
      throw err
    }
  }

  // --- ACTIONS: SENSOR MUTATIONS ---

  const addSensor = async (nodeId, { sensorId, customName, configValues }) => {
    const node = getNode(nodeId)
    if (!node) return

    const optimisticSensor = normalizeSensor({
      id: sensorId,
      name: customName || sensorId,
      display: customName || sensorId,
      status: 'pending',
      events24h: 0,
      isSilenced: false,
      osi: 'Sensor',
      icon: 'M12 12h0',
      lastHeartbeat: null,
    })

    node.installedSensors = node.installedSensors || []
    node.installedSensors.push(optimisticSensor)
    node.hasPendingConfig = true

    try {
      await api.post(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors`, {
        sensor_id: sensorId,
        custom_name: customName || sensorId,
        config_values: configValues,
      })
      fetchNodeDetails(nodeId)
    } catch (err) {
      const idx = node.installedSensors.findIndex(s => s.id === sensorId)
      if (idx !== -1) node.installedSensors.splice(idx, 1)
      node.hasPendingConfig = false
      console.error('Failed to add sensor:', err)
      throw err
    }
  }

  const updateSensor = async (nodeId, sensorId, { customName, configValues }) => {
    const node = getNode(nodeId)
    if (!node) return

    const sensor = getSensor(nodeId, sensorId)
    if (!sensor) return

    const previous = { ...sensor }
    const previousPending = node.hasPendingConfig

    sensor.display = customName || sensorId
    sensor.envVars = configValues
    node.hasPendingConfig = true

    try {
      await api.put(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors/${encodeURIComponent(sensorId)}`, {
        custom_name: customName || sensorId,
        config_values: configValues,
      })
      fetchNodeDetails(nodeId)
    } catch (err) {
      mergeSensor(sensor, previous)
      node.hasPendingConfig = previousPending
      console.error('Failed to update sensor:', err)
      throw err
    }
  }

  const removeSensor = async (nodeId, sensorId) => {
    if (!confirm('Remove this sensor? The node will be marked for deployment sync.')) return

    const node = getNode(nodeId)
    if (!node) return

    const sensorIdx = node.installedSensors?.findIndex(s => s.id === sensorId) ?? -1
    const previous = sensorIdx !== -1 ? { ...node.installedSensors[sensorIdx] } : null

    if (sensorIdx !== -1) node.installedSensors.splice(sensorIdx, 1)
    node.hasPendingConfig = true

    if (selectedSensor.value === sensorId) selectedSensor.value = null

    try {
      await api.delete(`/api/v1/nodes/${encodeURIComponent(nodeId)}/sensors/${encodeURIComponent(sensorId)}`)
    } catch (err) {
      if (previous && sensorIdx !== -1) {
        node.installedSensors.splice(sensorIdx, 0, previous)
        node.hasPendingConfig = false
      }
      console.error('Failed to remove sensor:', err)
      throw err
    }
  }

  const toggleSilence = async (nodeId, sensorId, targetState) => {
    const sensor = getSensor(nodeId, sensorId)
    if (!sensor) return

    const previous = sensor.isSilenced
    sensor.isSilenced = targetState

    try {
      await api.patch(`/api/v1/nodes/${nodeId}/sensors/${sensorId}/silence`, {
        is_silenced: targetState,
      })
    } catch (err) {
      sensor.isSilenced = previous
      console.error('Failed to toggle sensor silence:', err)
      throw err
    }
  }

  const silenceNode = async (nodeId) => {
    const node = getNode(nodeId)
    if (!node?.installedSensors?.length) return

    const allSilenced = node.installedSensors.every(s => s.isSilenced)
    const targetState = !allSilenced

    const previousStates = node.installedSensors.map(s => s.isSilenced)
    node.installedSensors.forEach(s => { s.isSilenced = targetState })

    try {
      await Promise.all(node.installedSensors.map(s =>
        api.patch(`/api/v1/nodes/${nodeId}/sensors/${s.id}/silence`, {
          is_silenced: targetState,
        })
      ))
    } catch (err) {
      node.installedSensors.forEach((s, i) => { s.isSilenced = previousStates[i] })
      console.error('Failed to silence node:', err)
      throw err
    }
  }

  // --- ACTIONS: COMPOSE / SYNC ---

  const fetchCompose = async (apiKey) => {
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
    if (!node?.apiKey) {
      throw new Error('Unable to sync: missing node API key')
    }

    const composeYaml = await fetchCompose(node.apiKey)
    node.hasPendingConfig = false
    fetchNodeDetails(nodeId)

    return composeYaml
  }

  // --- WEBSOCKET HANDLER ---

  const handleWsUpdate = (type, payload) => {
    if (type === 'SENSOR_HEARTBEAT') {
      const node = getNode(payload.node_id)
      if (node) {
        node.lastHeartbeat = payload.timestamp
        node.status = 'up'
        const sensor = getSensor(payload.node_id, payload.sensor_id)
        if (sensor) {
          sensor.lastHeartbeat = payload.timestamp
          sensor.status = 'up'
        }
      }
      return
    }

    if (type === 'SILENCE_SENSOR') {
      const sensor = getSensor(payload.node_id, payload.sensor_id)
      if (sensor) {
        sensor.isSilenced = payload.is_silenced ?? payload.isSilenced ?? sensor.isSilenced
      }
      return
    }

    if (type === 'NEW_SENSOR') {
      const node = getNode(payload.node_id)
      if (node && payload.sensor) {
        const sensorId = payload.sensor.id || payload.sensor.sensor_id
        const exists = getSensor(payload.node_id, sensorId)
        if (!exists) {
          node.installedSensors = node.installedSensors || []
          node.installedSensors.push(normalizeSensor(payload.sensor))
        }
        node.hasPendingConfig = true
      }
      return
    }

    if (type === 'DELETE_SENSOR') {
      const node = getNode(payload.node_id)
      if (node && payload.sensor_id) {
        const idx = node.installedSensors?.findIndex(s => s.id === payload.sensor_id) ?? -1
        if (idx !== -1) node.installedSensors.splice(idx, 1)
        node.hasPendingConfig = true
      }
      return
    }

    if (type === 'NODE_SYNCED') {
      const node = getNode(payload.node_id)
      if (node) {
        node.hasPendingConfig = false
        if (payload.active_revision) node.activeRevision = payload.active_revision
        if (payload.desired_revision !== undefined) node.desiredRevision = payload.desired_revision
        // Guarantee UI is 100% synced with the newly deployed sensors
        fetchNodeDetails(payload.node_id)
      }
      return
    }

    if (type === 'UPDATE_NODE') {
      if (payload.trigger_refresh) {
        fetchNodeDetails(payload.id)
        return
      }

      const normalized = normalizeNode(payload)
      const targetNode = getNode(normalized.id)
      if (targetNode) mergeNode(targetNode, normalized)
      return
    }

    if (type === 'DELETE_NODE') {
      const idx = nodes.value.findIndex(n => n.id === payload.node_id)
      if (idx !== -1) nodes.value.splice(idx, 1)
      if (selectedNode.value === payload.node_id) clearSelection()
      return
    }

    if (type === 'NEW_NODE') {
      const nodeId = payload.id || payload.node_id || payload.nodeId
      const exists = getNode(nodeId)
      if (!exists) {
        nodes.value.push(normalizeNode(payload))
      }
      if (nodeId) fetchNodeDetails(nodeId)
      return
    }

    if (type === 'NEW_EVENT') {
      if (payload.node_id && payload.sensor_id) {
        const node = getNode(payload.node_id)
        if (node) {
          const sensor = getSensor(payload.node_id, payload.sensor_id)
          if (sensor) sensor.events24h = (sensor.events24h || 0) + 1
          node.lastEvent = 'Just now'
        }
      } else {
        fetchFleet()
      }
      return
    }
  }

  return {
    nodes, uptimeData, selectedNode, selectedSensor, activeTimeframe,
    overallUptime, nodeMap, sensorIndex,
    getNode, getSensor,
    fetchFleet, fetchNodeDetails, fetchUptime, fetchManifests,
    selectTarget, clearSelection,
    createNode, deleteNode, updateNode,
    addSensor, updateSensor, removeSensor, toggleSilence, silenceNode,
    fetchCompose, generateCompose, syncNode,
    handleWsUpdate,
    normalizeSensor, normalizeNode,
  }
})