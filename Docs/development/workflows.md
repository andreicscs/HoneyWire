# Common Development Workflows

This guide provides step-by-step instructions for the most frequent development tasks in the HoneyWire monorepo.

## 1. Adding a New API Endpoint to the Hub

Because the Hub strictly adheres to a Domain Service architecture, adding a new endpoint requires touching several layers to ensure separation of concerns.

**Step 1: Define the Domain Model (if applicable)**
Add any new structs to `internal/models/`. Use appropriate JSON tags.

**Step 2: Implement the Business Logic (Service Layer)**
1. Navigate to `internal/services/`.
2. Define a new interface method on the relevant service (or create a new service).
3. Implement the method on the concrete struct. This layer should handle validation, DB calls (via the store interface), and side effects.

**Step 3: Define the HTTP Handler (Transport Layer)**
1. Navigate to `internal/api/handlers/`.
2. Create a handler function attached to the appropriate handler struct.
3. *Rule:* The handler must only read the request context/JSON, call the injected Service method, and write the JSON response. No business logic allowed here.

**Step 4: Register the Route**
1. Open `internal/api/router.go`.
2. Register the new HTTP path, attach necessary authentication middleware (`UIAuthMiddleware` vs `AgentAuthMiddleware`), and point it to your new handler.

**Step 5: Update the Frontend API Client**
1. Open `ui/src/services/api.ts` in the frontend code.
2. Add a strongly-typed wrapper method for your new endpoint.

---

## 2. Modifying an Analytics Dashboard (Projections)

HoneyWire uses CQRS-style projections for analytics to avoid complex front-end data crunching.

**Step 1: Update the DTO**
1. Open `internal/projections/dtos.go`.
2. Add the new required field to the Data Transfer Object (e.g., adding `UniqueIPs` to `FleetHealthDTO`).

**Step 2: Update the Calculator Function**
1. Open the relevant file in `internal/projections/` (e.g., `calculators.go`).
2. Modify the pure function to compute your new field based on the raw input events.

**Step 3: Update the Frontend Pinia Store**
1. Navigate to `ui/src/stores/`.
2. Update the TypeScript interfaces to match your new DTO schema.
3. Update the Vue component (`ui/src/components/...`) to render the new data field.

---

## 3. Adding a New UI Page

**Step 1: Create the View and Components**
1. Create a new `.vue` file in `ui/src/views/` (e.g., `NewFeature.vue`).
2. Break down complex parts into reusable widgets in `ui/src/components/`.

**Step 2: Register the View**
*Note: HoneyWire currently does not use a dedicated frontend router. Views are conditionally rendered.*
1. Open the main layout orchestrator (e.g., `ui/src/App.vue` or the main shell component).
2. Add your new view to the `v-if` / `v-else-if` conditional rendering logic linked to the active UI state (like `currentView`).

**Step 3: Wire State Management**
1. If the page requires complex state, create a new file in `ui/src/stores/` using Pinia.
2. Ensure you mutate state in-place (e.g., `array.push()`) rather than reassigning variables to preserve Vue's reactivity.