# HoneyWire Hub Backend Overview

The HoneyWire Hub backend is the central orchestrator of the system, written entirely in Go. It strictly adheres to a **Domain Service Pattern** (a lightweight form of Clean Architecture / Domain-Driven Design). This design ensures high testability, predictable state management, and clear separation of concerns.

## Technology Stack

Go (Golang) was chosen as the primary language for the Hub backend for several strategic reasons:
- **Concurrency & WebSockets:** Go's lightweight goroutines make it exceptionally well-suited for handling thousands of simultaneous, long-lived WebSocket connections without the overhead of traditional threading models.
- **Single Static Binary:** Go compiles to a single, statically linked executable. This allows the Hub to be deployed in a highly secure, minimal Distroless container sandbox with no runtime dependencies.
- **Performance:** As a distributed security system, the Hub must ingest high-velocity telemetry from sensors across the network without dropping packets. Go's performance characteristics ensure low latency and high throughput.
- **Standard Library:** Go's robust `net/http` standard library allowed us to build the entire API and router without relying on heavy third-party web frameworks, reducing the attack surface.
- **Embedded Database Compatibility:** Go's ability to seamlessly integrate with CGO-free, pure-Go SQLite drivers enables a zero-configuration, single-node persistence layer perfectly suited for edge deployments.

## Core Principles

The backend architecture is built upon five foundational principles:

1. **Thin Transport Layer:** HTTP handlers in the `api` package contain **zero** business logic. They are strictly responsible for parsing JSON, reading context, and formatting HTTP responses.
2. **Isolated Domain Services:** All business rules, side effects (like WebSockets or Push Notifications), and complex transactions live exclusively in the `services` package.
3. **Interface Segregation:** Services define their own narrow interfaces (e.g., `Store` and `Broadcaster`). They do not depend on concrete implementations like `SQLiteStore` or `WebSocketService`.
4. **Explicit Composition:** All dependencies are instantiated and wired together precisely once in the Composition Root (`cmd/hub/main.go`). Global state is strictly prohibited.
5. **Contextual Authentication:** Middleware authenticates requests and injects the verified identity (e.g., `NodeID`) into the standard `context.Context` to be used by the layers below.

## Directory Structure

The backend source code is structured as follows:

```text
internal/
├── api/            # Transport Layer (HTTP Handlers, Middleware, Router)
├── compose/        # Secure Compose Compiler & Validation Engine
├── models/         # Core Domain Entities (Structs, JSON tags)
├── projections/    # CQRS Read-Models (Analytics, Dashboards)
├── services/       # Domain Layer (Business Logic, Orchestration)
└── store/          # Persistence Layer (SQLite implementation)
```

## Core Subsystems & Layers

### 1. Transport Layer (`internal/api`)
The entry point for all network requests.
- **Handlers:** Bound to domain-specific struct receivers. They handle JSON marshaling/unmarshaling, input validation, and HTTP status codes. They **never** interact with the database directly.
- **Middleware:** Handles Authentication, extracts credentials, validates them, and attaches identity to the request context.
- **Router:** Maps HTTP routes to specific Handlers and applies Middleware groups.
    - **SPA Routing & Frontend Embedding:** The router securely serves the embedded Vue 3 frontend. It enforces strict API protection (returning pure 404 JSON for unmatched `/api/` routes) while providing a transparent SPA fallback (serving `index.html`) for unrecognized paths, enabling seamless frontend History API navigation (e.g., `/dashboard`) without reload errors.

### 2. Secure Compose Compiler (`internal/compose`)
Responsible for compiling deterministic, hardened `honeywire-compose.yml` configurations served to remote edge nodes. See the [Compose Compiler Architecture](./compose.md) for more details.
- **Secure Defaults by Inversion:** Explicitly maps specific allowed schema primitives (e.g. strict Linux capabilities, normalized volume paths) into a locked-down Compose base instead of attempting to filter arbitrary config maps.
- **Validation Engine:** Sanitizes configurations, normalizes directory paths via `filepath.Clean()`, enforces a capability allowlist, and rejects potentially unsafe interpolation patterns.
- **Dual-Manifest Version Resolution:** Supports manual updates and historical rollbacks. When a node requests its configuration, the compiler resolves the specific deployed version:
  - If the node is running the latest version, it pulls from the in-memory registry cache.
  - If the node is running an older/historical version, the compiler lazy-loads and caches that specific tagged manifest from the registry (e.g., `hw-sensor-tarpit-v1.0.0.json`) to perform compilation matching the node's deployed state.

### 3. Domain Service Layer (`internal/services`)
The brain of the application. Everything in `internal/services/*` is framework-agnostic.
- Contains the actual "verbs" of the system (e.g., `ProcessHeartbeat`, `CreateNode`).
- Handles all side effects.
- Relies on Dependency Injection.

### 4. Persistence Layer (`internal/store`)
The data access layer.
- **SQLite Store:** Implements the narrow interfaces required by the Services.
- Responsible for SQL queries, transactions, and JSON marshaling into Go structs.
- Operates in Write-Ahead Log (WAL) mode with connection pooling.
- Contains **no** business logic.

#### Schema Migrations & Constraints
- The Hub uses an embedded, automated migration system. 
- **CRITICAL:** Do NOT use the legacy `CREATE new_table -> DROP old_table -> RENAME` workaround for altering schemas if the table is referenced by foreign keys with `ON DELETE CASCADE`. Doing so while foreign keys are enforced will instantly trigger the cascade and wipe out dependent data (e.g., dropping `node_sensors` deletes all `events`).
- **Solution:** HoneyWire uses a modern SQLite driver (3.35.0+) that natively supports `ALTER TABLE ... DROP COLUMN`. Always use native `ALTER TABLE` operations to guarantee schema mutations are non-destructive and isolated.

### 5. Read/Analytics Layer (`internal/projections`)
A specialized CQRS (Command Query Responsibility Segregation) pattern used for heavy dashboard analytics.
- Used for high-volume aggregations like Threat Velocity and Threat Severity Distributions.
- Returns flat **DTOs** (Data Transfer Objects) derived via pure functions, preventing the frontend from traversing large data arrays.


## Authentication Strategies

HoneyWire employs a multi-tiered authentication strategy depending on the actor:

1. **UI Dashboard (Humans):** Secured via short-lived Sessions and HTTP-Only, Secure, SameSite `hw_auth` Cookies. Handled via `UIAuthMiddleware`.
2. **Sensors/Agents (Machines):** Secured via statically generated, cryptographically random API Keys (`hw_key_...`). Passed via `Authorization: Bearer` or `X-Api-Key` headers and validated by `AgentAuthMiddleware`. These are aggressively cached in memory to prevent database bottlenecks during heartbeat storms.
3. **Dual-Auth Endpoints:** Shared endpoints (like `/api/v1/manifests`) are protected by `DualAuthMiddleware`, which attempts Bearer authentication first before falling back to Cookie validation.
