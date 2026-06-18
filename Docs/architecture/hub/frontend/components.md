# Frontend Components

The HoneyWire frontend enforces a strict separation of concerns. UI rendering, layout orchestration, and ephemeral UI state belong in the component layer, while all business logic, filtering, and data normalization are pushed down to the Pinia stores.

## Component Categories

Components are divided into functional categories to maximize reusability and maintainability:

- **Dashboard Widgets (`components/dashboard/` `components/nodedetails/` ...):** Data-driven, presentation-focused charts and tables. They are generally stateless and reactive to projection updates (e.g., `EventTable`, `SeverityChart`).
- **UI Primitives (`components/ui/`):** Reusable design-system elements built strictly around our OKLCH token system. They are grouped into:
  - **Branding:** Logos and theme elements.
  - **Feedback:** Alerts, modals, toasts, and status indicators.
  - **Forms:** Inputs, buttons, and drop-down selectors.
  - **Layout:** Cards, widgets, and page shells.
  - **Navigation:** Menus, sidebars, and top navigation.

## Component Rules

Our core architectural principle is that the component layer must never bypass the store layer.

### Components
| ✅ Do | ❌ Don't |
|---|---|
| Stay presentation-focused | Import the API client (`client.ts`) |
| Receive data via props or stores | Call `fetch()` or `api.get()` |
| Emit actions upward | Own domain/business logic |
| Remain highly reusable | Normalize or map backend JSON payloads |

### Views
Views act as orchestrators for pages (e.g., `Dashboard.vue`, `NodeDetails.vue`).

| ✅ Do | ❌ Don't |
|---|---|
| Own ephemeral UI state (e.g., "is the modal open?") | Perform backend mutations directly |
| Delegate heavy logic to Pinia stores | Duplicate error rollback logic |
| Orchestrate layout and pass down props | Normalize API responses |

---

## Design System

The Hub frontend uses a centralized token system based on the OKLCH color space (located in `src/assets/style.css`).

### Principles

- Semantic tokens only — no hardcoded colors in components.
- Accessibility-first contrast.
- Dark/light parity.
- Theme switching swaps root variables only.

### Token Categories

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

### Tailwind Integration

Tokens exposed via `@theme`:

```css
bg-bg-surface    →  --bg-surface
text-text-h      →  --text-h
border-border-default  →  --border-default
```

### Dark Mode

```css
.dark { /* overrides root variables */ }
```