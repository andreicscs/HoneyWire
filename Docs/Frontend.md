# HoneyWire Frontend Architecture

## Stack

- Vue 3
- Pinia
- Vite
- Native Fetch API
- Native WebSocket API
- TailwindCSS
- Custom OKLCH Design System

---

# Architecture Overview

The frontend follows a strict layered architecture:

| Layer | Responsibility |
|---|---|
| Views & Components | UI rendering only |
| Stores (Pinia) | Global state + business logic |
| Services | Persistent network systems |
| Utils | Shared helpers/theme logic |

Components never directly manage backend state.

All backend communication flows through Pinia stores.

---

# Project Structure

```text
src/
├── api/                # API composables/config loaders
├── assets/             # Global styles & design system
├── components/
│   ├── dashboard/      # Dashboard widgets
│   ├── layout/         # Shell layout components
│   └── ui/             # Reusable design-system primitives
│
├── services/           # Persistent services (WebSocket)
├── stores/             # Pinia state domains
├── utils/              # Shared utilities/helpers
├── views/              # Route-level screens
│
├── App.vue             # Root orchestrator
└── main.js             # Application bootstrap
```

---

# State Architecture

## `app.js`

Global application/UI state.

### Responsibilities

- active view routing
- archive mode
- sidebar state
- armed/disarmed state
- version/config state
- authentication/logout

---

## `fleet.js`

Infrastructure state.

### Responsibilities

- nodes
- installed sensors
- node details
- uptime metrics
- sensor selection
- node deployment state

### Important

Sensors are identified using:

```text
node_id + sensor_id
```

This prevents cross-node collisions.

---

## `events.js`

Telemetry + alert pipeline.

### Responsibilities

- realtime events
- unread counters
- filtering
- archive handling
- websocket event ingestion

---

# Data Flow

HoneyWire uses strict unidirectional flow:

```text
UI Interaction
    ↓
Pinia Action
    ↓
API Request / State Mutation
    ↓
Reactive Update
    ↓
Component Re-render
```

Views never mutate state directly.

---

# WebSocket Architecture

## Service Layer

Location:

```text
src/services/ws.js
```

The WS service:

- owns socket lifecycle
- reconnects automatically
- emits parsed payloads
- contains zero Vue logic

---

## App Orchestrator

`App.vue` wires:

```text
WebSocket Events
    ↓
Store Actions
    ↓
Reactive UI Updates
```

Example:

```text
NEW_EVENT
    ↓
eventsStore.handleWsEvent()
```

---

# View Structure

| View | Purpose |
|---|---|
| Dashboard.vue | Telemetry & analytics |
| FleetView.vue | Node management |
| NodeDetailView.vue | Sensor deployment/configuration |
| Store.vue | Sensor catalog |
| Settings.vue | Hub configuration |
| Setup.vue | Initial setup flow |
| Login.vue | Authentication |

---

# Component System

## Dashboard Components

Located in:

```text
components/dashboard/
```

Contains telemetry widgets:

- EventTable
- SeverityChart
- ThreatVelocity
- UptimeHeatmap
- TrafficFilters

These components are data-driven and stateless whenever possible.

---

## UI Component Library

Located in:

```text
components/ui/
```

Reusable primitives grouped by domain.

### Categories

| Folder | Purpose |
|---|---|
| branding | logos/theme |
| feedback | alerts/modals/status |
| forms | inputs/buttons/selectors |
| layout | cards/widgets/page shells |
| navigation | menus/sidebar/nav |

---

# Design System

HoneyWire uses a centralized token-based design system.

Location:

```text
src/assets/style.css
```

Built entirely on CSS custom properties using OKLCH color space.

---

# Theme Architecture

## Core Principles

- semantic tokens only
- no hardcoded component colors
- light/dark parity
- accessibility-first contrast
- consistent spacing scale

---

# Color System

## Structural Colors

Used for layout hierarchy.

Examples:

```css
--bg
--bg-base
--bg-surface
--border-default
```

---

## Interactive Colors

Used for actions/buttons.

Examples:

```css
--primary-main
--secondary-main
--danger-main
```

---

## Severity Tokens

Telemetry severity colors:

```css
--sev-critical
--sev-high
--sev-medium
--sev-low
--sev-info
```

Used consistently across:

- charts
- alerts
- status dots
- event tables

---

# Typography

Two-font system:

| Token | Font |
|---|---|
| `--font-sans` | Inter |
| `--font-mono` | JetBrains Mono |

Typography is fully tokenized:

```css
--text-h1
--text-base
--text-sm
```

---

# Spacing System

Consistent layout rhythm:

```css
--space-card-p
--space-flow
--space-label-gap
```

No arbitrary spacing should be introduced in components.

---

# Radius & Elevation

## Radius Tokens

```css
--radius-sm
--radius-md
--radius-lg
```

## Shadow Tokens

```css
--shadow-sm
--shadow-md
--shadow-lg
```

All cards/modals/widgets use shared elevation values.

---

# Z-Index System

Strict stacking hierarchy:

| Token | Usage |
|---|---|
| `--z-dropdown` | menus |
| `--z-overlay` | backdrops |
| `--z-modal` | dialogs |
| `--z-toast` | notifications |

Avoid arbitrary z-index usage.

---

# Tailwind Integration

The design system is exposed through Tailwind using:

```css
@theme
```

This maps CSS tokens into utility classes.

Example:

```css
bg-bg-surface
text-text-h
border-border-default
```

This ensures:

- consistent theming
- centralized color control
- instant dark-mode support

---

# Dark Mode

Dark mode is token-driven using:

```css
.dark { ... }
```

Components never define separate dark styles manually unless necessary.

Theme switching only swaps root variables.

---

# Frontend Rules

## Components

Components SHOULD:

- remain presentation-focused
- receive data via props/stores
- emit actions upward

Components SHOULD NOT:

- perform complex business logic
- own global state
- duplicate API calls

---

## Stores

Stores are the single source of truth.

Stores:

- own async requests
- normalize backend data
- handle websocket mutations
- expose reactive state

---

## Services

Services must remain framework-agnostic.

They should not:

- import Vue
- import Pinia
- mutate UI directly

---

# Debugging Rules

## UI Not Updating

Check:

1. component action call
2. store mutation
3. reactive dependency
4. watcher trigger

---

## Realtime Issues

Check:

1. WS connection status
2. event routing in App.vue
3. store WS handler
4. active filters

---

## Sensor Selection Bugs

Always verify:

```text
node_id + sensor_id
```

Selection/filtering logic must use composite keys.

---

# Frontend Philosophy

HoneyWire prioritizes:

- deterministic state flow
- centralized business logic
- reusable primitives
- strict visual consistency
- zero duplicated network logic
- scalable component composition