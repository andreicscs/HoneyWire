# Fleet Store Architecture Guide

This document describes the design, state management, and normalization strategies used in the `fleet.ts` Pinia store.

The `fleet.ts` store manages the state of the infrastructure: Nodes, Sensors, their configurations, and their real-time heartbeats. Because this store powers complex CRUD operations, it utilizes specific architectural patterns to maintain UI consistency and speed.

---

# 1. The Composite Key Model

A major architectural decision in HoneyWire is that **Sensor IDs are only unique within the scope of their parent Node.**

If you deploy a `tcp-tarpit` to `Node A` and a `tcp-tarpit` to `Node B`, their raw `sensorId`s are identical.

To handle this safely on the frontend without nested, slow array traversal, the store creates a flat O(1) read model by hashing them into a **Composite Key**.

```typescript
// Inside normalizeSensorData()
const compositeId = `${nodeId}:${rawSensorId}`
```

*Rule:* Any UI component or store action attempting to fetch, patch, or silence a sensor MUST pass both the `nodeId` and the `rawSensorId` to generate the correct composite hash.

---

# 2. Optimistic Update with Rollback Cache

For non-critical metadata changes (like renaming a Node or adding a Tag), `app.ts`'s "Reconciled Update" (wait for backend, then fetch) feels too sluggish. Instead, `fleet.ts` uses **Optimistic UI with an explicit rollback mechanism.**

The `patchNode` and `patchSensor` gatekeepers are designed to return the *previous* state of the object right before the patch is applied.

```typescript
const updateNode = async (nodeId: string, payload: Partial<FleetNode>) => {
  // 1. Store previous state and immediately update UI
  const previousState = patchNode(nodeId, {
    alias: payload.alias
  })

  try {
    // 2. Await Backend
    await api.patch(`/api/v1/nodes/${nodeId}`, payload)
  } catch (err) {
    // 3. Rollback the exact object on failure
    if (previousState) patchNode(nodeId, previousState)
    throw err
  }
}
```

*Rule:* The `nodesById` and `sensorsById` record maps are the absolute truth. Never mutate the array returned by the `nodes` getter directly. Always route through `patchNode` or `patchSensor`.

---

# 3. Transport Boundary Normalization

The shape of the JSON that the backend returns (`rawNode`) is not necessarily the exact shape the Vue components want to consume.

The store implements a rigid Normalization layer at the ingress boundary (`normalizeNodeData`, `normalizeSensorData`).

```typescript
const normalizeNodeData = (raw: any): FleetNode | null => {
  // Protect against undefined values and provide safe fallbacks
  return {
    id: raw.nodeId,
    alias: raw.alias || 'Unnamed Node',
    hasPendingConfig: raw.hasPendingConfig ?? false,
    // ...
  }
}
```

*Rule:* A raw backend response MUST pass through a normalizer before touching the State Tree. This means if the Go backend changes a property from `has_pending_config` to `hasPendingConfig`, you only have to fix the frontend in *one* exact line of code.

---

# 4. Partial Invalidation (Subgraph Fetching)

When a user deletes a sensor on a Node, we don't want to call `fetchFleet()` and wipe out the entire network tree just to update one branch.

Instead, we execute a **Subgraph Fetch** via `fetchNodeDetails(nodeId)`.

This function securely reaches into the `sensorsById` map, deletes *only* the sensors belonging to that specific Node using our reverse index (`sensorsByNodeId`), and then merges the fresh node payload into the State Tree.

---

# 5. Encapsulation

Like `app.ts` and `events.ts`, the core objects (`nodesById`, `sensorsById`) are hidden behind private `ref`s and exported as read-only `computed` properties.

This physically prevents Vue components from accidentally executing code like:

```typescript
// Typescript will block this:
fleetStore.nodes[0].alias = "My Hacked Node"
```