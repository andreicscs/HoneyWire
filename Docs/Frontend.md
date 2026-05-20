# HoneyWire Frontend Architecture

## Stack

Vue 3 · Pinia · Vite · TailwindCSS · Native WebSocket · OKLCH Design System

---

# Architecture Overview

Strict layered architecture. Data flows one way. Each layer owns its responsibility and nothing else.

```
┌─────────────────────────────────┐
│  Views & Components             │  UI rendering + ephemeral state
├─────────────────────────────────┤
│  Stores (Pinia)                 │  Business logic + state ownership
├─────────────────────────────────┤
│  API Client · Services          │  Transport + error normalization
├─────────────────────────────────┤
│  Utils                          │  Shared helpers
└─────────────────────────────────┘
```

**No layer reaches above itself.** Views never call APIs. Stores never import Vue. Services never touch state.

---

# Core Principles

- Deterministic state flow
- Centralized business logic
- Optimistic UI with rollback safety
- Transport abstraction
- Framework-agnostic services
- Zero duplicated network logic
- Stable reactive identity
- Normalized backend data

---

# Project Structure

```
src/
├── api/
│   ├── client.js              # Centralized HTTP client
│   └── useConfig.js           # Config loader composable
│
├── assets/
│   └── style.css              # Design system tokens
│
├── components/
│   ├── dashboard/             # Dashboard widgets
│   ├── layout/                # App shell components
│   └── ui/                    # Reusable UI primitives
│
├── services/
│   └── ws.js                  # WebSocket service
│
├── stores/
│   ├── app.js                 # App + auth state
│   ├── events.js              # Event pipeline
│   └── fleet.js               # Infrastructure state
│
├── utils/
│   ├── chartConfig.js
│   ├── theme.js
│   └── useDropdown.js
│
├── views/
│   ├── Dashboard.vue
│   ├── FleetView.vue
│   ├── Login.vue
│   ├── NodeDetailView.vue
│   ├── Settings.vue
│   ├── Setup.vue
│   └── Store.vue
│
├── App.vue                    # Root orchestrator
└── main.js                    # Bootstrap
```

---

# Layer Responsibilities

## Views

Own ephemeral UI state only:

- Modal open/close
- Form input values
- Local loading flags
- Layout composition

**Never:** call APIs, normalize data, own business state, implement rollback.

---

## Components

Presentation-focused primitives.

**Should:** receive data via props/stores, emit actions upward, stay stateless when possible.

**Should not:** import the API client, call `fetch()`, own business logic, normalize backend payloads.

---

## Stores

Single source of truth. All backend communication flows through stores.

Stores own:

- Async requests
- Optimistic mutations + rollback
- Backend normalization
- WebSocket mutations
- Derived + indexed state
- Selection state

---

## API Client

`src/api/client.js` — centralized HTTP transport.

```js
api.get(url)
api.post(url, body)
api.patch(url, body)
api.delete(url)
api.request(url, options)   // non-JSON bodies, custom headers
```

The client owns transport, serialization, and error normalization. Nothing else.

Non-2xx throws `ApiError` with `.status` and `.message`. Stores catch and rollback.

**The client does NOT own:** business logic, rollback behavior, Vue state.

---

# State Architecture

## `app.js` — Application + Auth

| Responsibility | Key State |
|---|---|
| Auth lifecycle | `isAuthenticated`, `login()` |
| Setup flow | `requiresSetup`, `completeSetup()` |
| Active routing | `currentView` |
| Sidebar | `sidebarOpen` |
| Archive mode | `viewingArchive` |
| Armed state | `isArmed`, `toggleArmed()` |
| Version | `version` |
| Timeframes | `activeTimeframe`, `velocityTimeframe` |

### Auth Lifecycle Rule

> **`login()` does NOT set `isAuthenticated`.** App.vue controls shell reveal.

This prevents two critical bugs:

1. **Login unmount race** — if `isAuthenticated` flips during `login()`, the Login component unmounts before emitting its success event, so `loadAppData()` never runs.
2. **Empty store mount** — if the shell reveals before stores populate, Dashboard mounts into empty data.

**Correct sequence:**

```
login() succeeds → returns { success: true }
    ↓
Login emits 'login-success'
    ↓
App.vue calls loadAppData()
    ↓
stores populate + WebSocket connects
    ↓
isAuthenticated = true   ← shell revealed LAST
```

---

## `fleet.js` — Infrastructure

| Responsibility | Key State |
|---|---|
| Nodes | `nodes`, `fetchFleet()`, `createNode()` |
| Sensors | `getSensor()`, indexed access |
| Uptime | `uptimeData`, `fetchUptime()` |
| Deployment | `deployingSensors`, `deploySensor()` |
| Syncing | `syncingNodes`, `syncNode()` |
| WS updates | `handleWsUpdate()` |

### Sensor Identity

> **Always use `node_id + sensor_id` as a composite key.**

Never reference `sensor_id` globally — it's only unique within a node. Cross-node collisions will occur otherwise.

### Normalization

Backend payloads are normalized once at the store boundary:

```js
raw.last_heartbeat → lastHeartbeat
```

Components never normalize. They consume stable, frontend-safe structures.

### Reactive Merge

Preserve array identity. Never reassign:

```js
// ✅ Safe — preserves identity
array.splice(index, 1, newItem)
Object.assign(existing, updates)

// ❌ Breaks watchers and computed dependencies
array = filteredArray
```

### Indexed Access

O(1) lookup via computed maps:

```js
getNode(nodeId)
getSensor(nodeId, sensorId)
```

Avoids repeated nested scans across components.

---

## `events.js` — Telemetry Pipeline

| Responsibility | Key State |
|---|---|
| Realtime events | `events`, `handleWsEvent()` |
| Filtering | `filteredEvents` (computed) |
| Unread tracking | `unreadCount`, `markEventRead()` |
| Archive | `archiveEvent()`, `archiveAll()` |
| Bulk operations | `purgeEvents()` |

---

# Data Flow

Strict unidirectional:

```
User Action → View → Store Action → API Client → Backend
    ↓
Store Mutation → Reactive Update → Component Re-render
```

Views never mutate persistent state directly.

---

# Bootstrap Sequences

## Cold Boot (page load)

```
onMounted()
  → checkRequiresSetup()
  → checkSystemState()
  → loadAppData()
      → fetchConfig()
      → checkSetupStatus()
      → fetchFleet() + fetchUptime() + fetchEvents()
      → connect WebSocket
  → isAuthenticated = true
```

## Post-Login

```
login() → { success: true }
  → Login emits 'login-success'
  → onLoginSuccess()
  → loadAppData()
  → isAuthenticated = true
```

## Post-Setup

```
completeSetup() → { success: true }
  → Setup emits 'setup-complete'
  → onSetupComplete()
  → loadAppData()
  → isAuthenticated = true
```

**Invariant:** `isAuthenticated` is set **after** all data loads. Components mount into populated stores.

---

# WebSocket Architecture

## Service Layer

`src/services/ws.js` — framework-agnostic.

- Owns socket lifecycle
- Auto-reconnects
- Emits parsed payloads
- Zero Vue/Pinia imports

## App Orchestrator

`App.vue` wires events to store actions:

```
NEW_EVENT        → eventsStore.handleWsEvent()
NEW_SENSOR       → fleetStore.handleWsUpdate('NEW_SENSOR')
DELETE_SENSOR    → fleetStore.handleWsUpdate('DELETE_SENSOR')
NODE_SYNCED      → fleetStore.handleWsUpdate('NODE_SYNCED')
...
```

## Smart Reconnect

On reconnect, refetch authoritative state:

```
WS reconnect → fetchFleet() + fetchUptime() + fetchEvents()
```

Guarantees recovery of any missed realtime events.

---

# View Structure

| View | Purpose |
|---|---|
| Dashboard | Telemetry & analytics |
| FleetView | Node management |
| NodeDetailView | Sensor deployment & configuration |
| Store | Sensor catalog |
| Settings | Hub configuration |
| Setup | Initial setup flow |
| Login | Authentication |

---

# Component System

## Dashboard Widgets

`components/dashboard/` — data-driven, stateless when possible:

- EventTable
- SeverityChart
- ThreatVelocity
- TrafficFilters
- UptimeHeatmap

## UI Primitives

`components/ui/` — reusable design-system primitives:

| Category | Contents |
|---|---|
| branding | Logos, theme elements |
| feedback | Alerts, modals, status |
| forms | Inputs, buttons, selectors |
| layout | Cards, widgets, page shells |
| navigation | Menus, sidebar, nav |

---

# Design System

`src/assets/style.css` — centralized token system on OKLCH color space.

## Principles

- Semantic tokens only — no hardcoded colors in components
- Accessibility-first contrast
- Dark/light parity
- Theme switching swaps root variables only

## Token Categories

| Category | Examples | Usage |
|---|---|---|
| Structural | `--bg`, `--bg-surface`, `--border-default` | Layout hierarchy |
| Interactive | `--primary-main`, `--secondary-main`, `--danger-main` | Actions & states |
| Severity | `--sev-critical` through `--sev-info` | Charts, alerts, tables, status |
| Typography | `--text-h1`, `--text-base`, `--text-sm` | All text sizing |
| Spacing | `--space-card-p`, `--space-flow` | Layout rhythm |
| Elevation | `--radius-sm`, `--shadow-md` | Cards, modals, widgets |
| Z-index | `--z-dropdown` → `--z-toast` | Stacking hierarchy |

**No arbitrary values.** Use tokens.

## Typography

| Token | Font |
|---|---|
| `--font-sans` | Inter |
| `--font-mono` | JetBrains Mono |

## Tailwind Integration

Tokens exposed via `@theme`:

```css
bg-bg-surface    →  --bg-surface
text-text-h      →  --text-h
border-border-default  →  --border-default
```

## Dark Mode

```css
.dark { /* overrides root variables */ }
```

Components should not implement separate dark styles unless absolutely necessary.

---

# Frontend Rules

## Components

| ✅ Do | ❌ Don't |
|---|---|
| Stay presentation-focused | Import the API client |
| Receive data via props/stores | Call `fetch()` |
| Emit actions upward | Own business logic |
| Remain reusable | Normalize backend payloads |

## Views

| ✅ Do | ❌ Don't |
|---|---|
| Own ephemeral UI state | Perform backend mutations |
| Delegate to stores | Duplicate rollback logic |
| Orchestrate layout | Normalize API responses |

## Stores

| ✅ Do | ❌ Don't |
|---|---|
| Own all backend mutations | Use raw `fetch()` |
| Perform optimistic updates | Set `isAuthenticated` during `login()` |
| Rollback on failure | Reassign reactive arrays |
| Normalize at boundary | Let components normalize |

## Services

| ✅ Do | ❌ Don't |
|---|---|
| Stay framework-agnostic | Import Vue or Pinia |
| Own socket lifecycle | Mutate UI state directly |
| Emit parsed payloads | Own business logic |

---

# Debugging Guide

## Blank Dashboard After Login

The #1 cause: **authenticated shell revealed before stores populated.**

Check:

1. Did `loadAppData()` actually run?
2. Is `isAuthenticated` set **after** data loads?
3. Did Login emit before unmounting?
4. Did `login()` avoid setting `isAuthenticated` internally?

## UI Not Updating

1. Reactive dependency — is the component actually tracking the right ref?
2. Store mutation — did the mutation preserve array identity?
3. Watcher trigger — is the watcher source a valid ref/getter?
4. Computed dependency — are all inputs reactive?

## Realtime Issues

1. WS connection status — is it connected?
2. Reconnect flow — did it refetch?
3. App.vue routing — is the event mapped to the right handler?
4. Active filters — is the event filtered out by `node_id + sensor_id`?

## Optimistic Update Bugs

1. Rollback path — does it restore the exact previous state?
2. Mutation symmetry — does undo match the original mutation?
3. Array identity — did you use `splice`/`Object.assign`, not reassignment?

## Sensor Selection Bugs

> Always verify composite key: `node_id + sensor_id`

Never filter by `sensor_id` alone — cross-node collisions will produce wrong results.

---

# Architecture Philosophy

HoneyWire prioritizes:

- **Deterministic rendering** — same state, same output
- **Stable reactive identity** — no array reassignment, no broken watchers
- **Centralized state ownership** — stores are the truth
- **Optimistic responsiveness** — instant UI, background sync
- **Rollback safety** — every mutation has an undo
- **Strict layering** — no upward dependency
- **Transport abstraction** — swap API client without touching stores
- **Predictable data flow** — one direction, one owner