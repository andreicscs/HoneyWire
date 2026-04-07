# HoneyWire Hub API Reference

This document describes the HTTP API for the HoneyWire Hub backend.

## Authentication

### UI routes
- Requires an HTTP-only cookie-based session in `hw_auth` with a valid login token.
- If `DASHBOARD_PASSWORD` is set in the environment, all UI endpoints validate this cookie via the `verify_ui_auth` dependency.

### Agent routes
- Requires the shared secret passed via one of two headers:
  - `X-Api-Key: <API_SECRET>`
  - `Authorization: Bearer <API_SECRET>`
- Validated via the `verify_agent_auth` dependency. Called by sensors for heartbeat and event reporting.

---

## System endpoints

### GET /api/v1/system/state
- Returns the current armed/disarmed state.
- Response:
```json
{ "is_armed": true }
```

### PATCH /api/v1/system/state
- Toggles the system's armed state.
- Request body:
```json
{ "is_armed": false }
```
- Response:
```json
{ "status": "success", "is_armed": false }
```

### GET /api/v1/version
- Returns the current app version from the `VERSION` file or the `HW_VERSION` environment variable override. *(Note: This endpoint is public and does not require authentication).*
- Response:
```json
{ "version": "1.0.0" }
```

---

## Sensor fleet

### GET /api/v1/sensors
- Lists registered sensors, their status, and last-seen timestamps.
- Response example:
```json
[
  {
    "sensor_id": "alpha-node-01",
    "sensor_type": "tarpit",
    "last_seen": "2026-04-02 15:25:11",
    "details": {"version": "1.0.0", "mode": "hold"},
    "status": "online"
  }
]
```

---

## Events

### GET /api/v1/events
- Returns events in descending chronological order.
- Example response:
```json
[
  {
    "contract_version": "1.0.0",
    "id": 123,
    "timestamp": "2026-04-02 15:25:12",
    "sensor_id": "alpha-node-01",
    "sensor_type": "tarpit",
    "event_type": "tcp_connection",
    "severity": "high",
    "source": "10.0.0.5",
    "target": "Port 2222",
    "action_taken": "hold",
    "details": {"duration_sec": 12.3, "payload_sample": ["..."], "total_lines": 5},
    "is_read": false
  }
]
```

### PATCH /api/v1/events/read
- Marks all unread events as read in bulk.
- Response:
```json
{ "status": "success" }
```

### PATCH /api/v1/events/{event_id}/read
- Marks a specific event as read.
- Response:
```json
{ "status": "success" }
```

### DELETE /api/v1/events
- Clears all events in the database.
- Response:
```json
{ "status": "success" }
```

---

## Agent endpoints

### POST /api/v1/heartbeat
- Sensors call this every 30 seconds to maintain an "online" status.
- Request:
```json
{
  "sensor_id": "alpha-node-01",
  "sensor_type": "tarpit",
  "metadata": {"version": "1.0.0", "mode": "hold"}
}
```
- Response:
```json
{ "status": "alive" }
```

### POST /api/v1/event
- Sensors report intrusion events here. Triggers background notification tasks if the system is armed.
- *Note: Extraneous telemetry must be passed in the `details` object to satisfy the Pydantic schema.*
- Request:
```json
{
  "contract_version": "1.0.0",
  "sensor_id": "alpha-node-01",
  "sensor_type": "tarpit",
  "event_type": "tcp_connection",
  "severity": "high",
  "source": "10.0.0.5",
  "target": "Port 2222",
  "action_taken": "hold",
  "details": {
    "duration_sec": 12.3,
    "payload": ["sudo rm -rf /"]
  }
}
```
- Response:
```json
{ "status": "success" }
```

---

## UI login flow

### POST /login
- Authenticates the dashboard user.
- Request body:
```json
{ "password": "my_secure_password" }
```
- Response: Returns a JSON `{"status": "ok"}` and sets the HTTP-only `hw_auth` cookie for 30 days.

### GET /logout
- Invalidates the current session, deletes the `hw_auth` cookie, and redirects to the login screen.