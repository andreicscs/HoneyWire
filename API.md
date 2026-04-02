# HoneyWire Hub API Reference

This document describes the HTTP API for the HoneyWire Hub backend.

## Authentication

### UI routes
- Requires cookie-based session in `hw_auth` with a valid login.
- If `DASHBOARD_PASSWORD` is set, all UI endpoints use `verify_ui_auth`.

### Agent routes
- Requires header `x-api-key: <API_SECRET>` (shared secret).
- Called by sensors for heartbeat and event reporting.

---

## System endpoints

### GET /api/v1/system/state
- Returns current armed/disarmed state.
- Response:
```json
{ "is_armed": true }
```

### PATCH /api/v1/system/state
- Toggle system armed state.
- Request body:
```json
{ "is_armed": false }
```
- Response:
```json
{ "status": "success", "is_armed": false }
```

### GET /api/v1/version
- Returns current app version from `VERSION` file or env.
- Response:
```json
{ "version": "1.0.0" }
```

---

## Sensor fleet

### GET /api/v1/sensors
- Lists registered sensors and their last-seen times.
- Response example:
```json
[
  {
    "sensor_id": "alpha-node-01",
    "sensor_type": "tarpit",
    "last_seen": "2026-04-02 15:25:11",
    "metadata": {"version":"1.0.0","mode":"hold"},
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
- Marks all unread events as read.
- Response:
```json
{ "status": "success" }
```

### PATCH /api/v1/events/{event_id}/read
- Marks one event as read.
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
- Sensors call this every 30 seconds.
- Request:
```json
{
  "sensor_id": "alpha-node-01",
  "sensor_type": "tarpit",
  "metadata": {"version": "1.0.0", "mode":"hold"}
}
```
- Response:
```json
{ "status": "alive" }
```

### POST /api/v1/event
- Sensors report intrusion events.
- Request:
```json
{
  "sensor_id": "alpha-node-01",
  "sensor_type": "tarpit",
  "event_type": "tcp_connection",
  "severity": "high",
  "source": "10.0.0.5",
  "target": "Port 2222",
  "action_taken": "hold",
  "details": {
    "duration_sec": 12.3,
    "payload_sample": ["sudo rm -rf /"],
    "total_lines": 7
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
- Authenticate dashboard user.
- Body:
```json
{ "password": "my_secure_password" }
```
- Returns cookie `hw_auth`.

### GET /logout
- Clears session cookie and redirects to login screen.

---

## Notes
- API error handling is default `HTTPException` style (401, 422, etc.).
- Event `details` supports any JSON object (strings, arrays, numbers), displayed in UI modal.
- `is_read` is used by the UI to highlight unread alerts.
