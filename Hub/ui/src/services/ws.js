/**
 * HoneyWireWS
 * * Standalone, production-ready WebSocket service for real-time event streaming.
 * Decoupled from Vue reactivity - uses an Event-Emitter pattern to pass data to stores.
 */
export class HoneyWireWS {
  constructor(baseUrl = window.location.origin) {
    this.baseUrl = baseUrl;
    this.ws = null;
    this.isDestroyed = false;
    
    // Reconnection State
    this.retryCount = 0;
    this.maxRetries = 15;
    this.baseRetryDelay = 2000;
    this.maxRetryDelay = 30000;
    this.reconnectTimeoutId = null;

    // Heartbeat State
    this.pingIntervalMs = 25000; // 25s (Common proxy timeout limit is 30s)
    this.pongTimeoutMs = 5000;    // Wait 5s for backend response
    this.pingIntervalId = null;
    this.pongTimeoutId = null;

    // Outbound Queue (if messages need to be sent during reconnection)
    this.sendQueue = [];

    // Map WebSocket message string codes to callback keys
    this.eventMapping = {
      'NEW_EVENT': 'onNewEvent',
      'SENSOR_HEARTBEAT': 'onSensorHeartbeat',
      'SYNC_CHARTS': 'onSyncCharts',
      'NEW_SENSOR': 'onNewSensor',
      'DELETE_SENSOR': 'onDeleteSensor',
      'SILENCE_SENSOR': 'onSilenceSensor',
      'NEW_NODE': 'onNewNode',
      'UPDATE_NODE': 'onUpdateNode',
      'DELETE_NODE': 'onDeleteNode',
      'NODE_SYNCED': 'onNodeSynced'
    };

    // Callback registry
    this.callbacks = {
      onReconnect: null,
      onDisconnect: null,
      onError: null,
      ...Object.values(this.eventMapping).reduce((acc, curr) => ({ ...acc, [curr]: null }), {})
    };
  }

  /**
   * Register a callback for a specific message or lifecycle event
   */
  on(eventType, callback) {
    if (Object.prototype.hasOwnProperty.call(this.callbacks, eventType)) {
      this.callbacks[eventType] = callback;
    } else {
      console.warn(`WebSocket: Registering fallback handler for unmapped hook: "${eventType}"`);
      this.callbacks[eventType] = callback;
    }
  }

  /**
   * Establish WebSocket connection with automatic handling
   */
  connect() {
    if (this.isDestroyed) return;

    // Clean up any stray timeouts/existing connections safely first
    this._clearReconnection();
    this._stopHeartbeat();
    if (this.ws) {
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onerror = null;
      this.ws.onclose = null;
      this.ws.close();
    }

    if (this.retryCount >= this.maxRetries) {
      console.error(`WebSocket: Max retries (${this.maxRetries}) reached. Stopping.`);
      if (this.callbacks.onError) this.callbacks.onError(new Error('Max retries reached'));
      return;
    }

    const protocol = this.baseUrl.startsWith('https') ? 'wss:' : 'ws:';
    const host = this.baseUrl.replace(/^https?:\/\//, '');
    const wsUrl = `${protocol}//${host}/api/v1/ws`;

    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      if (this.isDestroyed) {
        this.disconnect();
        return;
      }
      
      const wasReconnecting = this.retryCount > 0;
      this.retryCount = 0;
      
      this._startHeartbeat();
      this._flushQueue();

      if (wasReconnecting && this.callbacks.onReconnect) {
        this.callbacks.onReconnect();
      }
    };

    this.ws.onmessage = (event) => {
      this._handleMessage(event);
    };

    this.ws.onerror = (error) => {
      if (this.callbacks.onError) this.callbacks.onError(error);
    };

    this.ws.onclose = (event) => {
      this._stopHeartbeat();
      
      if (this.callbacks.onDisconnect) {
        this.callbacks.onDisconnect(event);
      }

      if (!this.isDestroyed) {
        this._scheduleReconnect();
      }
    };
  }

  /**
   * Send data out to the server safely, queuing if offline
   */
  send(type, payload = {}) {
    const messageStr = JSON.stringify({ type, payload });

    if (this.isConnected()) {
      this.ws.send(messageStr);
    } else if (!this.isDestroyed) {
      this.sendQueue.push(messageStr);
    }
  }

  /**
   * Check connection status safely
   */
  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN;
  }

  /**
   * Total Teardown
   */
  disconnect() {
    this.isDestroyed = true;
    this._clearReconnection();
    this._stopHeartbeat();
    this.sendQueue = [];

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  /**
   * Private Engine Methods
   */

  _handleMessage(event) {
    // Any message received (including your SYNC_CHARTS broadcast) resets our sanity timer
    this._startHeartbeat(); 

    try {
      const data = JSON.parse(event.data);
      const callbackName = this.eventMapping[data.type];
      if (callbackName && this.callbacks[callbackName]) {
        this.callbacks[callbackName](data.payload);
      }
    } catch (e) {
      // Gracefully catch frames that aren't JSON
    }
  }

  _scheduleReconnect() {
    if (this.reconnectTimeoutId) return; // Block stacked racing timeouts

    this.retryCount++;
    
    // Exponential Backoff calculation + Jitter (random variance between 0-1000ms)
    const backoffDelay = Math.min(
      this.baseRetryDelay * Math.pow(2, this.retryCount - 1), 
      this.maxRetryDelay
    );
    const jitter = Math.random() * 1000;
    const finalDelay = backoffDelay + jitter;

    this.reconnectTimeoutId = setTimeout(() => {
      this.reconnectTimeoutId = null;
      this.connect();
    }, finalDelay);
  }

  _clearReconnection() {
    if (this.reconnectTimeoutId) {
      clearTimeout(this.reconnectTimeoutId);
      this.reconnectTimeoutId = null;
    }
  }

_startHeartbeat() {
  this._stopHeartbeat();

  // If we don't hear a single word from the server for 35 seconds, 
  // assume the connection is dead and force-close it.
  this.pongTimeoutId = setTimeout(() => {
    console.warn('WebSocket: Silent connection detected. Terminating.');
    if (this.ws) this.ws.close();
  }, 35000); 
}

  _resetPongTimeout() {
    if (this.pongTimeoutId) {
      clearTimeout(this.pongTimeoutId);
      this.pongTimeoutId = null;
    }
  }

  _stopHeartbeat() {
    if (this.pingIntervalId) clearInterval(this.pingIntervalId);
    this._resetPongTimeout();
    this.pingIntervalId = null;
  }

  _flushQueue() {
    while (this.sendQueue.length > 0 && this.isConnected()) {
      const msg = this.sendQueue.shift();
      this.ws.send(msg);
    }
  }
}