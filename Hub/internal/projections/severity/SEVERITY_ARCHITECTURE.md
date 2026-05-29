# Severity Analytics Projection Architecture Guide

This document describes the current severity analytics architecture used by the dashboard.
The system has been refactored away from frontend-side event aggregation and now uses backend-generated analytics projections with contextual filtering.

# Architecture Overview

## Backend Projection Model
```
Raw Events
    ↓
Backend Filtering (timeframe/node/sensor/archive)
    ↓
Backend Severity Aggregation
    ↓
Flat Severity Projection DTO
    ↓
Frontend Chart Rendering
```

# Architectural Goals

- Eliminate frontend traversal of raw event arrays.
- Push filtering and aggregation to the backend/database layer.
- Keep widgets lightweight and rendering-focused.
- Use immutable backend projection snapshots instead of frontend analytics recomputation.
- Support contextual filtering by:
  - timeframe
  - node
  - sensor
  - archive mode
- Maintain websocket compatibility through projection invalidation/refetch.

---


# Directory Structure

```txt
internal/
├── projections/
│   └── severity/
│       ├── dto.go
│       ├── calculator.go
│       └── projection.go
│
├── api/
│   └── analytics.go
```

---

# Backend Components

# 1. DTO Layer (`dto.go`)

Defines the flat API contract returned to the frontend.

## Example

```go
type SeverityProjection struct {
    Timeframe string `json:"timeframe"`

    Total    int `json:"total"`
    Critical int `json:"critical"`
    High     int `json:"high"`
    Medium   int `json:"medium"`
    Low      int `json:"low"`
    Info     int `json:"info"`
}
```

## Design Rules

- Flat structure only.
- No nested traversal required by the frontend.
- Represents a complete immutable analytics snapshot.

---

# 2. Calculator Layer (`calculator.go`)

Contains pure aggregation logic.

## Responsibilities

- Iterate filtered events.
- Count severities.
- Return aggregated counts.

## Rules

- No HTTP access.
- No database access.
- No Vue/frontend concerns.
- Pure deterministic aggregation only.

---

# 3. Projection Layer (`projection.go`)

Orchestrates the analytics pipeline.

## Responsibilities

1. Accept filtering context:
   - timeframe
   - node
   - sensor
   - archive visibility

2. Fetch filtered events from repository/storage layer.

3. Pass events into calculator.

4. Return the composed projection DTO.

---

# 4. API Handler (`analytics_handler.go`)

Thin HTTP adapter.

## Responsibilities

- Parse query parameters.
- Invoke projection builder.
- Serialize JSON response.

## Supported Query Parameters

| Parameter | Description |
|---|---|
| `timeframe` | Time window (`alltime`, `24h`, etc.) |
| `node` | Filter by node ID |
| `sensor` | Filter by sensor ID |
| `archive` | Include archived events |

---


# Frontend Integration

## Data Flow

### Initial Load / Filter Changes

1. Widget watches:
   - selected node
   - selected sensor
   - archive mode

2. Widget triggers:

```js
eventsStore.fetchSeverityProjection(...)
```

3. Store:
   - aborts stale requests
   - fetches new projection snapshot
   - replaces `severityProjection`

4. Widget reacts to new projection reference and updates chart.

---

## WebSocket Flow

### Event Arrival

1. New websocket event arrives.

2. Events store determines whether the event affects the currently active filter context.

3. If relevant:
   - invalidate projection
   - refetch `/api/v1/analytics/severity`

4. Store replaces snapshot.

5. Chart rerenders.

---

# Important Architectural Rule

The frontend does NOT:
- increment severity counters
- aggregate raw events
- mutate analytics state manually

The backend remains the authoritative analytics source.

WebSockets only trigger invalidation/refetch behavior.

---

# Store Responsibilities

## Events Store

The store owns:

- network transport
- request cancellation
- projection state
- websocket invalidation
- contextual fetching

## Example Fetch Flow

```js
async fetchSeverityProjection(timeframe, nodeId, sensorId)
```

### Features

- Uses `AbortController`
- Prevents stale responses
- Replaces immutable snapshot references

---

# Widget Responsibilities

# SeverityChart.vue

The widget is intentionally lightweight.

## Responsibilities

- Trigger contextual projection fetches.
- Render projection values.
- Update Chart.js instance.

## Forbidden Responsibilities

The widget must NOT:

- traverse raw event arrays
- aggregate severities
- compute percentages from event entities
- maintain live analytics state

---

# Current Reactivity Model

The widget watches:

```js
watch(severityProjection, updateData)
```

Because:
- projections are immutable snapshots
- store replaces object references
- deep reactivity is unnecessary

---

# Performance Characteristics

## Backend

Efficient because:
- filtering happens before aggregation
- database applies WHERE clauses
- aggregation runs on reduced datasets

## Frontend

Efficient because:
- chart consumes flat primitive values
- no array traversal
- no deep watchers
- no repeated reductions
- no raw event recomputation

---

# API Endpoint

### GET /api/v1/events/severity

Returns severity distribution analytics for the fleet.

---

## Query Parameters

| Parameter | Description |
|---|---|
| `timeframe` | Timeframe filter (e.g. `alltime`, `24H`) (default: `alltime`) |
| `node` | Filter by node ID (optional) |
| `sensor` | Filter by sensor ID (optional) |
| `viewingArchive` | Include archived events (`true`, `1`, `false`, or `0`) (default: `false`) |

---

## Example Requests

### Global Severity Distribution

```txt
/api/v1/events/severity
```

### Severity Distribution For a Specific Node

```txt
/api/v1/events/severity?node=node-1
```

### Severity Distribution For a Specific Sensor

```txt
/api/v1/events/severity?node=node-1&sensor=sensor-3
```

### Archived Events Only

```txt
/api/v1/events/severity?viewingArchive=true
```

### Combined Filtering

```txt
/api/v1/events/severity?timeframe=24H&node=node-1&sensor=sensor-3&viewingArchive=true
```

---

## Example Response

```json
{
  "timeframe": "alltime",
  "total": 150,
  "critical": 10,
  "high": 40,
  "medium": 50,
  "low": 30,
  "info": 20
}
```

---

# Current Architectural Principles

## Backend Owns Analytics Truth

The frontend never computes severity analytics from raw events.

---

## Immutable Projection Snapshots

Projection objects are treated as disposable snapshots:
- fetch
- replace
- render

No in-place mutation.

---

## Contextual Fetching

Analytics projections are contextual:
- node selection changes projection
- sensor selection changes projection
- archive mode changes projection

---

## Lightweight Widgets

Widgets are rendering-focused, not analytics-focused.

---