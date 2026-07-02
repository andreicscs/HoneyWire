# Events Store Architecture (`events.ts`)

## Entity State vs. Projection State
The `events.ts` store manages two fundamentally different state models with specific mutation rules:

### Mutable Entity State (Events List)
*   **Identity-Preserving:** Uses a Hybrid Update Model. When active filters change, the store refetches the complete history from the backend.
*   **Real-time Hydration:** When a `NEW_EVENT` arrives via WebSocket, the store evaluates if it belongs to the active view. If so, it is instantly `unshift`ed into the array without issuing a network request.

### Immutable Projection State (Analytics Charts)
*   **Backend-Generated:** The UI never loops over local events to aggregate data. 
*   **Atomically Replaced:** When a new projection arrives, the entire object reference is replaced to trigger clean re-renders via shallow watchers.

## Contextual Invalidation Strategy
To prevent race conditions during high-frequency filter changes (e.g., rapidly clicking timeframes), the store uses `AbortController` to cancel in-flight projection requests.
When a matching realtime WebSocket event arrives, the store does not manually mutate projections. Instead, it invalidates the current projection timestamp, triggering components to automatically refetch the exact backend snapshot.

## Composite Key Enforcement
Because sensor IDs are not globally unique, filtering and matching logic inside the store strictly evaluates both `nodeId` AND `sensorId` simultaneously to prevent telemetry cross-contamination.

## Encapsulation
State is hidden inside private `ref`s and accessed strictly through exported `computed` getters, physically preventing Vue components from accidentally mutating or clearing the event arrays.