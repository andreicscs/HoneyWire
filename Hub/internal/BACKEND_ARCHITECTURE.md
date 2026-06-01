# HoneyWire Hub Backend Architecture

This document outlines the architectural design and data flow of the HoneyWire Hub backend. The backend is written in Go and strictly adheres to a **Domain Service Pattern** (a lightweight form of Clean Architecture / Domain-Driven Design) to ensure high testability, predictable state management, and clear separation of concerns.

---

## Core Principles

1. **Thin Transport Layer:** HTTP handlers in the `api` package contain **zero** business logic. They only parse JSON, read context, and format HTTP responses.
2. **Isolated Domain Services:** Business rules, side effects (like WebSockets or Notifications), and complex transactions live exclusively in the `services` package.
3. **Interface Segregation:** Services define their own narrow `Store` and `Broadcaster` interfaces. They do not know about the concrete `SQLiteStore` or `WebSocketService`.
4. **Explicit Composition:** All dependencies are instantiated and wired together precisely once in the Composition Root (`cmd/hub/main.go`). Global state is strictly prohibited.
5. **Contextual Authentication:** Middleware authenticates requests and injects the verified identity (e.g., `NodeID`) into the standard `context.Context`.

---

## Directory Structure

```text
internal/
├── api/            # Transport Layer (HTTP Handlers, Middleware, Router)
├── models/         # Core Domain Entities (Structs, JSON tags)
├── projections/    # CQRS Read-Models (Analytics, Dashboards)
├── services/       # Domain Layer (Business Logic, Orchestration)
└── store/          # Persistence Layer (SQLite implementation)
```

---

## The Four Layers

### 1. Transport Layer (`internal/api`)
The entry point for all network requests.

* **Handlers (`nodes.go`, `events.go`, etc.):** 
  * Bound to domain-specific struct receivers (e.g., `NodeHandler`).
  * Responsible for JSON Unmarshaling, Input Validation, and HTTP Status Codes.
  * **Rule:** Never interact with the database directly. Always delegate to a Service.
* **Middleware (`middleware.go`):** 
  * Handles Authentication (`UIAuthMiddleware`, `AgentAuthMiddleware`, `DualAuthMiddleware`).
  * Extracts credentials (Cookies, Bearer tokens), validates them via the `auth.Service`, and attaches the resulting `nodeID` to the `http.Request` context.
* **Router (`router.go`):**
  * Maps HTTP routes to specific Handlers and applies Middleware groups.

### 2. Domain Service Layer (`internal/services`)
The brain of the application. Everything in `internal/services/*` is framework-agnostic.

* **Services (`event`, `node`, `config`, etc.):**
  * Contain the actual "verbs" of the system (`ProcessHeartbeat`, `CreateNode`).
  * Handle side effects (Dispatching webhooks, queuing SIEM logs, broadcasting WebSockets).
  * **Dependency Injection:** Services declare their dependencies via narrow interfaces. For example, `event.Service` defines a `Store` interface that `sqlite.go` implicitly satisfies.
* **Background Workers:**
  * Long-running tasks that belong to a specific domain (e.g., `eventSvc.StartRetentionWorker`, `sensorSvc.StartHealthMonitor`) live inside that service's package.

### 3. Persistence Layer (`internal/store`)
The data access layer.

* **SQLite Store (`sqlite.go`):**
  * Implements the narrow interfaces required by the Services.
  * Responsible for SQL queries, transactions, and JSON marshaling into Go structs.
  * Operates in Write-Ahead Log (WAL) mode with connection pooling for high concurrency.
  * **Rule:** Contains no business logic or external side effects.

### 4. Read/Analytics Layer (`internal/projections`)
A specialized CQRS pattern for heavy dashboard analytics.

* Used for high-volume aggregations like Threat Velocity and Severity Distributions.
* Instead of raw arrays, these return flat **DTOs** (Data Transfer Objects) derived via pure `calculator.go` functions.
* See `projections/severity/SEVERITY_ARCHITECTURE.md` for a deep dive.

---

## Data Flow Example (Event Processing)

When a sensor detects an intrusion and sends an HTTP POST to `/api/v1/event`:

1. **Router:** Matches the route and invokes `AgentAuthMiddleware`.
2. **Middleware:** 
   * Reads the `X-Api-Key` or `Bearer` token.
   * Calls `authService.AuthenticateNodeRequest()`.
   * Injects the authenticated `NodeID` into the request `Context`.
3. **API Handler (`events.go`):**
   * Parses the JSON body into a `models.Event`.
   * Extracts `NodeID` from the context.
   * Calls `eventService.ProcessEvent(event, nodeID)`.
4. **Event Service (`services/event/service.go`):**
   * Verifies version compatibility.
   * Calls `store.InsertEvent()`.
   * Calls `store.UpdateNodeLastHeartbeat()`.
   * Checks `store.IsSensorSilenced()`. If false -> calls `notifyService.Dispatch()`.
   * Calls `siemService.QueueEvent()`.
   * Calls `broadcaster.Broadcast("NEW_EVENT")` to update connected UI clients instantly.
5. **API Handler:** Returns `HTTP 200 OK`.

---

## Authentication Strategies

HoneyWire employs a multi-tiered authentication strategy depending on the actor:

1. **UI Dashboard (Humans):**
   * Secured via short-lived Sessions and HTTP-Only, Secure, SameSite `hw_auth` Cookies.
   * Managed by `services/auth` (Brute-force protection, lockout tracking, session invalidation).
   * Validated by `UIAuthMiddleware`.

2. **Sensors/Agents (Machines):**
   * Secured via statically generated, cryptographically random API Keys (`hw_key_...`).
   * Passed via `Authorization: Bearer` or `X-Api-Key` headers.
   * Cached in memory via `sync.Map` in `auth.Service` to prevent database bottlenecks during heartbeat storms.
   * Validated by `AgentAuthMiddleware`.

3. **Dual-Auth Endpoints:**
   * Endpoints like `/api/v1/manifests` are accessed by *both* humans (UI Dashboard) and machines (Sensors).
   * Handled safely via `DualAuthMiddleware`, which attempts Bearer auth first, falling back to Cookie validation.

---

## Background Workers

Workers are decoupled from the HTTP transport layer and managed exclusively via the `context.Context` instantiated in `main.go`.

| Worker | Location | Purpose |
|---|---|---|
| **Health Monitor** | `services/sensor` | Polls every 30s. If a sensor misses heartbeats > 60s, updates status to `down` and broadcasts a WS update. |
| **Event Retention** | `services/event` | Wakes hourly to delete/archive events older than configured thresholds to prevent DB bloat. |
| **Chart Sync** | `services/websocket` | Emits an empty payload every 30s telling the UI to tick its time-series charts forward smoothly. |
| **Auth Sweeper** | `services/auth` | Cleans up expired sessions and brute-force IP lockout maps to prevent memory leaks. |
| **SIEM Forwarder** | `services/siem` | Drains the in-memory event channel over TCP/UDP to external log aggregators. |
| **Notifier** | `services/notify` | Drains the webhook channel to Slack/Discord/Gotify, preventing external API latency from blocking HoneyWire HTTP responses. |

---

## Adding a New Feature (Developer Guide)

To add a new endpoint, follow this strict sequence:

1. **Models:** Define your data structures in `models/`.
2. **Store:** Write the SQL query in `store/`. Update the interface definition at the top of your target Service.
3. **Service:** Write the business logic in `services/<domain>/service.go`. Handle errors, validation, and side effects here.
4. **API Handler:** Write a thin HTTP wrapper in `api/<domain>.go`. Use `api.RespondError` and `api.SendJSON`.
5. **Router & Main:** Register the route in `api/router.go`.
