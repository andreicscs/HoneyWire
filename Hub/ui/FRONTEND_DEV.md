# HoneyWire Frontend Developer Guide

This document contains practical guidelines, project structure, component organization, and the design system used in the HoneyWire frontend. 

For state management, data flow, real-time updates, and architectural rules, see [Frontend Architecture](../Hub/ui/FRONTEND_ARCHITECTURE.md).

## Stack

Vue 3 · Pinia · Vite · TailwindCSS · Native WebSocket · OKLCH Design System

---

# Project Structure

```text
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

# Frontend Component Rules

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
