# HoneyWire Frontend Architecture & Data Flow

This document explains the structural design, state management, data flow, and real-time update strategy of the HoneyWire frontend. It focuses on how data is stored, updated, rendered, and the distinction between WebSocket (realtime) and API (authoritative) data sources.

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
8. [Persistence Boundaries](#persistence-boundaries)
9. [Bootstrap & Lifecycle](#bootstrap--lifecycle)
10. [Error Handling & Rollback](#error-handling--rollback)
11. [Debugging Guide](#debugging-guide)
12. [Analytics Projection Architecture](#analytics-projection-architecture)

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

No layer reaches above itself.
- Views never call APIs directly.
- Stores never import Vue or manage UI state.
- Services never touch component state.
- API Client never implements business logic.

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
9. **Backend-owned analytics** — Aggregations and projections are computed server-side, never derived from raw frontend entity traversal
10. **Immutable projection snapshots** — Analytics projections replace object references intentionally
11. **Contextual analytics orchestration** — Widgets request projections based on active UI context
12. **Transport race protection** — Projection fetches use AbortController cancellation to prevent stale overwrites

## Design Goals

The frontend architecture is optimized for:
- deterministic realtime synchronization
- low-latency UI updates
- backend-authoritative analytics
- predictable Vue reactivity behavior
- resilience to websocket disconnects
- minimal component-level business logic

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

### Action Patterns
- `login(password)` / `logout()` — Session management.
- `completeSetup()` / `checkSetupStatus()` — Hub setup coordination.
- `toggleArmed()` — Optimistic update of system arming state, with rollback on failure.

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
      lastHeartbeat: 1716345600000,
      installedSensors: [
        {
          id: "tcp-tarpit-1",
          status: "up",
          isSilenced: false,
          lastHeartbeat: 1716345600000,
          // ...
        }
      ]
    }
  ],
  
  uptimeData: [ ... ],
  selectedNode: "node-abc" | null,
  selectedSensor: "tcp-tarpit-1" | null,
  activeTimeframe: "24H" | "7D" | "30D" | "1H"
}
```

### Computed Properties (Indexed Access)

For O(1) lookup performance, the store maintains computed maps:

```javascript
const getNode = (nodeId) => nodeMap.value[nodeId] || null
const getSensor = (nodeId, sensorId) => sensorIndex.value[nodeId]?.[sensorId] || null
```

**Critical:** Always use composite key `node_id + sensor_id`. Sensor IDs are only unique within a node.

### Action Patterns
- **Data Fetching**: The store fetches complete node lists on cold boot, or individual node details after a mutation (like adding/removing a sensor).
- **Mutations (Optimistic Update)**: Actions like creating a node, or toggling sensor silence, are applied immediately to the local state, followed by an API request. On error, the state is rolled back.
- **Deletions**: Entities are removed locally immediately, and a full refetch is triggered if the backend deletion fails.

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
      severity: "critical",
      is_read: 0 | 1,
      is_archived: 0 | 1,
      // ...
    }
  ],
  unreadCount: 5,
  activeEvent: null,
  isFetching: false
}
```

### Filtering Model

Events are filtered reactively by:
- archive mode
- selected node
- selected sensor

Filtering occurs entirely inside the store so components never implement filtering logic.

### Action Patterns
- **Fetching**: Retrieves events based on active filters (archived state, node, sensor).
- **Mutations**: Marking events as read or archived.
- **WebSocket Updates**: Appends new incoming events to the local list immediately.

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
  3. Await API: api.patch(`/api/v1/nodes/${nodeId}/sensors/${sensorId}/silence`, ...)
    ↓
Backend processes request
    ↓
Success (2xx): Store does nothing (UI already updated)
Error (4xx/5xx): ROLLBACK sensor.isSilenced = previous, show toast
```

Pattern: Optimistic first, confirm async, rollback on error.

---

## 2. API Fetch → Store → Normalize → Merge → UI Update

**Example: Fetch fleet on cold boot**

```
App.vue calls: await fleetStore.fetchFleet()
    ↓
Store Action:
  1. API: await api.get('/api/v1/nodes')
  2. NORMALIZE: raw.map(normalizeNode)
  3. MERGE with existing (in-place)
    ↓
Vue reactivity triggered
    ↓
UI re-renders with new data
```

Pattern: Normalize at boundary, preserve array identity, merge existing to prevent watchers breaking.

---

## 3. WebSocket Event → Service → Store Handler → UI Update

**Example: Backend broadcasts NEW_SENSOR event**

```
Backend: Node deployed a sensor
    ↓
WS broadcast: { type: "NEW_SENSOR", payload: { ... } }
    ↓
ws.js routes to App.vue handler
    ↓
fleetStore.handleWsUpdate('NEW_SENSOR', payload)
    ↓
NORMALIZE and PUSH to node's array
    ↓
Vue reactivity triggered
```

Pattern: WebSocket updates are applied immediately (no rollback), optionally trigger full refetch for authoritative state.

---

# API Data vs WebSocket Data

## API Data (Authoritative)

**Characteristics:**
- Source of truth — represents backend state at fetch time
- Complete — includes all fields and nested data
- Normalized
- Used for: Cold boot, manual refreshes, critical mutations

## WebSocket Data (Realtime Delta)

**Characteristics:**
- Incremental — only includes changed fields
- Immediate
- Event-driven
- Used for: Heartbeats, new events, config syncs

Priority: WebSocket updates are applied immediately; API fetches verify and correct state.

---

# Normalization & Reactivity

## Data Normalization

All backend payloads are normalized at the store boundary using `normalize*` functions. Components never normalize. This absorbs backend schema changes at the boundary and provides components with a consistent frontend schema.

## Reactive Identity Preservation

Vue 3 reactivity depends on object identity. If you break the identity, watchers and computed properties fail.

### ❌ WRONG: Reassignment breaks identity

```javascript
nodes.value = newArray.map(normalizeNode)  // New reference = broken watchers
```

### ✅ CORRECT: Mutation preserves identity

Always mutate arrays/objects in-place to preserve identity. Never reassign. Use `splice()`, `push()`, `Object.assign()`.

```javascript
// Clear without reassigning
nodes.value.splice(0, nodes.value.length)
incoming.forEach(node => nodes.value.push(node))

// Mutate in-place
Object.assign(existing, { alias: "new alias" })
```

---

# WebSocket Integration

The WebSocket layer is decoupled from Vue/Pinia via the `HoneyWireWS` class in `services/ws.js`. 

**Responsibilities:**
- Establish connection and auto-reconnect with exponential backoff.
- Parse incoming JSON messages and dispatch to registered callbacks.
- Maintains no state management.

`App.vue` registers handlers to pass parsed messages to the respective store `handleWsEvent` or `handleWsUpdate` methods.

---

# Persistence Boundaries

Understanding what survives a page refresh is key to the state design:

| State | Persistent | Source |
|-------|------------|--------|
| Authentication | Session cookie | Backend |
| Fleet state | Ephemeral | Rehydrated via API bootstrap + WS |
| Events | Ephemeral | Rehydrated via API bootstrap + WS |
| UI selections | Optional localStorage | Frontend |
| Projections | Ephemeral | Rehydrated via backend API |

---

# Bootstrap & Lifecycle

## Cold Boot Sequence

User loads the app (`onMounted` in App.vue):

1. Check if setup is required (`checkRequiresSetup`).
2. Check if authenticated (`checkSystemState`).
3. Load application data in parallel (`fetchFleet`, `fetchEvents`, etc.).
4. Connect WebSocket and register handlers.
5. **Critical Invariant:** `isAuthenticated = true` is set **last**, after all data has been fetched. This prevents the authenticated shell from rendering before stores are populated.

---

# Error Handling & Rollback

All API errors are caught at the store level, not the component level. State mutations use explicit rollback patterns on error.

```javascript
// 1. Save previous state
const previous = sensor.isSilenced
// 2. OPTIMISTIC update
sensor.isSilenced = targetState

try {
  // 3. Send to backend
  await api.patch(...)
} catch (err) {
  // 4. ERROR — ROLLBACK
  sensor.isSilenced = previous
  throw err
}
```

---

# Debugging Guide

## Blank Dashboard After Login
- Check: `loadAppData()` actually completed and all `await Promise.all([...])` calls resolved.
- Fix: Ensure `isAuthenticated = true` is set AFTER data loads.

## UI Not Updating
- Check: Array reassignment broke reactivity (`nodes.value = newArray`). Fix by using `splice()`, `push()`, `Object.assign()`.
- Check: Watched property is accessed with `.value` in `<script setup>`.

## WebSocket Not Receiving Updates
- Check: WebSocket not connected (check Network tab for "101 Switching Protocols").
- Check: Handler not registered before `wsService.connect()`.
- Check: Event filtered out inside `handleWsEvent()` due to `selectedNode` / `selectedSensor`.

## Composite Key Bugs (Sensors)
- Symptom: Wrong sensor updated, cross-node collisions.
- Fix: Always pass both `nodeId` and `sensorId` to get a sensor, as `sensorId` alone is not unique.

## Optimistic Update Didn't Rollback
- Check: Saved state before optimistic update was not captured or was applied to the wrong object reference.

---

# Analytics Projection Architecture

The frontend delegates all analytics aggregations to the backend. This section covers how we manage projection state.

## Projection vs Entity State

HoneyWire distinguishes between two fundamentally different frontend state models:

### 1. Entity State (Mutable)
Examples: `nodes[]`, `installedSensors[]`, `events[]`

Characteristics:
- Long-lived
- Incrementally mutated
- Identity-preserving
- Updated directly from WebSocket deltas

Rules:
- Never replace array references
- Merge in-place
- Use `splice`/`push`/`Object.assign`

Purpose: Efficient realtime synchronization of raw operational data.

### 2. Projection State (Immutable)
Examples: `severityProjection`, `threatVelocityProjection`

Characteristics:
- Backend-generated
- Flat DTO snapshots
- Replaced atomically
- Filter/context dependent

Rules:
- Reference replacement is intentional
- No deep watchers
- No local aggregation
- No array traversal derivation

Purpose: Efficient rendering of authoritative backend analytics.

Example:
```javascript
// Correct projection replacement
this.severityProjection = await res.json()
```
Projection snapshots intentionally replace references so shallow watchers trigger deterministic chart updates.

## Projection Invalidation Strategy

Analytics projections use a different realtime model than raw entities. While raw entity updates from WebSocket mutate local state directly, realtime events for projections trigger conditional invalidation.

Flow (e.g. `NEW_EVENT`):
1. WS event arrives.
2. Store evaluates the active filter context.
3. If the event affects the current projection:
   - Abort in-flight projection requests (via `AbortController`).
   - Refetch the updated projection snapshot from the backend.
4. The projection reference is replaced.
5. The chart's shallow watcher re-renders.

This architecture prevents frontend aggregation drift, context desynchronization, and stale filtered analytics.
