# Events Store Architecture Guide

This document describes the design, state management, and projection strategies used in the `events.ts` Pinia store.

The `events.ts` store manages the telemetry pipeline, bridging the gap between raw backend event logs, high-frequency WebSocket deltas, and deeply contextual UI analytics projections.

---

# 1. Entity State vs. Projection State

The `events.ts` store distinguishes between two fundamentally different frontend state models, which are treated with different mutation rules.

### Mutable Entity State
**Examples:** `events[]`

*   **Long-Lived:** The event list persists across page navigations.
*   **Identity-Preserving:** When new events arrive over WebSocket, they are `unshift`ed directly into the array without replacing the array reference itself.
*   **Client-Filtered:** We fetch all events for a given context and use Vue `computed` properties (`filteredEvents`) to narrow down the view based on `appStore.viewingArchive`.

#### The Hybrid Update Model (Refetch vs. Hydration)
The `events[]` array uses a hybrid approach to keep the UI snappy and reduce backend load during high-frequency attacks:

1. **Initial Load & Filter Changes (Refetch):** When the app first loads or the user changes their active filters (e.g., selects a different node), the store issues an `api.get` request. This completely replaces the `events` array with the authoritative history from the SQLite database.
2. **Real-time Delta Updates (Hydration):** When a `NEW_EVENT` arrives via WebSocket, the store does *not* refetch the event list. Instead, it evaluates the active filters locally inside `handleWsEvent`. If the new event belongs to the current view, it is *hydrated* directly into the local array using `unshift()`. This prepends the event to the top of the UI instantly without issuing a network request.

*(Note: This is the exact opposite of how Immutable Projections work, which rely exclusively on refetching).*

### Immutable Projection State
**Examples:** `severityProjection`, `threatVelocityProjection`

*   **Backend-Generated:** The UI *never* loops over local events to aggregate data. Chart data is calculated entirely by SQLite on the server.
*   **Atomically Replaced:** When a new projection arrives, the entire object reference is replaced intentionally to trigger clean, deterministic re-renders in Chart.js via Vue's shallow watchers.

```typescript
// Correct projection replacement:
state.value.threatVelocityProjection = (await response.json()) as ThreatVelocityProjection
```

---

# 2. Contextual Invalidation Strategy

Because Projections are server-rendered based on the user's active filters (Timeframe, Selected Node, Selected Sensor), they are highly vulnerable to race conditions if left unmanaged during a high-frequency intrusion.

The store uses a combination of `AbortController` and selective invalidation to keep the dashboard stable.

## The Abort Controller Gatekeeper
If a user clicks through three different timeframes rapidly, the store will cancel the in-flight HTTP requests for the previous two, guaranteeing that the final resolved promise belongs to the most recently requested context.

```typescript
const fetchThreatVelocityProjection = async (...) => {
  if (velocityAbortController) velocityAbortController.abort()
  velocityAbortController = new AbortController()

  // ... execute fetch with velocityAbortController.signal
}
```

## The Realtime Invalidation Flow
When a `NEW_EVENT` arrives via WebSocket:

1.  **Evaluate Context:** `handleWsEvent` checks if the new event actually belongs to the user's *current view* (e.g., if they are filtering for Node A, and the event happened on Node B, do nothing to the charts).
2.  **Invalidate Projections:** If the event *does* affect the current view, we don't attempt to manually mutate the local projection (which would cause aggregation drift). Instead, we trigger an invalidation.
3.  **Refetch:** The component (e.g., `ThreatVelocity.vue`) watches the invalidation timestamp and automatically asks the store to execute `fetchThreatVelocityProjection` again using the new exact context.

---

# 3. Composite Key Enforcement

Sensors in HoneyWire do not possess globally unique UUIDs; their IDs are only unique within the scope of their parent Node (e.g., a `tcp-tarpit` on `Node A` has the exact same string ID as a `tcp-tarpit` on `Node B`).

The `events.ts` store enforces strict composite key checks before allowing a WebSocket event into the filtered pipeline.

```typescript
const sensorMatch = 
  selectedSensor && 
  selectedNode && 
  payload.nodeId === selectedNode?.id && 
  payload.sensorId === selectedSensor?.sensorId
```
*Rule:* Filtering or matching logic must always evaluate `nodeId` AND `sensorId` simultaneously to prevent telemetry cross-contamination.

---

# 4. Encapsulation & Read-Only Access

Like all modern stores in the application, the actual `EventsState` is hidden inside a reactive `ref`.

Components accessing the store only interact with strictly exported `computed` getters. This guarantees that `App.vue` or `ThreatVelocity.vue` can never accidentally clear or overwrite the event arrays directly.

```typescript
// In the store:
const events = computed<EventPayload[]>(() => state.value.events)
const unreadCount = computed<number>(() => state.value.unreadCount)
```