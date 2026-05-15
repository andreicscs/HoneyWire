# HoneyWire Hub — API Reference (v1.0)

[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)

All API routes are prefixed with `/api/v1` unless otherwise noted.


## Authentication

HoneyWire uses two separate authentication mechanisms depending on the caller.

### Dashboard (UI) routes

Protected by an HTTP-only session cookie named `hw_auth`. The cookie is issued by `POST /login` and is valid for 30 days.

### Agent (sensor) routes

Protected by a unique node key generated during provisioning. Pass the key using the Authorization header:

```text
Authorization: Bearer <HW_NODE_KEY>
```

## WebSocket

### GET /api/v1/ws

Upgrades the connection to a persistent WebSocket for real-time dashboard updates. Requires a valid `hw_auth` session cookie.

The Hub pushes a JSON message to all connected dashboard clients whenever a sensor reports an event, a new sensor comes online, a sensor is removed, a sensor's silence state changes, or when time-series charts need synchronization.

All messages share the same envelope:

```json
{ "type": "<MESSAGE_TYPE>", "payload": { ... } }
```

| Type | Trigger | Payload |
|---|---|---|
| `NEW_EVENT` | A sensor reports an event | Full event object (see event schema below) |
| `NEW_SENSOR` | A sensor sends its first heartbeat (new sensor detected) | `{ "node_id": "...", "sensor_id": "...", "timestamp": "..." }` |
| `SENSOR_HEARTBEAT` | A sensor sends a periodic heartbeat (every ~30 seconds) | `{ "node_id": "...", "sensor_id": "...", "timestamp": "..." }` |
| `SYNC_CHARTS` | Periodic time-series chart synchronization orchestrator (every 1 minute) | `null` |
| `DELETE_SENSOR` | A sensor is forgotten via the dashboard | `{ "node_id": "...", "sensor_id": "..." }` |
| `SILENCE_SENSOR` | A sensor\'s silence state is toggled | `{ "node_id": "...", "sensor_id": "...", "is_silenced": true }` |

**`NEW_EVENT` payload example:**

```json
{
  "type": "NEW_EVENT",
  "payload": {
    "id": 42,
    "timestamp": "2026-04-12 18:30:05",
    "contract_version": "1.0",
    "node_id": "node-12345678",
    "sensor_id": "core-dpi-engine",
    "event_trigger": "malformed_jwt_detected",
    "severity": "critical",
    "source": "104.28.19.12",
    "target": "Auth Gateway",
    "details": {
      "protocol": "TCP",
      "action_taken": "logged"
    },
    "is_read": false,
    "is_archived": false
  }
}
```

### Pure-Push Architecture

HoneyWire eliminates frontend polling entirely. The dashboard establishes a single persistent WebSocket connection and receives all state changes in real-time:

- **Events** push instantly via `NEW_EVENT` 
- **New sensors** are detected instantly via `NEW_SENSOR` and trigger a full fleet refresh
- **Status indicators** (Live online/offline dots) are updated at 0-latency thanks to `SENSOR_HEARTBEAT` messages natively updating the Vue state.
- **Fleet changes** (remove/silence) broadcast immediately via `DELETE_SENSOR` and `SILENCE_SENSOR`
- **Uptime charts** update seamlessly via `SYNC_CHARTS` orchestrator messages (every 1 minute). This "Push-to-Pull" pattern keeps all connected clients' historical time-series blocks perfectly synchronized without relying on wasteful, throttle-prone browser timers.

**Reconnection Recovery:** If the user's connection drops, the frontend automatically reconnects, fires an `onReconnect` hook, and fetches any missed data. This ensures the dashboard never displays stale information or misses events that occurred while offline.

## System & Configuration

HoneyWire splits configuration into two layers:
1. **Infrastructure Level (`.env`):** Defines immutable properties like ports, database paths, and emergency dashboard password overrides (`HW_DASHBOARD_PASSWORD`).
2. **Runtime Level (SQLite):** Governs hot-reloadable application logic like API keys, retention policies, and webhooks.

### GET /api/v1/setup/status
Checks if the database requires an initial master password and routing configuration. Automatically returns `false` if the `HW_DASHBOARD_PASSWORD` environment variable is strictly set.

**Response:**
```json
{
  "requires_setup": true
}
```

### POST /api/v1/setup
Initializes the runtime configuration and secures the Hub. Fails with `403 Forbidden` if the system has already been set up or if the environment variable lock is active.

**Payload:**
```json
{
  "password": "super_secure_password123",
  "hub_endpoint": "https://honeywire.my-domain.com",
  "hub_key": "hw_sk_randomstring"
}
```
**Response:** `200 OK`

---

### GET /api/v1/config
Retrieves the runtime settings.
**Requires Authentication:** Yes (UI Cookie)

**Default Values (On first boot):**
* `auto_archive_days` / `auto_purge_days`: `0` (Disabled)
* `webhook_type`: `ntfy`
* `webhook_events`: `["critical", "high", "medium", "low", "info"]`

**Response:**
```json
{
  "hub_endpoint": "https://honeywire.my-domain.com",
  "hub_key": "hw_sk_randomstring",
  "auto_archive_days": 0,
  "auto_purge_days": 30,
  "webhook_type": "ntfy",
  "webhook_url": "https://ntfy.sh/my_alerts",
  "webhook_events": ["critical", "high", "medium", "low", "info"]
}
```

### PATCH /api/v1/config
Partially updates the runtime configuration. Omitted fields are ignored and remain unchanged in the database. Valid types for `webhook_type` are: `ntfy`, `gotify`, `discord`, `slack`.
**Requires Authentication:** Yes (UI Cookie)

**Payload Example:**
```json
{
  "auto_archive_days": 14,
  "webhook_type": "discord",
  "webhook_url": "https://discord.com/api/webhooks/...",
  "webhook_events": ["critical", "high"]
}
```
**Response:** `200 OK`

---

### GET /api/v1/version
Returns the running Hub version.

**Response**
```json
{ "version": "1.0.0" }
```

### GET /api/v1/system/state
Returns the current armed/disarmed state. Disarmed hubs log events normally but suppress all push notifications.

**Response**
```json
{ "is_armed": true }
```

### PATCH /api/v1/system/state
Sets the armed state.

**Request**
```json
{ "is_armed": false }
```

**Response**
```json
{ "status": "success", "is_armed": false }
```

---

### PATCH /api/v1/system/password
Updates the Master Password. The current password must be provided and validated against the database. On success, all active sessions are terminated. Fails with `403 Forbidden` if the `HW_DASHBOARD_PASSWORD` environment variable is set.
**Requires Authentication:** Yes (UI Cookie)

**Payload:**
```json
{
  "current_password": "old_password123",
  "new_password": "new_password456"
}
```
**Response:** `200 OK`

---

---

### POST /api/v1/system/reset
Performs a full factory reset. Wipes all events, sensors, heartbeats, and configurations. The Hub will immediately revert to Setup mode. Terminates all active sessions.
**Requires Authentication:** Yes (UI Cookie)

**Payload:**
```json
{
  "password": "your_master_password"
}
```
**Response:** `200 OK`

**Errors**: 
  * 400 Bad Request if the payload is missing/malformed.
  * 401 Unauthorized if the password does not match.
---


## Node Provisioning

### POST /api/v1/tokens/generate
Generates a one-time pairing token for the node deployment wizard.
**Requires Authentication:** Yes (UI Cookie)

**Response**
```json
{
  "token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "expires_in": 900
}
```

### POST /api/v1/wizard/link
Public endpoint called by the wizard script to exchange a token for node credentials. The token is deleted immediately after use.

**Request**
```json
{
  "token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "alias": "production-db-node"
}
```

**Response**
```json
{
  "node_id": "b3f5c9e2",
  "node_key": "long_hex_string_...",
  "alias": "production-db-node",
  "timestamp": "2026-04-12T18:30:05Z"
}
```

## Compose Generation

### POST /api/v1/compose/generate
Generates a raw `docker-compose.yml` string for the target node based on selected sensors.
**Requires Authentication:** Yes (UI Cookie OR Node Key)

**Request**
```json
{
  "node_id": "node-12345678",
  "hub_endpoint": "https://hub.honeywire.local",
  "hub_key": "hub_key_abc",
  "sensors": [
    {
      "sensor_id": "alpha-node-01",
      "env_values": {},
      "manifest": { ... }
    }
  ]
}
```

**Response** — `application/x-yaml` payload containing the composed Docker instructions.

## Sensor Fleet

### GET /api/v1/sensors

Returns all registered sensors with live status and metadata. A sensor is considered `offline` if its last heartbeat arrived more than 90 seconds ago.

**Response example**
```json
[
  {
    "node_id": "node-12345678",
    "sensor_id": "tarpit-01",
    "last_seen": "2026-04-07 15:25:11",
    "metadata": {
      "agent_version": "1.0.0",
      "contract_version": "1.0",
      "sensor_type": "web_honeypot"
    },
    "status": "online",
    "is_silenced": false
  }
]
```

---

### PATCH /api/v1/sensors/{sensor_id}/silence

Silences or un-silences a sensor. Silenced sensors continue logging events to the database but never trigger push notifications, regardless of the system armed state.

**Request**
```json
{ "node_id": "node-12345678", "is_silenced": true }
```

**Response**
```json
{ "status": "success", "node_id": "node-12345678", "sensor_id": "tarpit-01", "is_silenced": true }
```

---

### DELETE /api/v1/sensors/{sensor_id}

Removes a sensor from active monitoring. Deletes the sensor record and its full heartbeat history. Events previously generated by this sensor are retained for auditing.

**Query parameters**

| Parameter | Values | Default | Description |
|---|---|---|---|
| `node_id` | any node ID | — | The node owning the sensor |

**Response**
```json
{ "status": "success", "message": "Sensor forgotten successfully" }
```

**Error** — `404 Not Found` if the sensor ID does not exist.

---

### GET /api/v1/uptime

Returns heatmap data used by the Fleet Health dashboard. The response is an array of per-sensor block sequences where each block represents a time window and carries a status of `up`, `degraded`, `down`, or `nodata`.

**Query parameters**

| Parameter | Values | Default |
|---|---|---|
| `timeframe` | `1H`, `24H`, `7D`, `30D` | `24H` |

| Timeframe | Blocks | Window per block |
|---|---|---|
| `1H` | 30 | 2 minutes |
| `24H` | 24 | 1 hour |
| `7D` | 7 | 1 day |
| `30D` | 30 | 1 day |

**Response example**
```json
[
  {
    "id": "tarpit-01",
    "node_id": "node-12345678",
    "name": "tarpit-01",
    "isOnline": true,
    "blocks": [
      { "status": "up", "timeLabel": "23 hours ago", "label": "Online" },
      { "status": "degraded", "timeLabel": "4 hours ago", "label": "Degraded (47/60 pings)" },
      { "status": "up", "timeLabel": "Current", "label": "Online (Live)" }
    ]
  }
]
```

---

## Events

### GET /api/v1/events/unread

Returns the count of active (non-archived), unread events. Intended for lightweight badge updates — does not return event payloads.

**Response**
```json
{ "count": 12 }
```

---

### GET /api/v1/events

Returns a list of events, newest first.

**Query parameters**

| Parameter | Values | Default | Description |
|---|---|---|---|
| `archived` | `true`, `false` | `false` | Whether to return archived or active events |
| `sensor_id` | any sensor ID | — | Filter to a specific sensor |
| `node_id` | any node ID | — | Filter to a specific node |

**Response** — array of event objects. See the event schema in the [WebSocket](#websocket) section for field reference.

---

### PATCH /api/v1/events/read

Marks all unread active events as read.

**Response**
```json
{ "status": "success" }
```

---

### PATCH /api/v1/events/{event_id}/read

Marks a single event as read.

**Response**
```json
{ "status": "success" }
```

---

### PATCH /api/v1/events/{event_id}/archive

Archives a single event, marking it as read and hiding it from the active event view.

**Response**
```json
{ "status": "success" }
```

---

### PATCH /api/v1/events/archive-all

Archives all currently active (non-archived) events in bulk.

**Response**
```json
{ "status": "success" }
```

---

### DELETE /api/v1/events

Permanently deletes all events from the database. This action is irreversible and is logged server-side with the caller's IP address.

**Query parameters**

| Parameter | Values | Default | Description |
|---|---|---|---|
| `dryrun` | `true`, `false` | `false` | If true, returns the count of events that *would* be deleted without executing the deletion. |

**Response (Standard)**
```json
{ 
  "status": "success",
  "dryrun": false
}
```

**Response (Dryrun)**
```json
{ 
  "status": "success",
  "dryrun": true,
  "would_delete": 4512
}
```

---

## Agent Endpoints

These endpoints are called by sensors, not the dashboard. Both require the configured database `hub_key`.

### POST /api/v1/heartbeat

Called by sensors every 30 seconds to signal they are alive and update their metadata. If this is the first heartbeat from a given `sensor_id`, the sensor is registered automatically and a `NEW_SENSOR` WebSocket message is broadcast to all dashboard clients.

**Request**
```json
{
  "node_id": "node-12345678",
  "sensor_id": "alpha-node-01",
  "metadata": {
    "agent_version": "1.0.0",
    "contract_version": "1.0",
    "sensor_type": "tarpit"
  }
}
```

**Response**
```json
{ "status": "alive" }
```

---

### POST /api/v1/event

Reports an intrusion event to the Hub. The Hub validates that the `contract_version` major number matches its own before accepting the event. On a mismatch, `426 Upgrade Required` is returned and the sensor should be updated.

If the Hub is armed and the reporting sensor is not silenced, a push notification is dispatched immediately via the configured notifiers (ntfy/Gotify/Discord/Slack).

**Request**
```json
{
  "contract_version": "1.0",
  "node_id": "node-12345678",
  "sensor_id": "core-dpi-engine",
  "event_trigger": "malformed_jwt_detected",
  "severity": "critical",
  "source": "104.28.19.12",
  "target": "Auth Gateway",
  "details": {
    "protocol": "TCP",
    "headers_stripped": true,
    "payload_sample": [
      "Authorization: Bearer eyJhbG... [TRUNCATED]",
      "User-Agent: curl/7.64.1"
    ],
    "action_taken": "logged"
  }
}
```

**Severity values:** `info`, `low`, `medium`, `high`, `critical`

**Response**
```json
{ "status": "success" }
```

**Errors**

| Status | Meaning |
|---|---|
| `400 Bad Request` | Malformed JSON or missing `contract_version` |
| `426 Upgrade Required` | Major version mismatch between sensor and Hub |

---

## Dashboard Auth

### POST /login

Authenticates a dashboard session. On success, sets an HTTP-only `hw_auth` cookie valid for 30 days. Repeated failed attempts from the same IP are rate-limited: 10 failures triggers a 15-minute lockout.

**Request**
```json
{ "password": "your_password" }
```

**Response** — `200 OK` with cookie on success, `401 Unauthorized` on wrong password, `429 Too Many Requests` during lockout.

---

### POST /logout

Invalidates the current session token and clears the `hw_auth` cookie.

**Response** — Redirects to `/`.