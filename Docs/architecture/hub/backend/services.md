# Domain Services & Orchestration

The `internal/services` layer is the brain of the HoneyWire Hub. It contains the actual "verbs" of the system and is strictly framework-agnostic. Services define their own dependencies via narrow interfaces, ensuring high testability and clear boundaries.

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

## Data Flow Example (Event Processing)

To understand how the layers interact, consider the lifecycle of an intrusion event when a sensor POSTs to `/api/v1/event`:

1. **Router (`internal/api/router.go`):** Matches the route and invokes `AgentAuthMiddleware`.
2. **Middleware:** 
   - Reads the `X-Api-Key` or `Bearer` token.
   - Calls `authService.AuthenticateNodeRequest()`.
   - Injects the authenticated `NodeID` into the request `Context`.
3. **API Handler (`internal/api/events.go`):**
   - Parses the JSON body into a `models.Event`.
   - Extracts `NodeID` from the context.
   - Calls `eventService.ProcessEvent(event, nodeID)`.
4. **Event Service (`internal/services/event/service.go`):**
   - Verifies version compatibility.
   - Calls `store.InsertEvent()`.
   - Calls `store.UpdateNodeLastHeartbeat()`.
   - Checks `store.IsSensorSilenced()`. If false, triggers `notifyService.Dispatch()`.
   - Sends the event to the SIEM via `siemService.QueueEvent()`.
   - Broadcasts the update to the UI via `broadcaster.Broadcast("NEW_EVENT")`.
5. **API Handler:** Returns a clean `HTTP 200 OK`.

## Developer Guide: Adding a New Feature

When extending the backend, strict adherence to the established flow is required:

1. **Models:** Define your data structures in `models/`.
2. **Store:** Write the SQL queries in `store/`. Update the interface definition at the top of your target Service.
3. **Service:** Write the business logic in `services/<domain>/service.go`. Handle all errors, business validation, and side effects here.
4. **API Handler:** Write a thin HTTP wrapper in `api/<domain>.go`. Use `api.RespondError` and `api.SendJSON`.
5. **Router & Main:** Register the route in `api/router.go` and wire the dependencies in `cmd/hub/main.go`.
