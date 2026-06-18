# Hub Frontend Architecture & Design

The HoneyWire Hub frontend is a Vue 3 application built on a strict, layered architecture. It prioritizes predictable state management, real-time synchronization, and deterministic rendering. It strictly separates raw operational state from analytical dashboards.

## Technology Stack

The HoneyWire frontend leverages a modern, highly reactive stack optimized for real-time telemetry and predictable state management:
- **Vue 3 (Composition API):** Chosen for its lightweight and granular reactivity system, which is crucial for handling high-frequency WebSocket deltas without triggering unnecessary component re-renders.
- **TypeScript:** Adopted to ensure type safety across complex Domain objects (Events, Nodes, Sensors) and to prevent runtime errors when consuming dynamic backend schemas.
- **Pinia:** The official Vue state management library. It provides central, deterministic state ownership and robust support for TypeScript, making it ideal for managing our dual state models (Entity vs. Projection).
- **Vite:** Selected for its fast cold starts and Hot Module Replacement (HMR), significantly improving the developer experience.
- **TailwindCSS:** Provides a utility-first styling approach, allowing us to enforce a strict, semantic design system.

## Layered Architecture

The application enforces unidirectional dependencies:

```text
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

**Key Rules:**
- Views never call APIs directly.
- Stores never import Vue or manage UI state.
- Services never touch component state.

## Core Principles

1. **Deterministic Rendering:** The same state always produces the same UI output.
2. **Centralized State Ownership:** Pinia Stores are the single source of truth.
3. **Selective Optimistic Responsiveness:** Lightweight toggles (e.g., sensor silence, mark-as-read) update the UI immediately with asynchronous confirmation. Most standard mutations (e.g., deploying sensors, archiving events) wait for the API to confirm before updating state.
4. **Rollback Safety:** Where optimistic updates are used, there is a clear undo path on failure.
5. **Reactive Identity Stability:** Array and object references are preserved; they are mutated in-place to ensure Vue watchers do not break.
6. **Normalized Boundaries:** Data is normalized once at the store entry point.
7. **Backend-Owned Analytics:** Aggregations and projections are computed server-side, never derived from raw frontend entity traversal. *(See [Exception regarding events](#known-architectural-exception-event-pagination))*

## State Models

The frontend maintains two fundamentally different state models:

### 1. Entity State (Mutable)
Examples: `nodes[]`, `events[]`
- **Characteristics:** Long-lived, incrementally mutated, identity-preserving.
- **Rules:** Never replace array references. Merge data in-place using `splice` or `push`.
- **Purpose:** Efficient real-time synchronization of raw operational data via WebSocket deltas.

### 2. Projection State (Immutable)
Examples: `severityProjection`, `threatVelocityProjection`
- **Characteristics:** Backend-generated flat DTO snapshots. Dependent on the active UI filter context.
- **Rules:** Reference replacement is intentional. No deep watchers, local aggregation, or array traversal.
- **Purpose:** Efficient rendering of authoritative backend analytics.

## Data Flow Lifecycle

Data flows predictably through the system:

**1. User Action (Optimistic Update)**
```text
View clicks action → Store saves previous state → Store updates state optimistically → Store calls API
  ↳ If API succeeds: Do nothing (UI already updated).
  ↳ If API fails: Rollback state to previous, show error toast.
```

**2. API Fetch**
```text
Component requests data → Store calls API → Normalize payload → Merge with existing state (in-place) → UI re-renders.
```

## Realtime & WebSocket Integration

The system balances authoritative REST API data with real-time WebSocket deltas:

- **API Data (Authoritative):** Represents the complete backend state at fetch time. It is used for cold boots, manual refreshes, and critical mutations.
- **WebSocket Data (Realtime Delta):** Incremental and event-driven. The WebSocket layer is entirely decoupled from Vue/Pinia via the `HoneyWireWS` class. `App.vue` registers handlers that route parsed messages into the respective Pinia store (`handleWsEvent` or `handleWsUpdate`).

### Projection Invalidation Strategy
Because projections are immutable snapshots, they cannot be updated incrementally via WebSocket deltas like entity arrays. When a `NEW_EVENT` arrives via WebSocket:
1. The Store evaluates if the event falls within the current active filter context (timeframe, node, sensor).
2. If relevant, the Store aborts any in-flight requests (using `AbortController`) and re-fetches the updated projection snapshot from the API.
3. The projection reference is replaced, and the chart re-renders.

This design prevents frontend aggregation drift and ensures the backend remains the sole authority on analytics.

## Persistence Boundaries

To understand state durability:

- **Session Cookie:** Authentication (Persistent, HTTP-Only).
- **Frontend Store (Ephemeral):** Fleet state, events, and projections are rehydrated via API bootstrap on page load.
- **Local Storage:** Used exclusively for saving the user's theme preference (Light/Dark mode). It is explicitly *not* used for storing events, fleet data, timeframe selections, or sidebar states.

## Known Architectural Exception: Event Pagination

HoneyWire adheres to the principle of **Backend-Owned Analytics**, meaning heavy filtering and aggregation should always occur server-side (as seen with our immutable Projection DTOs).

**Exception:** Currently, **event pagination is handled entirely on the frontend**. 

This decision was explicitly chosen to prioritize initial simplicity and speed of development. By pulling events and paginating them in the Vue layer, we avoided writing complex cursor or offset-based pagination in the initial iteration of the SQLite persistence layer.

**Future Migration Path:**
This is considered technical debt. If frontend performance bottlenecks arise due to memory overhead (e.g., traversing thousands of events in the browser), or if HoneyWire adopters request it for scale, the event pagination logic will be migrated to the backend to match the architecture of our dashboard projections.
