import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

/**
 * Fleet Store (Infrastructure State)
 * 
 * Manages sensors, uptime data, and node/sensor selection.
 * 
 * CRITICAL: All array operations MUST use the composite key:
 * node_id AND sensor_id must be checked together.
 * Example: fleet.find(s => s.node_id === nodeId && s.sensor_id === sensorId)
 */

export const useFleetStore = defineStore('fleet', () => {
  // --- STATE ---
  const sensors = ref([])
  const uptimeData = ref([])
  const selectedNode = ref(null)
  const selectedSensor = ref(null)
  const activeTimeframe = ref('24H')

  // --- GETTERS ---
  /**
   * Calculate overall uptime percentage from all sensors' uptime blocks
   */
  const overallUptime = computed(() => {
    if (!uptimeData.value || uptimeData.value.length === 0) return '0.0%'

    let validBlocks = 0
    let upBlocks = 0

    uptimeData.value.forEach(sensor => {
      sensor.blocks.forEach(block => {
        if (block.status !== 'nodata') {
          validBlocks++
          if (block.status === 'up') {
            upBlocks += 1
          } else if (block.status === 'degraded') {
            upBlocks += 0.8
          }
        }
      })
    })

    if (validBlocks === 0) return '100.0%'
    return ((upBlocks / validBlocks) * 100).toFixed(1) + '%'
  })

  // --- ACTIONS ---

  /**
   * Fetch all sensors from the backend
   */
  const fetchFleet = async () => {
    try {
      const [sn] = await Promise.all([
        fetch('/api/v1/sensors').then(r => r.json()).catch(() => []),
      ])

      if (sn && sn.length) {
        sensors.value = sn.map(newSensor => {
          return { ...newSensor, is_silenced: !!newSensor.is_silenced }
        })
      }
    } catch (e) {
      console.error('Failed to fetch fleet data', e)
    }
  }

  /**
  * Fetch uptime data for the given timeframe
  * @param {string|null} timeframe - e.g., '24H', '7D', '30D'
  */
  const fetchUptime = async (timeframe) => {
    // If absolutely nothing is passed, fallback to the ref just in case
    const target = timeframe || activeTimeframe.value; 
    try {
        const data = await fetch(`/api/v1/uptime?timeframe=${target}`).then(r => r.json())
        uptimeData.value = data || []
    } catch (e) {
        console.error('Failed to fetch uptime', e)
    }
}

  /**
   * Select a target node and sensor (or deselect if already selected)
   * Composite key: both nodeId and sensorId must match for toggle
   * @param {string} nodeId
   * @param {string} sensorId
   */
  const selectTarget = (nodeId, sensorId = null) => {
    if (selectedNode.value === nodeId && selectedSensor.value === sensorId) {
        // 1. Exact match (Toggle OFF)
        selectedNode.value = null
        selectedSensor.value = null
    } else if (sensorId === null && selectedNode.value === nodeId && selectedSensor.value !== null) {
        // 2. Node clicked, but a sensor is currently selected: CLEAR the sensor, KEEP the node
        selectedSensor.value = null
    } else if (sensorId === null && selectedNode.value === nodeId && selectedSensor.value === null) {
        // 3. Node clicked, and ONLY the node is currently selected: CLEAR the node
        selectedNode.value = null
    } else {
        // 4. Set new selection
        selectedNode.value = nodeId
        selectedSensor.value = sensorId
    }
  }

  /**
   * Delete a specific sensor (Composite Key: nodeId + sensorId)
   * @param {string} nodeId
   * @param {string} sensorId
   */
  const forgetSensor = async (nodeId, sensorId) => {
    const originalSensors = [...sensors.value]
    const originalUptime = [...uptimeData.value]

    // COMPOSITE KEY: Filter by both node_id AND sensor_id
    sensors.value = sensors.value.filter(
      s => !(s.node_id === nodeId && s.sensor_id === sensorId)
    )
    uptimeData.value = uptimeData.value.filter(
      s => !(s.node_id === nodeId && s.id === sensorId)
    )

    // Clear selection if we just deleted the selected sensor
    if (selectedSensor.value === sensorId && selectedNode.value === nodeId) {
      selectedSensor.value = null
      selectedNode.value = null
    }

    try {
      const response = await fetch(
        `/api/v1/sensors/${sensorId}?node_id=${nodeId}`,
        { method: 'DELETE' }
      )
      if (!response.ok) throw new Error(`Server error: ${response.status}`)
    } catch (err) {
      console.error('Failed to forget sensor:', err)
      // Rollback on failure
      sensors.value = originalSensors
      uptimeData.value = originalUptime
      alert('Failed to remove sensor. Please try again.')
    }
  }

  /**
   * Delete an entire node and all its sensors (Composite Key filtering)
   * @param {string} nodeId
   */
  const forgetNode = async (nodeId) => {
    if (
      !confirm(
        `Are you sure you want to delete Node "${nodeId}" and ALL of its underlying sensors?`
      )
    ) {
      return
    }

    // COMPOSITE KEY: Find all sensors for this node
    const nodeSensors = sensors.value.filter(s => s.node_id === nodeId)

    for (const s of nodeSensors) {
      await forgetSensor(s.node_id, s.sensor_id)
    }

    if (selectedNode.value === nodeId) {
      selectedNode.value = null
      selectedSensor.value = null
    }
  }

  /**
   * Toggle the silence state of a sensor (Composite Key lookup)
   * @param {string} nodeId
   * @param {string} sensorId
   */
  const toggleSilence = async (nodeId, sensorId) => {
    // COMPOSITE KEY: Find by both node_id AND sensor_id
    const sensor = sensors.value.find(
      s => s.node_id === nodeId && s.sensor_id === sensorId
    )

    if (sensor) {
      const previousState = sensor.is_silenced
      sensor.is_silenced = !sensor.is_silenced

      try {
        const response = await fetch(
          `/api/v1/sensors/${sensorId}/silence`,
          {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              node_id: nodeId,
              is_silenced: sensor.is_silenced,
            }),
          }
        )
        if (!response.ok) throw new Error(`Server error: ${response.status}`)
      } catch (err) {
        sensor.is_silenced = previousState
        console.error('Failed to toggle sensor silence:', err)
        alert(
          `Failed to ${previousState ? 'unsilence' : 'silence'} sensor. Please try again.`
        )
      }
    }
  }

  /**
   * Toggle silence state for all sensors in a node (Composite Key filtering)
   * @param {string} nodeId
   */
  const silenceNode = async (nodeId) => {
    // COMPOSITE KEY: Get all sensors matching this node
    const nodeSensors = sensors.value.filter(s => s.node_id === nodeId)
    if (!nodeSensors.length) return

    const allSilenced = nodeSensors.every(s => s.is_silenced)
    const targetState = !allSilenced

    for (const s of nodeSensors) {
      if (s.is_silenced !== targetState) {
        await toggleSilence(s.node_id, s.sensor_id)
      }
    }
  }

  /**
   * Handle WebSocket updates: NEW_SENSOR, DELETE_SENSOR, SILENCE_SENSOR
   * Composite Key enforcement for all array modifications
   * @param {string} type - Message type
   * @param {object} payload - Message payload
   */
  const handleWsUpdate = (type, payload) => {
    if (type === 'NEW_SENSOR') {
      // New sensor: refetch to get full data
      fetchFleet()
      fetchUptime()
    } else if (type === 'DELETE_SENSOR') {
      // COMPOSITE KEY: Filter by both node_id AND sensor_id
      sensors.value = sensors.value.filter(
        s => !(s.node_id === payload.node_id && s.sensor_id === payload.sensor_id)
      )
      uptimeData.value = uptimeData.value.filter(
        s => !(s.node_id === payload.node_id && s.id === payload.sensor_id)
      )

      // Clear selection if deleted sensor was selected
      if (
        selectedSensor.value === payload.sensor_id &&
        selectedNode.value === payload.node_id
      ) {
        selectedSensor.value = null
        selectedNode.value = null
      }
    } else if (type === 'SILENCE_SENSOR') {
      // COMPOSITE KEY: Find by both node_id AND sensor_id
      const sensor = sensors.value.find(
        s =>
          s.node_id === payload.node_id && s.sensor_id === payload.sensor_id
      )
      if (sensor) {
        sensor.is_silenced = payload.is_silenced
      }
    } else if (type === 'SENSOR_HEARTBEAT') {
      // Update the main sensors array
      const sensor = sensors.value.find(
        s => s.node_id === payload.node_id && s.sensor_id === payload.sensor_id
      )
      if (sensor) {
        sensor.last_seen = payload.timestamp || new Date().toISOString()
        sensor.status = 'online'
      }

      // Find the same sensor in the uptimeData array and force it to Online
      const uptimeSensor = uptimeData.value.find(
        s => s.node_id === payload.node_id && s.id === payload.sensor_id
      )
      if (uptimeSensor) {
        uptimeSensor.isOnline = true
        
        if (uptimeSensor.blocks && uptimeSensor.blocks.length > 0) {
            uptimeSensor.blocks[uptimeSensor.blocks.length - 1].status = 'up'
            uptimeSensor.blocks[uptimeSensor.blocks.length - 1].label = 'Online (Live)'
        }
      }
    } 
  }

  return {
    // State
    sensors,
    uptimeData,
    selectedNode,
    selectedSensor,
    activeTimeframe,
    // Getters
    overallUptime,
    // Actions
    fetchFleet,
    fetchUptime,
    selectTarget,
    forgetSensor,
    forgetNode,
    toggleSilence,
    silenceNode,
    handleWsUpdate,
  }
})
    