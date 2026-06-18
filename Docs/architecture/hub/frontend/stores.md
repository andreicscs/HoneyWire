# State Management & Stores

HoneyWire uses three primary Pinia stores to manage its domain state. Components delegate all business logic, filtering, and network interactions to these stores.

## 1. App Store (`app.ts`)

**Ownership:** UI navigation, authentication, and overall system state.

- **State:** `isAuthenticated`, `requiresSetup`, `currentView`, `sidebarOpen`, `isArmed`, `activeTimeframe`.
- **Actions:** Login/Logout, completing the initial setup, toggling the system arming state.
- **Note:** `isAuthenticated` acts as a final gatekeeper. During a cold boot, it is only set to `true` *after* all underlying data (fleet, events) has finished loading.

> 📖 **[View the detailed App Store Architecture](./store-app.md)**

## 2. Fleet Store (`fleet.ts`)

**Ownership:** Infrastructure state (Nodes, sensors, uptime, deployment metadata).

- **Structure:** Contains a normalized array of nodes, each containing its respective `installedSensors`.
- **Computed Maps:** Maintains O(1) lookup maps (e.g., `getNode(nodeId)`). 
- **Composite Keys:** Sensors are only unique within a node. The store strictly uses the composite key `nodeId + sensorId`.
- **Actions:** Fetches node lists, handles sensor creation/deletion, and optimistic updates for toggling sensor silence.

> 📖 **[View the detailed Fleet Store Architecture](./store-fleet.md)**

## 3. Events Store (`events.ts`)

**Ownership:** Telemetry state, intrusion events, and unread tracking.

- **Structure:** Array of normalized event objects and unread counters.
- **Filtering:** All filtering by archive mode, selected node, or selected sensor occurs *reactively inside the store*. Components do not implement filtering logic.
- **Actions:** Fetching events based on context, marking as read, and appending new incoming WebSocket events.

> 📖 **[View the detailed Events Store Architecture](./store-events.md)**

## Reactive Identity Preservation

A critical pattern in the HoneyWire frontend is preserving Vue 3's reactive object identity. 

**Rule:** Never reassign arrays or objects in the store. Always mutate in-place.

```typescript
// ❌ WRONG: Breaks watchers and computed properties
nodes.value = incomingNodes.map(normalizeNode)

// ✅ CORRECT: Mutates in-place, preserving identity
nodes.value.splice(0, nodes.value.length)
incomingNodes.forEach(n => nodes.value.push(normalizeNode(n)))
```

## Error Handling & Rollback

Because HoneyWire relies on Optimistic Updates for a snappy UI, errors must be handled gracefully:

1. **Save State:** `const previous = sensor.isSilenced`
2. **Optimistic Update:** `sensor.isSilenced = targetState` (UI updates instantly)
3. **API Call:** `await api.patch(...)`
4. **Rollback on Error:** In the `catch` block, `sensor.isSilenced = previous`
