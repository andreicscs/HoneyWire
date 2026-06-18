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