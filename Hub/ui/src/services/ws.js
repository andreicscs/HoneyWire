/**
 * HoneyWireWS
 * 
 * Standalone WebSocket service for real-time event streaming.
 * Decoupled from Vue reactivity - uses callbacks/event emitters to pass data to stores.
 * 
 * Features:
 * - Automatic reconnection with exponential backoff
 * - JSON message parsing
 * - Callback-based event dispatch
 * - No direct Vue ref imports
 */

export class HoneyWireWS {
  constructor(baseUrl = window.location.origin) {
    this.baseUrl = baseUrl
    this.ws = null
    this.isDestroyed = false
    this.retryCount = 0
    this.maxRetries = 10
    this.retryDelay = 3000
    this.healthCheckInterval = null

    // Event callbacks: will be set by the caller
    this.callbacks = {
      onNewEvent: null,
      onSensorHeartbeat: null,
      onNewSensor: null,
      onDeleteSensor: null,
      onSilenceSensor: null,
      onReconnect: null,
      onSyncCharts: null,
      onNewNode: null,
      onUpdateNode: null,
      onDeleteNode: null,
      onNodeSynced: null,
    }
  }

  /**
   * Register a callback for a specific message type
   * @param {string} eventType - e.g., 'onNewEvent', 'onSensorHeartbeat'
   * @param {Function} callback - Function to call with (payload)
   */
  on(eventType, callback) {
    if (this.callbacks.hasOwnProperty(eventType)) {
      this.callbacks[eventType] = callback
    }
  }

  /**
   * Establish WebSocket connection with automatic reconnect
   */
  connect() {
    if (this.isDestroyed) {
      console.warn('WebSocket: Cannot connect after destroy()')
      return
    }

    if (this.retryCount >= this.maxRetries) {
      console.error(
        `WebSocket: Max retries (${this.maxRetries}) reached, stopping reconnection attempts`
      )
      return
    }

    const protocol = this.baseUrl.startsWith('https') ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws`

    console.log(`WebSocket: Connecting to ${wsUrl}`)

    this.ws = new WebSocket(wsUrl)

    this.ws.onopen = () => {
      console.log('WebSocket: Connected')
      
      // If retryCount > 0, it means we dropped and just successfully reconnected
      if (this.retryCount > 0 && this.callbacks.onReconnect) {
        this.callbacks.onReconnect()
      }
      
      this.retryCount = 0
      this.retryDelay = 3000
    }

    this.ws.onmessage = (event) => {
      this._handleMessage(event)
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket: Error', error)
    }

    this.ws.onclose = () => {
      console.warn('WebSocket: Disconnected')
      if (!this.isDestroyed) {
        this._scheduleReconnect()
      }
    }
  }

  /**
   * Parse incoming message and dispatch to registered callbacks
   * @private
   */
  _handleMessage(event) {
    try {
      const data = JSON.parse(event.data)
      console.log('WebSocket: Message', data.type, data.payload)

      switch (data.type) {
        case 'NEW_EVENT':
          if (this.callbacks.onNewEvent) {
            this.callbacks.onNewEvent(data.payload)
          }
          break

        case 'SENSOR_HEARTBEAT':
          if (this.callbacks.onSensorHeartbeat) {
            this.callbacks.onSensorHeartbeat(data.payload)
          }
          break
        
        case 'SYNC_CHARTS':
         if (this.callbacks.onSyncCharts) this.callbacks.onSyncCharts()
         break

        case 'NEW_SENSOR':
          if (this.callbacks.onNewSensor) {
            this.callbacks.onNewSensor(data.payload)
          }
          break

        case 'DELETE_SENSOR':
          if (this.callbacks.onDeleteSensor) {
            this.callbacks.onDeleteSensor(data.payload)
          }
          break

        case 'SILENCE_SENSOR':
          if (this.callbacks.onSilenceSensor) {
            this.callbacks.onSilenceSensor(data.payload)
          }
          break
        case 'NEW_NODE':
          if (this.callbacks.onNewNode) {
            this.callbacks.onNewNode(data.payload)
          }
          break

        case 'UPDATE_NODE':
          if (this.callbacks.onUpdateNode) {
            this.callbacks.onUpdateNode(data.payload)
          }
          break

        case 'DELETE_NODE':
          if (this.callbacks.onDeleteNode) {
            this.callbacks.onDeleteNode(data.payload)
          }
          break
        case 'NODE_SYNCED':
          if (this.callbacks.onNodeSynced) {
            this.callbacks.onNodeSynced(data.payload)
          }
          break

        default:
          console.warn(`WebSocket: Unknown message type: ${data.type}`)
      }
    } catch (error) {
      console.error('WebSocket: Parse error', error, event.data)
    }
  }

  /**
   * Schedule a reconnection attempt with exponential backoff
   * @private
   */
  _scheduleReconnect() {
    this.retryCount++
    console.warn(
      `WebSocket: Retry ${this.retryCount}/${this.maxRetries} in ${this.retryDelay}ms`
    )

    setTimeout(() => {
      this.connect()
    }, this.retryDelay)

    // Exponential backoff: double the delay, capped at 30s
    this.retryDelay = Math.min(this.retryDelay * 2, 30000)
  }

  /**
   * Close the WebSocket and clean up
   */
  disconnect() {
    this.isDestroyed = true
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval)
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  /**
   * Check the current connection state
   * @returns {boolean}
   */
  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN
  }
}
