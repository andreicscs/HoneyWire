# Project Structure Map

This document is a static map of the HoneyWire monorepo. It answers the question: *"Where do I edit X?"*

For step-by-step guides on *how* to change these components, see `workflows.md`.

---

## 1. Hub (`/Hub`)
The central backend and frontend dashboard.

### Go Backend
*   **`cmd/hub/`**: The main entry point (`main.go`). This is where dependencies are instantiated and wired together.
*   **`internal/api/`**: The HTTP transport layer. Contains routers, middleware, and handlers. *No business logic goes here.*
*   **`internal/services/`**: The domain layer. This is where business logic, side-effects, and background workers live (e.g., event processing, node provisioning).
*   **`internal/store/`**: The SQLite persistence layer. Contains raw SQL queries, database migrations, and `models/`.
*   **`internal/projections/`**: The CQRS analytics engine. Pure functions that calculate dashboard charts and stats from raw events.

### Vue 3 Frontend
*   **`ui/src/views/`**: Page-level orchestrators (e.g., `Dashboard.vue`, `Fleet.vue`).
*   **`ui/src/components/`**: Reusable UI widgets and layout primitives.
*   **`ui/src/stores/`**: Pinia state management (`app.ts`, `fleet.ts`, `events.ts`). *All frontend business logic and data filtering lives here.*
*   **`ui/src/services/`**: API clients and the WebSocket connection manager.

---

## 2. Wizard (`/wizard`)
The host-side deployment and discovery CLI.

*   **`cmd/wizard/`**: The main entry point.
*   **`internal/cli/`**: UI formatting, prompts, and terminal output styling.
*   **`internal/commands/`**: The CLI verbs (e.g., `discover.go`, `apply.go`, `teardown.go`).
*   **`internal/app/`**: Local state management and configuration loading.
*   **`internal/deploy/`**: Compose file generation and Docker daemon interaction.
*   **`core/discovery/`**: The engine that evaluates host services against sensor heuristics.
*   **`core/scanner/`**: Low-level Linux inspection logic (reading `/proc`, mapping open sockets).
*   **`core/api/`**: The Hub API client.

---

## 3. Sensors (`/Sensors`)
The containerized decoy applications.

*   **`official/`**: Production-grade sensors maintained by the core team (e.g., `TcpTarpit`, `FileCanary`).
*   **`community/`**: Third-party sensors submitted by users.
*   **`templates/`**: Boilerplate scaffolds for creating new sensors (e.g., `go-sensor-template`).

---

## 4. SDKs (`/SDKs`)
Language-specific libraries for building sensors.

*   **`go-honeywire/`**: The official Go SDK. Handles heartbeat loops, event queuing, and strict Hub API contract adherence.