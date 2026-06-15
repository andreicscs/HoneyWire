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
{ "requiresSetup": true }
```

### POST /api/v1/setup

Completes first-time hub setup and stores the hashed admin password, hub endpoint, and hub API key.
If `HW_DASHBOARD_PASSWORD` is set, setup is blocked.

**Request:**
```json
{
  "password": "super_secure_password123",
  "hubEndpoint": "https://honeywire.my-domain.com",
  "hubKey": "hw_sk_randomstring"
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

Fetches the sensor manifest catalog from `RegistryURL` or the default public manifest registry.

### POST /api/v1/compose/generate

Generates a `docker-compose.yml` preview from the selected sensor manifests and UI-provided environment values.

**Request:**
```json
{
  "hubEndpoint": "https://honeywire.my-domain.com",
  "hubKey": "hub_key_abc",
  "sensors": [
    {
      "sensorId": "alpha-node-01",
      "envValues": {
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

#### GET /api/v1/nodes/{nodeId}

Returns details for a single node, including installed sensors and per-sensor event counts.

#### PATCH /api/v1/nodes/{nodeId}

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

#### DELETE /api/v1/nodes/{nodeId}

Deletes a node and cascades to remove its sensors, event history, and heartbeat records.

#### POST /api/v1/nodes/{nodeId}/sensors

Adds a sensor to a node and flags the node as pending configuration sync.

**Request:**
```json
{
  "sensorId": "hw-tcp-tarpit",
  "customName": "TCP Tarpit",
  "configValues": {
    "HW_SEVERITY": "high"
  }
}
```

#### PUT /api/v1/nodes/{nodeId}/sensors/{sensorId}

Updates a sensor's custom name and config values and marks the parent node pending sync.

#### DELETE /api/v1/nodes/{nodeId}/sensors/{sensorId}

Removes a sensor from the node and marks the node pending sync.

#### PATCH /api/v1/nodes/{nodeId}/sensors/{sensorId}/silence

Toggles a sensor's silence state. Silenced sensors still log events but suppress push notifications.

**Request:**
```json
{ "isSilenced": true }
```

**Response:**
```json
{
  "status": "success",
  "sensorId": "hw-tcp-tarpit",
  "isSilenced": true
}
```

### Configuration and System Settings

#### GET /api/v1/config

Returns runtime configuration loaded from SQLite. 

**Response Schema:**
| Field | Type | Description |
|-------|------|-------------|
| `hubEndpoint` | `string` | The fully qualified URL of the Hub (e.g., `https://hub.example.com`). |
| `autoArchiveDays` | `integer` | Days before active events are moved to the archive. `0` = Keep forever. |
| `autoPurgeDays` | `integer` | Days before archived events are permanently deleted. `0` = Keep forever. |
| `webhookType` | `string` | The active push notification provider (`ntfy`, `gotify`, `discord`, `slack`, `none`). |
| `webhookUrl` | `string` | The target URL for the selected webhook provider. |
| `webhookEvents` | `string[]` | Array of severity levels that trigger a notification (e.g., `["critical", "high"]`). |
| `siemAddress` | `string` | Address for RFC5424 syslog forwarding (e.g., `10.0.0.50:514`). Empty if disabled. |
| `siemProtocol` | `string` | Protocol for SIEM forwarding (`tcp`, `udp`). |

**Response:**
```json
{
  "hubEndpoint": "https://honeywire.my-domain.com",
  "autoArchiveDays": 90,
  "autoPurgeDays": 180,
  "webhookType": "none",
  "webhookUrl": "",
  "webhookEvents": [],
  "siemAddress": "",
  "siemProtocol": "tcp"
}
```

#### PATCH /api/v1/config

Updates runtime settings. Only supported fields are applied. This endpoint also hot-reloads webhook and SIEM configuration.

**Accepted Fields:**
- `hubEndpoint`
- `autoArchiveDays`
- `autoPurgeDays`
- `webhookType` (`ntfy`, `gotify`, `discord`, `slack`, `none`)
- `webhookUrl`
- `webhookEvents`
- `siemAddress`
- `siemProtocol` (`tcp`, `udp`)

**Request example:**
```json
{
  "autoArchiveDays": 14,
  "webhookType": "discord",
  "webhookUrl": "https://discord.com/api/webhooks/...",
  "webhookEvents": ["critical", "high"]
}
```

#### GET /api/v1/system/state

Returns whether the Hub is armed.

**Response:**
```json
{ "isArmed": true }
```

#### PATCH /api/v1/system/state

Sets the armed/disarmed state.

**Request:**
```json
{ "isArmed": false }
```

#### PATCH /api/v1/system/password

Changes the dashboard admin password. When `HW_DASHBOARD_PASSWORD` is set, this operation is forbidden.

**Request:**
```json
{
  "currentPassword": "oldPassword",
  "newPassword": "newPassword"
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
- `nodeId` — filter by node ID
- `sensorId` — filter by sensor ID

### PATCH /api/v1/events/read

Marks all active events as read.

### PATCH /api/v1/events/{eventId}/read

Marks a single event as read.

### PATCH /api/v1/events/{eventId}/archive

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

**Example Response:**
```json
{
  "timeframe": "24H",
  "total": 80,
  "critical": 5,
  "high": 12,
  "medium": 25,
  "low": 8,
  "info": 30
}
```

### GET /api/v1/events/velocity

Returns time-bucketed velocity analytics for the fleet.

**Query parameters:**
- `timeframe` — timeframe filter (`1H`, `24H`, `7D`, `30D`) (default: `24H`)
- `nodeId` — filter by node ID (optional)
- `sensorId` — filter by sensor ID (optional)
- `archived` — include archived events (`true` or `false`) (default: `false`)

**Example Response:**
```json
{
  "timeframe": "24H", 
  "bucketSizeMs": 3600000,
  "generatedAt": 1716552000000,
  "bucketTimestamps": [1716465600000, 1716469200000],
  "labels": ["-23h", "-22h", "Now"],
  "exactTimes": ["May 23, 08:00 AM", "May 23, 09:00 AM"],
  "series": {
    "critical": [0, 1, 0],
    "high": [2, 0, 5],
    "medium": [0, 0, 0],
    "low": [0, 0, 0],
    "info": [1, 3, 2]
  },
  "recentEventCount": 14
}
```

## Uptime and Health

### GET /api/v1/uptime

Returns uptime blocks for fleet health charts.

**Query parameters:**
- `timeframe` — `1H`, `24H`, `7D`, `30D` (default: `24H`)

**Example Response:**
```json
{
  "timeframe": "24H", 
  "generatedAt": "2024-05-24T10:00:00Z",
  "summary": {
    "overallUptime": 98.5
  },
  "groups": [
    {
      "nodeId": "node-alpha",
      "nodeAlias": "Production Web Server",
      "worstStatus": "degraded",
      "sensors": [
        {
          "sensorId": "hw-tcp-tarpit",
          "displayName": "TCP Tarpit",
          "status": "up",
          "isSilenced": false,
          "blocks": [
            { "status": "up", "label": "Online", "timeLabel": "-23h" },
            { "status": "up", "label": "Online", "timeLabel": "-22h" },
            { "status": "up", "label": "Online", "timeLabel": "Current" }
          ]
        },
        {
          "sensorId": "hw-file-canary",
          "displayName": "File Integrity Monitor",
          "status": "degraded",
          "isSilenced": false,
          "blocks": [
            { "status": "up", "label": "Online", "timeLabel": "-23h" },
            { "status": "degraded", "label": "Degraded (50/60 pings)", "timeLabel": "-22h" },
            { "status": "up", "label": "Online", "timeLabel": "Current" }
          ]
        }
      ]
    },
    {
      "nodeId": "node-beta",
      "nodeAlias": "Dev Database",
      "worstStatus": "down",
      "sensors": [
        {
          "sensorId": "hw-network-scan",
          "displayName": "Network Scan Detector",
          "status": "down",
          "isSilenced": true,
          "blocks": [
            { "status": "down", "label": "Offline", "timeLabel": "-23h" },
            { "status": "down", "label": "Offline", "timeLabel": "-22h" },
            { "status": "down", "label": "Offline", "timeLabel": "Current" }
          ]
        }
      ]
    }
  ]
}
```

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
  "sensorId": "alpha-node-01",
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
> 📖 **[View the Universal Event Standard Contract](./architecture/dataContracts.md#1-the-universal-event-standard)**

**Severity values:** `info`, `low`, `medium`, `high`, `critical`

**Response:**
```json
{ "status": "success" }
```

**Errors:**
- `400 Bad Request` for invalid JSON or missing required fields.
- `426 Upgrade Required` if the sensor major version does not match the Hub.
Required` if the sensor major version does not match the Hub.
