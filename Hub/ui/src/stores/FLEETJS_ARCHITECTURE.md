# Fleet Store Architecture Guide

This document describes the architectural design, state segregation, and mutation strategies used in the `fleet.js` Pinia store. 

The system has been refactored away from deep-object mutation and nested arrays toward a flat, normalized, and strictly gatekept state model. This ensures predictable Vue reactivity, race-condition immunity during WebSocket bursts, and O(1) relational lookups.

---

# Architecture Overview

## The 6-Phase State Model

```text
Incoming Payloads (HTTP / WS)
    ↓
Normalization & Composite ID Mapping
    ↓
State Gatekeepers (Snapshot vs Entity Patchers)
    ↓
Flat Authoritative Dictionaries (nodesById, sensorsById)
    ↓
Dynamic Read Models (Computed Projections)
    ↓
UI Components
```

---

# 1. State Segregation

The store strictly divides its reactive variables into three distinct categories. **Never mix these boundaries.**

### A. Authoritative Backend Snapshot State
The absolute source of truth. Contains flattened dictionaries keyed by strictly formatted IDs.
```javascript
const nodesById = ref({})
const sensorsById = ref({}) // Keyed by composite ID: `${nodeId}:${rawSensorId}`
```

### B. Frontend Operational UI State
Ephemeral state owned purely by the user's current session. Never sent to the backend as-is.
```javascript
const pendingNodeActions = ref(new Map())
const selectedNodeId = ref(null)
const selectedSensorId = ref(null) // Strictly holds composite ID
```

### C. Transport Control
Prevents race conditions (e.g., overlapping fetches or WebSocket overlaps) using `AbortController`.
```javascript
const abortControllers = new Map()
```

---

# 2. Data Normalization & Invariants

Raw API payloads are **never** injected directly into state. They pass through pure normalization functions.

## The Composite Sensor Key
Sensors do not have globally unique IDs from the backend (a `tcp-tarpit` on Node A has the same raw ID as a `tcp-tarpit` on Node B). To prevent state collisions, the store enforces a strict composite key invariant:

`Composite ID = {nodeId}:{rawSensorId}`

All internal dictionaries use the Composite ID. Public actions exposed to UI components require both the `nodeId` and `rawSensorId` to construct this key safely.

---

# 3. The Two Mutation Gatekeepers

Direct mutation of `nodesById` or `sensorsById` is strictly forbidden outside of these internal helper functions. This enforces the immutable *clone-update-replace* pattern required by Vue for reliable UI updates, and prevents WebSocket patches from bypassing validation or identity invariants.

### Gatekeeper 1: `commitStructuralSnapshot`
Used exclusively for HTTP responses. Atomically replaces entire object branches.
```javascript
const commitStructuralSnapshot = (nextNodes, nextSensors) => {
    nodesById.value = nextNodes
    sensorsById.value = nextSensors
}
```

### Gatekeeper 2: `applyWsEphemeralPatch`
Used exclusively by the WebSocket handler. Patches isolated fields (like `lastHeartbeat`) on existing entities without triggering full graph re-renders.
```javascript
const applyWsEphemeralPatch = (nodeId, sensorId, nodePatch, sensorPatch) => {
    // Shallow clone and patch
    if (nodeId && nodesById.value[nodeId] && nodePatch) {
        nodesById.value[nodeId] = { ...nodesById.value[nodeId], ...nodePatch }
    }
    // ...
}
```

### Gatekeeper 3: `applyOptimisticPatch`
Used by UI-driven API mutations. Captures the prior state and returns it, allowing immediate UI updates and providing an automatic rollback path if the HTTP request fails.
```javascript
const applyOptimisticPatch = (nodeId, patch) => {
    const existing = nodesById.value[nodeId]
    if (!existing) return null
    const previousState = { ...existing }
    nodesById.value[nodeId] = { ...existing, ...patch }
    return previousState
}
```

---

# 4. Dynamic Read Models (Selectors)

Because the authoritative state is flat, the store dynamically reconstructs the relational graph (Nodes -> Sensors) via computed properties.

## O(N) Compute -> O(1) Access
The `sensorsByNodeId` computed property iterates over sensors once when the dependency graph changes, building a map.

```javascript
const sensorsByNodeId = computed(() => {
    const map = {}
    for (const s of Object.values(sensorsById.value)) {
        (map[s.nodeId] ||= []).push(s)
    }
    return map
})
```

## Safe Data Hydration
When components request a Node, the store dynamically attaches the sensors.
```javascript
const getNode = (id) => {
    const node = nodesById.value[id]
    if (!node) return null
    return {
        ...node,
        installedSensors: sensorsByNodeId.value[node.id] || []
    }
}
```
*Rule:* Views must access nodes through `getNode()` or `selectedNode` to guarantee sensors are present.

---

# 5. WebSocket Update Rules

The WebSocket handler (`handleWsUpdate`) enforces strict boundary logic to prevent state drift between the backend and the frontend.

## Rule 1: Ephemeral Data = Immediate Local Patch
High-frequency telemetry (Heartbeats, Event Counters) are patched locally using `applyWsEphemeralPatch`. This prevents the UI from resetting or flashing during burst events.

## Rule 2: Structural Data = Authoritative Snapshot Fetch
Changes to the fleet topology (New Node, Deleted Sensor, Config Sync) **never** mutate the local dictionaries directly. Instead, they trigger a targeted HTTP fetch via `fetchNodeDetails(nodeId)`.

```javascript
// Ephemeral telemetry (Direct Patch)
if (type === 'SENSOR_HEARTBEAT') {
    applyWsEphemeralPatch(...)
    return
}

// Structural topology change (Refetch)
if (type === 'NEW_SENSOR' || type === 'DELETE_SENSOR') {
    fetchNodeDetails(payload.node_id)
    return
}
```

---

# 6. O(K) Garbage Collection (Sub-graph Overwrites)

When `fetchNodeDetails(nodeId)` runs to update a single node's snapshot, it must safely remove old sensors that no longer exist on the backend, without touching sensors belonging to other nodes.

Instead of an O(N) scan across all fleet sensors, it uses the O(1) `sensorsByNodeId` map to selectively prune the dictionary:

```javascript
const nextSensors = { ...sensorsById.value }

// O(K) Garbage Collection using the computed index
const oldSensors = sensorsByNodeId.value[node.id] || []
for (const s of oldSensors) delete nextSensors[s.id]

// Insert new incoming sensors
// ...
commitStructuralSnapshot(nextNodes, nextSensors)
```

---

# Important Architectural Rules

1. **Components never use composite IDs:** UI actions invoke actions using `(nodeId, rawSensorId)`. The store creates the composite key internally.
2. **No Deep Reactivity Mutations:** Arrays and Objects are never modified using `.push()` or `.splice()` inside the state tree. They are cloned, patched, and replaced via the Gatekeepers.
3. **UI Selection is an Object:** `selectedNode` and `selectedSensor` export complete, hydrated Objects, not just String IDs.