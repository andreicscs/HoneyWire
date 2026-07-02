# Fleet Store Architecture (`fleet.ts`)

## Composite Key Model
Sensor IDs are only unique within the scope of their parent Node. To prevent nested array traversal, the store uses a flat O(1) read model by hashing them into a **Composite Key** (`${nodeId}:${rawSensorId}`). Store actions must pass both to generate the correct hash.

## Optimistic Update with Rollback Cache
For non-critical metadata changes, `fleet.ts` uses Optimistic UI. Gatekeepers (`patchNode`, `patchSensor`) apply the update immediately and return the *previous* state. If the subsequent backend API call fails, the store uses the previous state to seamlessly rollback.

## Transport Boundary Normalization
Raw backend JSON is strictly normalized (`normalizeNodeData`, `normalizeSensorData`) before entering the State Tree. This insulates Vue components from backend schema changes and missing properties.

## Partial Invalidation (Subgraph Fetching)
Mutations (e.g., deleting a sensor) do not trigger a full `fetchFleet()`. Instead, a **Subgraph Fetch** (`fetchNodeDetails(nodeId)`) executes. It securely deletes the specific node's branch from the `sensorsById` map and merges the fresh payload.

## Encapsulation
Core objects (`nodesById`, `sensorsById`) are hidden behind private `ref`s and exported as read-only `computed` properties, physically preventing Vue components from accidentally executing direct mutations.