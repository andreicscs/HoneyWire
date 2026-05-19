import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useFleetStore = defineStore('fleet', () => {
  // --- STATE ---
  const nodes = ref([]) // V2: Sensors are nested inside nodes!
  const uptimeData = ref([])
  const selectedNode = ref(null)
  const selectedSensor = ref(null)
  const activeTimeframe = ref('24H')

  // --- GETTERS ---
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
    return validBlocks === 0 ? '100.0%' : ((upBlocks / validBlocks) * 100).toFixed(1) + '%'
  })

  // --- ACTIONS ---
  const fetchFleet = async () => {
    try {
      const res = await fetch('/api/v1/nodes')
      if (res.ok) {
        nodes.value = await res.json() || []
      }
    } catch (e) {
      console.error('Failed to fetch fleet data', e)
    }
  }

  const fetchUptime = async (timeframe) => {
    const target = timeframe || activeTimeframe.value; 
    try {
        const res = await fetch(`/api/v1/uptime?timeframe=${target}`)
        if (res.ok) uptimeData.value = await res.json() || []
    } catch (e) {
        console.error('Failed to fetch uptime', e)
    }
  }

  const selectTarget = (nodeId, sensorId = null) => {
    if (selectedNode.value === nodeId && selectedSensor.value === sensorId) {
        selectedNode.value = null
        selectedSensor.value = null
    } else if (sensorId === null && selectedNode.value === nodeId && selectedSensor.value !== null) {
        selectedSensor.value = null
    } else if (sensorId === null && selectedNode.value === nodeId && selectedSensor.value === null) {
        selectedNode.value = null
    } else {
        selectedNode.value = nodeId
        selectedSensor.value = sensorId
    }
  }

  const deleteNode = async (nodeId) => {
    if (!confirm(`Delete Node "${nodeId}" and ALL of its underlying sensors?`)) return
    try {
      const res = await fetch(`/api/v1/nodes/${nodeId}`, { method: 'DELETE' })
      if (!res.ok) throw new Error('Delete failed')
      
      if (selectedNode.value === nodeId) {
        selectedNode.value = null
        selectedSensor.value = null
      }
      await fetchFleet()
    } catch (err) {
      console.error('Failed to delete node:', err)
    }
  }

  const deleteSensor = async (nodeId, sensorId) => {
    if (!confirm('Remove this sensor? The node will be marked for deployment sync.')) return
    try {
      const res = await fetch(`/api/v1/nodes/${nodeId}/sensors/${sensorId}`, { method: 'DELETE' })
      if (!res.ok) throw new Error('Delete failed')
      
      if (selectedSensor.value === sensorId) selectedSensor.value = null
      await fetchFleet()
    } catch (err) {
      console.error('Failed to delete sensor:', err)
    }
  }

  const toggleSilence = async (nodeId, sensorId, targetState) => {
    try {
      const res = await fetch(`/api/v1/nodes/${nodeId}/sensors/${sensorId}/silence`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_silenced: targetState }),
      })
      if (!res.ok) throw new Error('Silence toggle failed')
      await fetchFleet()
    } catch (err) {
      console.error('Failed to toggle sensor silence:', err)
    }
  }

  const silenceNode = async (nodeId) => {
    const node = nodes.value.find(n => n.id === nodeId)
    if (!node || !node.installedSensors || !node.installedSensors.length) return

    const allSilenced = node.installedSensors.every(s => s.isSilenced)
    const targetState = !allSilenced

    try {
      await Promise.all(node.installedSensors.map(s => 
          fetch(`/api/v1/nodes/${nodeId}/sensors/${s.id}/silence`, {
              method: 'PATCH',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ is_silenced: targetState })
          })
      ))
      await fetchFleet()
    } catch (err) {
      console.error("Failed to silence node:", err)
    }
  }

  const handleWsUpdate = (type, payload) => {
    // For V2, the safest and cleanest way to handle state changes (since sensors are nested)
    // is to just refetch the fleet on critical events, or do targeted updates.
    if (['NODE_SYNCED', 'UPDATE_NODE', 'DELETE_NODE', 'SILENCE_SENSOR'].includes(type)) {
      fetchFleet()
    } else if (type === 'NEW_EVENT') {
      fetchFleet() // Refetches to update the 24h event counts
    } else if (type === 'SENSOR_HEARTBEAT') {
      const node = nodes.value.find(n => n.id === payload.node_id)
      if (node) {
        node.lastHeartbeat = payload.timestamp
        node.status = 'up'
        const sensor = (node.installedSensors || []).find(s => s.id === payload.sensor_id)
        if (sensor) {
          sensor.lastHeartbeat = payload.timestamp
          sensor.status = 'up'
        }
      }
    }
  }

  const updateNode = async (nodeId, payload) => {
    try {
      const res = await fetch(`/api/v1/nodes/${nodeId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      })
      if (!res.ok) throw new Error('Failed to update node')
      
      // Sync state from backend to ensure UI matches reality
      await fetchFleet() 
    } catch (err) {
      console.error('Failed to update node:', err)
      throw err // Throw back to component for UI rollback
    }
  }

  return {
    nodes, uptimeData, selectedNode, selectedSensor, activeTimeframe,
    overallUptime,
    fetchFleet, fetchUptime, selectTarget, deleteNode, deleteSensor, toggleSilence, silenceNode, handleWsUpdate,
    updateNode, 
  }
})