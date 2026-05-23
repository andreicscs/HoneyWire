# HoneyWire Hub — API Reference

All Hub routes are prefixed with `/api/v1` unless otherwise noted.

## Authentication

HoneyWire uses two separate authentication methods.

### Dashboard (UI) routes

Dashboard endpoints are protected by an HTTP-only session cookie named `hw_auth`.
The cookie is issued by `POST /login` and is valid for 30 days.

### Agent (sensor) routes

Sensor/agent endpoints are authenticated with a node API key using the Authorization header:

```http
Authorization: Bearer <HW_NODE_KEY>
```

## Public Endpoints

### GET /api/v1/version

Returns the running Hub version.

**Response:**
```json
{ "version": "2.0.0" }
```

### POST /login

Authenticates a dashboard user and sets the `hw_auth` cookie.

**Request:**
```json
{ "password": "your_password" }
```

**Response:** `200 OK` with cookie on success.

**Errors:** `401 Unauthorized` for invalid credentials, `429 Too Many Requests` after repeated failures.

### POST /logout

Invalidates the current dashboard session and redirects the client to `/`.

**Response:** `303 See Other`

### GET /api/v1/setup/status

Returns whether the Hub requires initial setup.
When `HW_DASHBOARD_PASSWORD` is set, setup is considered locked and this endpoint returns `requires_setup: false`.

**Response:**
```json
{ "requires_setup": true }
```

### POST /api/v1/setup

Completes first-time hub setup and stores the hashed admin password, hub endpoint, and hub API key.
If `HW_DASHBOARD_PASSWORD` is set, setup is blocked.

**Request:**
```json
{
  "password": "super_secure_password123",
  "hub_endpoint": "https://honeywire.my-domain.com",
  "hub_key": "hw_sk_randomstring"
}
```

**Response:** `200 OK`

**Errors:** `403 Forbidden` if setup is locked or already complete.

## Dashboard Endpoints (UI Cookie Required)

These endpoints require a valid `hw_auth` session cookie.

### GET /api/v1/ws

Upgrades the connection to a WebSocket for realtime dashboard updates.

All messages use the envelope:

```json
{ "type": "<MESSAGE_TYPE>", "payload": { ... } }
```

Common message types:

- `NEW_EVENT` — a sensor reported an event.
- `NEW_SENSOR` — a new sensor was discovered.
- `SENSOR_HEARTBEAT` — a sensor heartbeat was received.
- `NODE_SYNCED` — a node reported that pending config was applied.
- `SILENCE_SENSOR` — a sensor silence state changed.
- `SYNC_CHARTS` — instructs UIs to refresh chart data.

### GET /api/v1/manifests

Fetches the sensor manifest catalog from `HW_MANIFEST_URL` or the default public manifest registry.

### POST /api/v1/compose/generate

Generates a `docker-compose.yml` preview from the selected sensor manifests and UI-provided environment values.

**Request:**
```json
{
  "hub_endpoint": "https://hub.honeywire.local",
  "hub_key": "hub_key_abc",
  "sensors": [
    {
      "sensor_id": "alpha-node-01",
      "env_values": {
        "HW_SEVERITY": "medium"
      },
      "manifest": { ... }
    }
  ]
}
```

**Response:** `application/x-yaml` compose payload.

### Nodes and Sensors

#### POST /api/v1/nodes

Creates a new node entry and returns the generated node credentials.

**Request:**
```json
{
  "alias": "production-db-node",
  "tags": ["database", "prod"]
}
```

**Response:**
```json
{
  "api_key": "hw_key_abcd1234...",
  "alias": "production-db-node"
}
```

#### GET /api/v1/nodes

Returns all registered nodes and their installed sensors. Each node includes current status, heartbeat metadata, and pending config state.

#### GET /api/v1/nodes/{id}

Returns details for a single node, including installed sensors and per-sensor event counts.

#### PATCH /api/v1/nodes/{id}

Updates a node's alias, tags, public IP, or private IP.

**Request:**
```json
{
  "alias": "prod-db-node",
  "tags": ["database","primary"],
  "publicIp": "198.51.100.12",
  "privateIp": "10.0.0.12"
}
```

#### DELETE /api/v1/nodes/{id}

Deletes a node and cascades to remove its sensors, event history, and heartbeat records.

#### POST /api/v1/nodes/{id}/sensors

Adds a sensor to a node and flags the node as pending configuration sync.

**Request:**
```json
{
  "sensor_id": "hw-tcp-tarpit",
  "custom_name": "TCP Tarpit",
  "config_values": {
    "HW_SEVERITY": "high"
  }
}
```

#### PUT /api/v1/nodes/{id}/sensors/{sensor_id}

Updates a sensor's custom name and config values and marks the parent node pending sync.

#### DELETE /api/v1/nodes/{id}/sensors/{sensor_id}

Removes a sensor from the node and marks the node pending sync.

#### PATCH /api/v1/nodes/{id}/sensors/{sensor_id}/silence

Toggles a sensor's silence state. Silenced sensors still log events but suppress push notifications.

**Request:**
```json
{ "is_silenced": true }
```

**Response:**
```json
{
  "status": "success",
  "sensor_id": "hw-tcp-tarpit",
  "is_silenced": true
}
```

### Configuration and System Settings

#### GET /api/v1/config

Returns runtime configuration loaded from SQLite.

**Response:**
```json
{
  "hub_endpoint": "https://honeywire.my-domain.com",
  "hub_key": "hw_sk_randomstring",
  "auto_archive_days": 90,
  "auto_purge_days": 180,
  "webhook_url": "",
  "webhook_type": "none",
  "webhook_events": [],
  "siem_address": "",
  "siem_protocol": "syslog"
}
```

#### PATCH /api/v1/config

Updates runtime settings. Only supported fields are applied. This endpoint also hot-reloads webhook and SIEM configuration.

**Supported fields:**
- `hub_endpoint`
- `hub_key`
- `auto_archive_days`
- `auto_purge_days`
- `webhook_url`
- `webhook_type` (`ntfy`, `gotify`, `discord`, `slack`, `none`)
- `webhook_events`
- `siem_address`
- `siem_protocol` (`tcp`, `udp`)

**Request example:**
```json
{
  "auto_archive_days": 14,
  "webhook_type": "discord",
  "webhook_url": "https://discord.com/api/webhooks/...",
  "webhook_events": ["critical", "high"]
}
```

#### GET /api/v1/system/state

Returns whether the Hub is armed.

**Response:**
```json
{ "is_armed": true }
```

#### PATCH /api/v1/system/state

Sets the armed/disarmed state.

**Request:**
```json
{ "is_armed": false }
```

#### PATCH /api/v1/system/password

Changes the dashboard admin password. When `HW_DASHBOARD_PASSWORD` is set, this operation is forbidden.

**Request:**
```json
{
  "current_password": "old_password",
  "new_password": "new_password"
}
```

#### POST /api/v1/system/reset

Performs a full factory reset of the runtime database, clearing events, sensors, heartbeats, and config.
Requires the current admin password.

**Request:**
```json
{ "password": "your_master_password" }
```

**Response:** `200 OK`

## Fleet and Events

### GET /api/v1/events/unread

Returns the count of active, unread events.

**Response:**
```json
{ "count": 12 }
```

### GET /api/v1/events

Returns a list of events, newest first.

**Query parameters:**
- `archived` — `true` or `false` (default: `false`)
- `node_id` — filter by node ID
- `sensor_id` — filter by sensor ID

### PATCH /api/v1/events/read

Marks all active events as read.

### PATCH /api/v1/events/{event_id}/read

Marks a single event as read.

### PATCH /api/v1/events/{event_id}/archive

Archives a single event and marks it as read.

### PATCH /api/v1/events/archive-all

Archives all active events.

### DELETE /api/v1/events

Deletes all events permanently.

**Query parameters:**
- `dryrun=true` — return the count of deletable events without deleting them.

**Response (dryrun):**
```json
{
  "status": "success",
  "dryrun": true,
  "would_delete": 4512
}
```

**Response:**
```json
{ "status": "success", "dryrun": false }
```

### GET /api/v1/events/severity

Returns severity distribution analytics for the fleet.

**Query parameters:**
- `timeframe` — timeframe filter (e.g. `alltime`, `24H`) (default: `alltime`)
- `node` — filter by node ID (optional)
- `sensor` — filter by sensor ID (optional)
- `viewingArchive` — `true`, `1`, `false`, or `0` (default: `false`)

## Uptime and Health

### GET /api/v1/uptime

Returns uptime blocks for fleet health charts.

**Query parameters:**
- `timeframe` — `1H`, `24H`, `7D`, `30D` (default: `24H`)

## Node Compose

### GET /api/v1/nodes/compose

Returns a generated `docker-compose.yml` for a node based on the node's current installed sensors.

**Authentication:** `Authorization: Bearer <HW_NODE_KEY>`

This endpoint is read-only for node deployment: it returns the current or requested `HW_CONFIG_REV` without marking the node as synced. The node is marked synced only when its heartbeat reports the expected revision.

## Agent Endpoints

These endpoints are used by sensor agents and require the node API key bearer token.

### POST /api/v1/heartbeat

Reports a sensor heartbeat and updates runtime metadata.

**Request:**
```json
{
  "sensor_id": "alpha-node-01",
  "metadata": {
    "agent_version": "1.0.0",
    "contract_version": "1.0",
    "sensor_type": "tarpit",
    "HW_CONFIG_REV": "rev_abc123"
  }
}
```

**Response:**
```json
{ "status": "alive" }
```

### POST /api/v1/event

Reports an intrusion event to the Hub.

**Request:**
```json
{
  "contract_version": "1.0",
  "sensor_id": "core-dpi-engine",
  "event_trigger": "malformed_jwt_detected",
  "severity": "critical",
  "source": "104.28.19.12",
  "target": "Auth Gateway",
  "details": {
    "protocol": "TCP",
    "action_taken": "logged"
  }
}
```

**Severity values:** `info`, `low`, `medium`, `high`, `critical`

**Response:**
```json
{ "status": "success" }
```

**Errors:**
- `400 Bad Request` for invalid JSON or missing required fields.
- `426 Upgrade Required` if the sensor major version does not match the Hub.
