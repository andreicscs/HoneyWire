# HoneyWire Frontend Architecture & Data Flow

This document explains the structural design, state management, data flow, and real-time update strategy of the HoneyWire frontend. It focuses on how data is stored, updated, rendered, and the critical distinction between WebSocket (realtime) and API (authoritative) data sources.

For practical development guidelines, project structure, and component rules, see [Frontend Developer Guide](./Frontend.md).

---

# Table of Contents

1. [Layered Architecture](#layered-architecture)
2. [Core Principles](#core-principles)
3. [State Storage](#state-storage)
4. [Data Flow Lifecycle](#data-flow-lifecycle)
5. [API Data vs WebSocket Data](#api-data-vs-websocket-data)
6. [Normalization & Reactivity](#normalization--reactivity)
7. [WebSocket Integration](#websocket-integration)
8. [Bootstrap & Lifecycle](#bootstrap--lifecycle)
9. [Error Handling & Rollback](#error-handling--rollback)
10. [Debugging Guide](#debugging-guide)

---

# Layered Architecture

HoneyWire enforces strict layered architecture with unidirectional dependencies:

```
┌─────────────────────────────────────────┐
│  Views & Components                     │
│  (UI rendering + ephemeral state)       │
├─────────────────────────────────────────┤
│  Stores (Pinia)                         │
│  (Business logic + state ownership)     │
├─────────────────────────────────────────┤
│  API Client · WebSocket Service         │
│  (Transport + error handling)           │
├─────────────────────────────────────────┤
│  Utils & Helpers                        │
│  (Shared functions)                     │
└─────────────────────────────────────────┘
```

**Critical Rule:** No layer reaches above itself.
- Views **never** call APIs directly
- Stores **never** import Vue or manage UI state
- Services **never** touch component state
- API Client **never** implements business logic

---

# Core Principles

1. **Deterministic rendering** — Same state always produces same output
2. **Centralized state ownership** — Stores are the single source of truth
3. **Optimistic responsiveness** — UI updates immediately, backend confirms asynchronously
4. **Rollback safety** — Every mutation has a clear undo path
5. **Reactive identity stability** — Array/object references preserved, never reassigned
6. **Normalized boundaries** — Data normalized once at store entry point, never in components
7. **Transport abstraction** — API layer decoupled from business logic
8. **Predictable data flow** — One direction, one owner per piece of state

---

# State Storage

HoneyWire uses three main Pinia stores, each owning a distinct domain:

## Store: `app.js` — Application & Auth State

**Ownership:** UI navigation, authentication, system state

| State | Purpose | Source |
|-------|---------|--------|
| `isAuthenticated` | Shell reveal toggle | Set by App.vue after `loadAppData()` completes |
| `requiresSetup` | Initial setup flow gate | API: `GET /api/v1/setup/status` |
| `currentView` | Active page (dashboard, fleet, settings, etc.) | UI selection |
| `sidebarOpen` | Sidebar visibility toggle | UI toggle |
| `viewingArchive` | Archive view mode (vs active events) | UI toggle |
| `isArmed` | System armed/disarmed state | API: `GET /api/v1/system/state` + WS updates |
| `version` | Hub version string | API: `GET /api/v1/version` |
| `activeTimeframe` | Dashboard chart timeframe (24H, 7D, 30D, 1H) | UI selection |
| `velocityTimeframe` | Threat velocity timeframe | UI selection |

**Actions:**
- `login(password)` — Authenticate user (returns success, does NOT set `isAuthenticated`)
- `logout()` — Invalidate session
- `completeSetup()` — Store setup credentials
- `checkSetupStatus()` — Fetch initial system state
- `checkRequiresSetup()` — Determine if hub needs setup
- `checkSystemState()` — Check if authenticated
- `toggleArmed()` — Toggle system armed state (optimistic update + rollback)

---

## Store: `fleet.js` — Infrastructure State

**Ownership:** Nodes, sensors, uptime, deployment metadata

### State Structure

```javascript
{
  nodes: [
    {
      id: "node-abc",
      alias: "production-db",
      tags: ["database", "prod"],
      status: "up" | "down" | "unknown" | "pending",
      publicIp: "203.0.113.5",
      privateIp: "10.0.1.5",
      lastEvent: "2h ago",
      lastHeartbeat: 1716345600000,
      hasPendingConfig: false,
      activeRevision: "rev_123",
      desiredRevision: "rev_124",
      installedSensors: [
        {
          id: "tcp-tarpit-1",
          name: "TCP Tarpit",
          display: "Custom TCP Tarpit",
          status: "up",
          isSilenced: false,
          events24h: 3,
          osi: "Layer 4",
          lastHeartbeat: 1716345600000,
          envVars: { HW_SEVERITY: "high" },
          metadata: { ... }
        },
        // ... more sensors
      ]
    },
    // ... more nodes
  ],
  
  uptimeData: [
    {
      sensor_id: "tcp-tarpit-1",
      blocks: [
        { timestamp: 1716259200000, status: "up" | "down" | "degraded" | "nodata" },
        // ... 24H of blocks
      ]
    },
    // ... more sensors
  ],
  
  selectedNode: "node-abc" | null,
  selectedSensor: "tcp-tarpit-1" | null,
  activeTimeframe: "24H" | "7D" | "30D" | "1H"
}
```

### Computed Properties (Indexed Access)

For O(1) lookup performance, the store maintains computed maps:

```javascript
// nodeMap: { [nodeId]: node }
const getNode = (nodeId) => nodeMap.value[nodeId] || null

// sensorIndex: { [nodeId]: { [sensorId]: sensor } }
const getSensor = (nodeId, sensorId) => sensorIndex.value[nodeId]?.[sensorId] || null

// Example: Get sensor data
const sensor = fleet.getSensor("node-abc", "tcp-tarpit-1")
```

**Critical:** Always use composite key `node_id + sensor_id`. Sensor IDs are only unique within a node.

### Data Fetch Actions

| Action | Endpoint | Purpose | When Called |
|--------|----------|---------|-------------|
| `fetchFleet()` | `GET /api/v1/nodes` | Load all nodes & sensors | Cold boot, manual refresh, WS reconnect |
| `fetchNodeDetails(nodeId)` | `GET /api/v1/nodes/{id}` | Load single node details | After add/remove sensor, manual refresh |
| `fetchUptime(timeframe)` | `GET /api/v1/uptime` | Load uptime blocks | Cold boot, timeframe change, WS `SYNC_CHARTS` |
| `fetchManifests()` | `GET /api/v1/manifests` | Load sensor catalog | Store view load |

### Data Mutation Actions

| Action | Endpoint | Purpose | Optimistic Update |
|--------|----------|---------|-------------------|
| `createNode(alias, tags)` | `POST /api/v1/nodes` | Create new node | Add partial node immediately |
| `updateNode(nodeId, payload)` | `PATCH /api/v1/nodes/{id}` | Update node metadata | Apply changes immediately |
| `deleteNode(nodeId)` | `DELETE /api/v1/nodes/{id}` | Delete node | Remove immediately, refetch on error |
| `addSensor(nodeId, sensorConfig)` | `POST /api/v1/nodes/{id}/sensors` | Deploy sensor | Add optimistic sensor, refetch details |
| `updateSensor(nodeId, sensorId, config)` | `PUT /api/v1/nodes/{id}/sensors/{sensorId}` | Update sensor config | Apply changes immediately |
| `removeSensor(nodeId, sensorId)` | `DELETE /api/v1/nodes/{id}/sensors/{sensorId}` | Remove sensor | Remove immediately, refetch on error |
| `toggleSilence(nodeId, sensorId, state)` | `PATCH /api/v1/nodes/{id}/sensors/{sensorId}/silence` | Silence sensor | Toggle immediately, rollback on error |

---

## Store: `events.js` — Telemetry State

**Ownership:** Intrusion events, event filtering, unread tracking

### State Structure

```javascript
{
  events: [
    {
      id: "event-xyz",
      node_id: "node-abc",
      sensor_id: "tcp-tarpit-1",
      source: "203.0.113.99",
      target: "Auth Gateway",
      severity: "critical" | "high" | "medium" | "low" | "info",
      event_trigger: "malformed_jwt_detected",
      is_read: 0 | 1,
      is_archived: 0 | 1,
      timestamp: 1716345600000,
      details: { protocol: "TCP", action_taken: "logged" }
    },
    // ... more events
  ],
  
  unreadCount: 5,
  activeEvent: null,
  isFetching: false
}
```

### Computed Properties

```javascript
// Filtered events based on current selections and view mode
filteredEvents = computed(() => {
  // Filter by archive state (viewingArchive from app store)
  // Filter by selectedNode (if selected)
  // Filter by selectedSensor (if selected)
  return events.value.filter(...)
})
```

### Event Actions

| Action | Endpoint | Purpose |
|--------|----------|---------|
| `fetchEvents(archived, nodeId, sensorId)` | `GET /api/v1/events` | Fetch events with filters |
| `markEventRead(eventId)` | `PATCH /api/v1/events/{id}/read` | Mark single event read |
| `markAllRead()` | `PATCH /api/v1/events/read` | Mark all events read |
| `archiveEvent(eventId)` | `PATCH /api/v1/events/{id}/archive` | Archive single event |
| `archiveAll()` | `PATCH /api/v1/events/archive-all` | Archive all active events |
| `handleWsEvent(payload)` | (WebSocket) | Apply incoming event from WS |

---

# Data Flow Lifecycle

## 1. User Action → Store → API → Backend

**Example: User toggles silence on a sensor**

```
View clicks: "Silence Sensor"
    ↓
View calls: fleetStore.toggleSilence(nodeId, sensorId, true)
    ↓
Store Action starts:
  1. Save previous state: const previous = sensor.isSilenced
  2. OPTIMISTIC: sensor.isSilenced = true  ← UI updates immediately
  3. Await API: api.patch(`/api/v1/nodes/${nodeId}/sensors/${sensorId}/silence`, { is_silenced: true })
    ↓
Backend processes request
    ↓
Success (2xx):
  - API client returns resolved promise
  - Store does nothing (UI already updated)
    ↓
Error (4xx/5xx):
  - API client throws ApiError
  - Store catches and ROLLBACK: sensor.isSilenced = previous
  - View receives error and shows toast notification
```

**Key Pattern:** Optimistic first, confirm async, rollback on error.

---

## 2. API Fetch → Store → Normalize → Merge → UI Update

**Example: Fetch fleet on cold boot**

```
App.vue calls: await fleetStore.fetchFleet()
    ↓
Store Action:
  1. API: const res = await api.get('/api/v1/nodes')
  2. Deserialize: const raw = await res.json()  [array of raw node objects]
    ↓
  3. NORMALIZE: raw.map(normalizeNode)
     - raw.last_heartbeat → lastHeartbeat
     - raw.is_silenced → isSilenced
     - raw.installed_sensors → installedSensors
     - Recursively normalize sensors
    ↓
  4. MERGE with existing:
     - For each incoming node:
       - If exists: mergeNode(existing, incoming)
         * Update sensor array: splice/push, never reassign
         * Update fields: Object.assign()
       - If new: push to nodes.value
     - Remove nodes not in incoming (deleted on backend)
    ↓
  5. Vue reactivity triggered:
     - Watchers on nodes.value fire
     - Computed properties recompute
    ↓
UI re-renders with new data
```

**Key Pattern:** Normalize at boundary, preserve array identity, merge existing to prevent watchers breaking.

---

## 3. WebSocket Event → Service → Store Handler → UI Update

**Example: Backend broadcasts NEW_SENSOR event**

```
Backend: Node deployed a sensor
    ↓
WS broadcast: { type: "NEW_SENSOR", payload: { node_id: "...", sensor: {...} } }
    ↓
Service (ws.js) receives message
    ↓
_handleMessage() parses JSON, routes by type
    ↓
Callback dispatch: this.callbacks.onNewSensor(payload)
    ↓
App.vue registered handler:
  wsService.on('onNewSensor', (payload) => fleetStore.handleWsUpdate('NEW_SENSOR', payload))
    ↓
Store.handleWsUpdate('NEW_SENSOR', payload):
  1. Get node: const node = getNode(payload.node_id)
  2. Check if sensor exists: const exists = getSensor(payload.node_id, payload.sensor.id)
  3. If not exists:
     - NORMALIZE sensor: normalizeSensor(payload.sensor)
     - PUSH to node's array: node.installedSensors.push(normalized)
     - Mark as pending: node.hasPendingConfig = true
  4. Optionally refetch details: fetchNodeDetails(payload.node_id)
    ↓
Vue reactivity triggered:
  - Component watching installedSensors sees change
  - Component re-renders
```

**Key Pattern:** WebSocket updates are applied immediately (no rollback), optionally trigger full refetch for authoritative state.

---

# API Data vs WebSocket Data

This is the **critical distinction** between the two data sources:

## API Data (Authoritative)

**Characteristics:**
- **Source of truth** — represents backend state at fetch time
- **Complete** — includes all fields and nested data
- **Normalized** — consistent key naming (last_heartbeat, installed_sensors)
- **Explicit** — full payload must be fetched

**When used:**
- Cold boot (load initial state)
- Manual refresh (user clicks "Refresh")
- WS reconnect (recover missed updates)
- Critical mutations (create/delete nodes, deploy sensors)

**Example:**
```javascript
// Full node state fetched from backend
const res = await api.get('/api/v1/nodes')
// Returns: [
//   {
//     id: "node-1",
//     alias: "production-db",
//     installed_sensors: [ { id: "sensor-1", ... }, ... ],
//     last_heartbeat: 1716345600000,
//     ...
//   }
// ]
```

---

## WebSocket Data (Realtime Delta)

**Characteristics:**
- **Incremental** — only includes changed fields
- **Immediate** — arrives within milliseconds
- **Payload-efficient** — minimal serialization
- **Event-driven** — type-specific updates

**When used:**
- Sensor heartbeat (update last_heartbeat)
- New event detected (append to events array)
- Configuration applied (mark as synced)
- Sensor added/removed (update sensor list)

**Example:**
```javascript
// Heartbeat update from WebSocket (minimal payload)
// Type: SENSOR_HEARTBEAT
// Payload: {
//   node_id: "node-1",
//   sensor_id: "sensor-1",
//   timestamp: 1716345602000,
//   status: "up"
// }

// Store handler applies immediately:
const sensor = getSensor(payload.node_id, payload.sensor_id)
sensor.lastHeartbeat = payload.timestamp
sensor.status = payload.status  // UI updates instantly
```

---

## Data Update Strategy

| Scenario | API | WS | Behavior |
|----------|-----|----|----|
| **Cold boot** | ✓ | — | Full fetch from API |
| **Sensor heartbeat arrives** | — | ✓ | Immediate update (low latency) |
| **User adds sensor** | ✓ | — | Optimistic add, fetch full details |
| **WS reconnects** | ✓ | — | Full refetch (catch missed updates) |
| **New event detected** | — | ✓ | Prepend to events (immediate) |
| **Node synced** | — | ✓ | Mark as synced, fetch details |

**Priority Rule:** WebSocket updates are applied immediately; API fetches verify and correct state.

---

# Normalization & Reactivity

## Data Normalization

All backend payloads are normalized at the store boundary using `normalize*` functions. Components **never** normalize.

### Example: `normalizeNode(raw)`

```javascript
const normalizeNode = (raw) => ({
  id: raw.id || raw.node_id || raw.nodeId,                    // ID normalization
  alias: raw.alias || raw.name || 'Unnamed Node',             // Friendly name
  status: raw.status || 'unknown',
  publicIp: raw.publicIp || raw.public_ip || null,            // snake_case → camelCase
  privateIp: raw.privateIp || raw.private_ip || null,
  tags: raw.tags || [],
  apiKey: raw.apiKey || raw.api_key || null,
  hasPendingConfig: raw.hasPendingConfig ?? raw.pending_config ?? false,
  activeRevision: raw.activeRevision || raw.active_revision || '',
  desiredRevision: raw.desiredRevision || raw.desired_revision || '',
  lastEvent: raw.lastEvent || raw.last_event || 'Never',
  lastHeartbeat: raw.lastHeartbeat || raw.last_heartbeat || null,
  installedSensors: (raw.installedSensors || raw.installed_sensors || [])
    .map(normalizeSensor)  // Recursively normalize nested arrays
})
```

**Benefits:**
- Backend schema changes absorbed at boundary
- Components work with consistent frontend schema
- Easy to handle multiple API versions
- Fallback values prevent undefined errors

---

## Reactive Identity Preservation

Vue 3 reactivity depends on object identity. If you break the identity, watchers and computed properties fail.

### ❌ WRONG: Reassignment breaks identity

```javascript
// This breaks Vue reactivity!
nodes.value = newArray.map(normalizeNode)  // New reference = broken watchers
```

### ✅ CORRECT: Mutation preserves identity

```javascript
// Method 1: splice + push (for array updates)
nodes.value.splice(0, nodes.value.length)  // Clear without reassigning
incoming.forEach(node => nodes.value.push(node))

// Method 2: Object.assign (for object updates)
const existing = nodes.value[0]
Object.assign(existing, { alias: "new alias" })  // Mutate in-place

// Method 3: Preserve during merge
mergeNode(existing, incoming) {
  // Update sensor array safely
  const incomingSensors = incoming.installedSensors || []
  if (!existing.installedSensors) existing.installedSensors = []
  
  // Instead of: existing.installedSensors = incomingSensors
  // Do: Update contents while preserving array reference
  const existingSensors = existing.installedSensors
  
  // Add/update from incoming
  incomingSensors.forEach(newSensor => {
    const idx = existingSensors.findIndex(s => s.id === newSensor.id)
    if (idx !== -1) Object.assign(existingSensors[idx], newSensor)
    else existingSensors.push(newSensor)
  })
  
  // Remove deleted from incoming
  for (let i = existingSensors.length - 1; i >= 0; i--) {
    if (!incomingSensors.find(s => s.id === existingSensors[i].id)) {
      existingSensors.splice(i, 1)
    }
  }
}
```

**Rule:** Always mutate arrays/objects in-place. Never reassign. Use `splice()`, `push()`, `Object.assign()`.

---

# WebSocket Integration

## Architecture

The WebSocket layer is completely decoupled from Vue/Pinia:

```
┌─────────────────────┐
│  HoneyWireWS        │  ← Framework-agnostic service
│  (services/ws.js)   │     No Vue imports, callback-based
└──────────┬──────────┘
           │
           │ callbacks
           ↓
     ┌─────────────┐
     │  App.vue    │  ← Orchestrator (registers handlers)
     └──────┬──────┘
            │ routes to
            ↓
     ┌─────────────────┐
     │  Stores         │  ← Business logic
     │  (fleet, events)│
     └─────────────────┘
            │
            ↓
     ┌─────────────────┐
     │  Components     │  ← UI rendering
     └─────────────────┘
```

### Service Layer: `HoneyWireWS` class

**Responsibilities:**
- Establish WebSocket connection
- Auto-reconnect with exponential backoff
- Parse incoming JSON messages
- Dispatch to registered callbacks
- No state management

**Key Methods:**

```javascript
const wsService = new HoneyWireWS()

// Register handlers before connecting
wsService.on('onNewEvent', (payload) => eventsStore.handleWsEvent(payload))
wsService.on('onSensorHeartbeat', (payload) => fleetStore.handleWsUpdate('SENSOR_HEARTBEAT', payload))
wsService.on('onReconnect', async () => {
  // Full data refetch on reconnect
  await Promise.all([
    fleetStore.fetchFleet(),
    fleetStore.fetchUptime(fleetStore.activeTimeframe),
    eventsStore.fetchEvents(),
  ])
})

// Connect
wsService.connect()
```

### Message Types

| Type | Payload | Handler | Purpose |
|------|---------|---------|---------|
| `NEW_EVENT` | `{ node_id, sensor_id, source, ... }` | `eventsStore.handleWsEvent()` | New intrusion detected |
| `SENSOR_HEARTBEAT` | `{ node_id, sensor_id, timestamp, status }` | `fleetStore.handleWsUpdate()` | Sensor alive, update status |
| `NEW_SENSOR` | `{ node_id, sensor: {...} }` | `fleetStore.handleWsUpdate()` | Sensor deployed |
| `DELETE_SENSOR` | `{ node_id, sensor_id }` | `fleetStore.handleWsUpdate()` | Sensor removed |
| `SILENCE_SENSOR` | `{ node_id, sensor_id, is_silenced }` | `fleetStore.handleWsUpdate()` | Sensor silenced/unsilenced |
| `NEW_NODE` | `{ id, alias, ... }` | `fleetStore.handleWsUpdate()` | Node created |
| `UPDATE_NODE` | `{ id, ...updates }` | `fleetStore.handleWsUpdate()` | Node metadata changed |
| `DELETE_NODE` | `{ node_id }` | `fleetStore.handleWsUpdate()` | Node deleted |
| `NODE_SYNCED` | `{ node_id, active_revision }` | `fleetStore.handleWsUpdate()` | Config deployed |
| `SYNC_CHARTS` | `{}` | `fleetStore.fetchUptime()` | Refetch uptime charts |

### Auto-Reconnect Strategy

```javascript
// Connection established
wsService.connect()  // Connects to /api/v1/ws

// Connection drops
// Automatic reconnect with exponential backoff:
// Retry 1: 3s delay
// Retry 2: 6s delay
// Retry 3: 12s delay
// ... up to 30s max delay
// Max 10 retries total

// On successful reconnect:
// 1. wsService.onReconnect fires
// 2. App.vue handler triggers full data refetch
// 3. Any missed events/updates recovered
// 4. UI synchronized with backend state
```

---

# Bootstrap & Lifecycle

## Cold Boot Sequence

User loads the app (`onMounted` in App.vue):

```javascript
onMounted(() => {
  checkAuthAndInit()
})

const checkAuthAndInit = async () => {
  try {
    // 1. Check if setup is required
    const needsSetup = await appStore.checkRequiresSetup()
    if (needsSetup) {
      appStore.requiresSetup = true  // Show Setup view
      return
    }

    // 2. Check if authenticated (verify session cookie)
    const authenticated = await appStore.checkSystemState()
    if (!authenticated) {
      appStore.isAuthenticated = false  // Show Login view
      return
    }

    // 3. Load application data
    await loadAppData()

  } catch (e) {
    console.error("Hub connection error:", e)
    appStore.isAuthenticated = false  // Show Login view
  }
}

const loadAppData = async () => {
  try {
    // Parallel fetch: All data sources at once
    await Promise.all([
      fetchConfig(),
      appStore.checkSetupStatus(),  // isArmed, version
      fleetStore.fetchFleet(),
      fleetStore.fetchUptime(fleetStore.activeTimeframe),
      eventsStore.fetchEvents(),
    ])

    // Register WebSocket event handlers
    wsService.on('onNewEvent', (payload) => eventsStore.handleWsEvent(payload))
    wsService.on('onNewSensor', (payload) => fleetStore.handleWsUpdate('NEW_SENSOR', payload))
    wsService.on('onDeleteSensor', (payload) => fleetStore.handleWsUpdate('DELETE_SENSOR', payload))
    wsService.on('onSilenceSensor', (payload) => fleetStore.handleWsUpdate('SILENCE_SENSOR', payload))
    wsService.on('onSensorHeartbeat', (payload) => fleetStore.handleWsUpdate('SENSOR_HEARTBEAT', payload))
    wsService.on('onNewNode', (payload) => fleetStore.handleWsUpdate('NEW_NODE', payload))
    wsService.on('onUpdateNode', (payload) => fleetStore.handleWsUpdate('UPDATE_NODE', payload))
    wsService.on('onDeleteNode', (payload) => fleetStore.handleWsUpdate('DELETE_NODE', payload))
    wsService.on('onNodeSynced', (payload) => fleetStore.handleWsUpdate('NODE_SYNCED', payload))
    wsService.on('onReconnect', async () => {
      console.log("WebSocket reconnected: syncing missed data...")
      await Promise.all([
        fleetStore.fetchFleet(),
        fleetStore.fetchUptime(fleetStore.activeTimeframe),
        eventsStore.fetchEvents(),
      ])
    })
    wsService.on('onSyncCharts', () => {
      fleetStore.fetchUptime(fleetStore.activeTimeframe)
    })

    // Connect WebSocket
    wsService.connect()

    // CRITICAL: Set authentication AFTER all data loads
    // This ensures components mount into populated stores
    appStore.isAuthenticated = true
    appStore.isInitialized = true

  } catch (e) {
    console.error("Failed to load application data:", e)
    // Graceful degradation: show dashboard even if some data failed
    appStore.isAuthenticated = true
    appStore.isInitialized = true
  }
}
```

**Critical Invariant:** `isAuthenticated = true` is set **last**, after all data has been fetched. This prevents the authenticated shell from rendering before stores are populated.

## Post-Login Sequence

```
User submits login form
  ↓
Login.vue calls: appStore.login(password)
  ↓
Store action: await api.post('/login', { password })
  ↓
Backend verifies password, sets hw_auth cookie
  ↓
api.post() resolves → { success: true }
  ↓
Login.vue emits: this.$emit('login-success')
  ↓
App.vue handler: onLoginSuccess()
  ↓
appStore.requiresSetup = false
await loadAppData()  ← Same as cold boot
  ↓
All data fetched, WS connected
  ↓
appStore.isAuthenticated = true  ← Shell reveals
  ↓
Dashboard component mounts into populated stores
```

**Note:** `login()` does NOT set `isAuthenticated`. This prevents the Login component from unmounting before it emits 'login-success', which would leave `loadAppData()` uncalled.

---

# Error Handling & Rollback

## API Error Handling

All API errors are caught at the store level, not the component level:

```javascript
const toggleSilence = async (nodeId, sensorId, targetState) => {
  const sensor = getSensor(nodeId, sensorId)
  if (!sensor) return

  // 1. Save previous state
  const previous = sensor.isSilenced

  // 2. OPTIMISTIC update
  sensor.isSilenced = targetState

  try {
    // 3. Send to backend
    await api.patch(`/api/v1/nodes/${nodeId}/sensors/${sensorId}/silence`, {
      is_silenced: targetState,
    })
    // Success — nothing needed (UI already updated)

  } catch (err) {
    // 4. ERROR — ROLLBACK
    sensor.isSilenced = previous
    console.error('Failed to toggle sensor silence:', err)
    throw err  // Let component handle UI feedback (toast)
  }
}
```

## API Error Class

```javascript
class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status  // HTTP status code
  }
}

// Usage in stores
try {
  await api.patch(...)
} catch (err) {
  if (err.status === 401) {
    // Unauthorized
  } else if (err.status === 409) {
    // Conflict
  } else {
    // Generic error
  }
}
```

## Rollback Patterns

### Single Field
```javascript
const previous = sensor.isSilenced
sensor.isSilenced = newValue  // optimistic
// On error:
sensor.isSilenced = previous
```

### Multiple Fields
```javascript
const previous = {
  alias: node.alias,
  tags: [...node.tags],
  publicIp: node.publicIp,
  privateIp: node.privateIp,
}
// Apply optimistic changes
Object.assign(node, updates)
// On error:
Object.assign(node, previous)
```

### Array State
```javascript
const sensorIdx = node.installedSensors.findIndex(s => s.id === sensorId)
const previous = sensorIdx !== -1 ? {...node.installedSensors[sensorIdx]} : null

// Optimistic remove
if (sensorIdx !== -1) node.installedSensors.splice(sensorIdx, 1)

// On error:
if (previous && sensorIdx !== -1) {
  node.installedSensors.splice(sensorIdx, 0, previous)
}
```

---

# Debugging Guide

## Blank Dashboard After Login

**Symptom:** Login succeeds, but dashboard shows no data (empty nodes, no events)

**Causes & Fixes:**

1. **Authenticated shell revealed before stores populated**
   - Check: `loadAppData()` actually completed
   - Check: All `await Promise.all([...])` calls resolved
   - Look at Network tab: All `/api/v1/*` requests returned 200
   - Fix: Ensure `isAuthenticated = true` is set AFTER data loads

2. **Store state exists but components not watching**
   - Check: Component has `const { nodes } = storeToRefs(fleetStore)`
   - Fix: Use `storeToRefs()` to destructure (preserves reactivity)
   - ❌ Wrong: `const nodes = fleetStore.nodes` (loses reactivity)
   - ✅ Right: `const { nodes } = storeToRefs(fleetStore)`

3. **Fetch failed silently**
   - Check: Browser console for errors
   - Check: Network tab for failed requests (4xx/5xx)
   - Check: `isFetching` state didn't complete
   - Fix: Handle errors in fetch actions (currently caught but logged)

## UI Not Updating

**Symptom:** State changed but component doesn't re-render

**Causes & Fixes:**

1. **Array reassignment broke reactivity**
   - ❌ Wrong: `nodes.value = newArray`
   - ✅ Right: Use `splice()`, `push()`, `Object.assign()`
   - Debug: Check component watchers firing (Vue DevTools)

2. **Watched property not reactive**
   - Check: Using `ref()` for state (not plain objects)
   - Check: Property accessed with `.value` in `<script setup>`
   - Debug: Log the ref in mounted: `console.log(nodes.value)`

3. **Computed property dependency missing**
   - Check: Computed function lists all reactive dependencies
   - ❌ Wrong: `computed(() => nodes.value.filter(...).length)`
   - ✅ Right: Explicitly list: `computed(() => { ... fleetStore.nodes, ... })`

4. **Filter/computed chain broken**
   - Debug: Log `filteredEvents.value` to verify filtering
   - Check: Filter logic correctly handles composite keys
   - Example: Filtering by `sensor_id` alone (missing `node_id`)

## WebSocket Not Receiving Updates

**Symptom:** Events appear on backend but don't reach UI

**Causes & Fixes:**

1. **WebSocket not connected**
   - Debug: Open DevTools → Network tab → WS filter
   - Check: `ws://host/api/v1/ws` shows "101 Switching Protocols"
   - If connecting: Red "X" icon = failed connection
   - Fix: Check auth (session cookie), check backend logs

2. **Handler not registered**
   - Check: `wsService.on('onNewEvent', handler)` called
   - Check: Called BEFORE `wsService.connect()`
   - Fix: Verify all handlers registered in `loadAppData()`

3. **Message not matching event type**
   - Debug: Console logs in `ws.js` _handleMessage()
   - Check: Incoming `data.type` matches registered type
   - Example: Backend sends `"NEW_SENSOR"`, code listens for `"onNewSensor"` (wrong)

4. **Event filtered out**
   - Check: `handleWsEvent()` filtering by `selectedNode` + `selectedSensor`
   - Example: Event has `node_id: "node-1"` but store has `selectedNode: "node-2"`
   - Debug: Log in `handleWsEvent()` to check filter logic

5. **Reconnect loop but no data refresh**
   - Check: `onReconnect` callback registered
   - Check: `onReconnect` actually calls `fetchFleet()` etc.
   - Debug: See console logs: "WebSocket Reconnected: Syncing missed data..."

## Composite Key Bugs (Sensors)

**Symptom:** Wrong sensor updated, cross-node collisions

**Example Bug:**
```javascript
// ❌ WRONG: Using sensor_id alone (not unique!)
const getSensor = (sensorId) => {
  return nodes.value
    .flatMap(n => n.installedSensors)
    .find(s => s.id === sensorId)
}

// If two nodes both have a sensor named "tcp-tarpit", first match is returned
// Silencing node-2's sensor actually silences node-1's
```

**Correct Pattern:**
```javascript
// ✅ CORRECT: Composite key (node_id + sensor_id)
const getSensor = (nodeId, sensorId) => {
  const node = getNode(nodeId)
  return node?.installedSensors?.find(s => s.id === sensorId) || null
}

// Always pass both:
fleetStore.getSensor("node-abc", "tcp-tarpit")
```

## Optimistic Update Didn't Rollback

**Symptom:** User clicks "Silence", UI updates, API call fails, UI doesn't revert

**Causes & Fixes:**

1. **Previous state not captured**
   - Check: Saved state before optimistic update
   - Fix: Add `const previous = sensor.isSilenced` at start

2. **Rollback applied to wrong object**
   - Check: `getSensor()` returns same reference as UI is using
   - Debug: `console.log(previous, sensor)` — are they same?
   - Fix: Use store getters, not local references

3. **Promise chain broken**
   - Check: Error handling in try/catch
   - ❌ Wrong: Fire-and-forget `api.patch().catch()`
   - ✅ Right: `await api.patch()` with explicit try/catch

---

# Summary

## Key Takeaways

1. **Layered architecture:** Strict separation, unidirectional flow
2. **State storage:** Three stores (app, fleet, events) each owning a domain
3. **Data flow:** Views → Stores → API/WS → Backend; Backend → WS/API → Stores → UI
4. **API data:** Authoritative, complete, used for cold boot and verification
5. **WebSocket data:** Incremental, realtime, applied immediately
6. **Normalization:** Done once at store boundary, never in components
7. **Reactivity:** Preserve array/object identity, use splice/assign, never reassign
8. **Error handling:** Optimistic first, rollback on failure, always catch at store level
9. **Bootstrap:** Data fetches in parallel, WS connects last, `isAuthenticated` set last
10. **Reconnect:** Full data refetch to catch any missed updates during disconnect