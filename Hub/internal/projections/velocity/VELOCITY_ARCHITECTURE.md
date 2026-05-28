# Threat Velocity Analytics Projection Architecture Guide

This document describes the threat velocity analytics architecture used by the dashboard's line chart.
The system has been refactored away from frontend-side event array traversal and time-bucketing, and now uses backend-generated analytics projections with contextual filtering.

# Architecture Overview

## Backend Projection Model
```
Raw Events
    ↓
Backend Filtering (timeframe/node/sensor/archive)
    ↓
Backend Time-Bucketing & Aggregation
    ↓
Time-Series Velocity Projection DTO
    ↓
Frontend Chart Rendering
```

# Architectural Goals

- Eliminate frontend traversal and date-math over raw event arrays.
- Push filtering, time-bucketing, and aggregation to the backend/database layer.
- Keep widgets lightweight and strictly focused on chart rendering.
- Ensure deterministic time boundaries using precise timestamp math.
- Support contextual filtering by:
  - timeframe
  - node
  - sensor
  - archive mode
- Maintain precise real-time synchronization through smart rollover timeouts and websocket invalidation.

---

# Directory Structure

```txt
internal/
├── projections/
│   └── velocity/
│       ├── dto.go
│       ├── calculator.go
│       ├── projection.go
│       └── VELOCITY_ARCHITECTURE.md
│
├── api/
│   └── analytics.go (GetVelocityAnalytics)
```

---

# Backend Components

# 1. DTO Layer (`dto.go`)

Defines the flat API contract returned to the frontend. Unlike Severity, Velocity requires time-series arrays.

## Example

```go
type ThreatVelocityProjection struct {
    Timeframe        string             `json:"timeframe"`
    BucketSizeMs     int64              `json:"bucketSizeMs"`
    GeneratedAt      int64              `json:"generatedAt"`

    BucketTimestamps []int64            `json:"bucketTimestamps"` // Raw epoch timestamps
    Labels           []string           `json:"labels"`            // X-axis labels (e.g., "Now", "-2m")
    ExactTimes       []string           `json:"exactTimes"`       // Tooltip timestamps

    Series           map[string][]int   `json:"series"`            // Map of severity to bucketed counts
    RecentEventCount int                `json:"recentEventCount"`
}
```

## Design Rules

- Pre-calculated axis arrays (`Labels`, `ExactTimes`).
- Ready-to-use dataset map (`Series`).
- Contains all necessary metadata for the frontend to know *when* the data expires (`BucketSizeMs`, `GeneratedAt`).

---

# 2. Calculator Layer (`calculator.go`)

Contains pure time-bucketing and aggregation logic.

## Responsibilities

- Determine bucket counts and sizes based on `timeframe`.
- Generate precise start times and human-readable labels for each bucket.
- Iterate filtered events, parse timestamps, and place them in the correct bucket index.
- Return the populated multi-series struct.

## Rules

- Pure deterministic aggregation only.
- Accepts `time.Now()` as an explicit parameter for testability and deterministic generation.

---

# 3. Projection Layer (`projection.go`)

Orchestrates the velocity pipeline.

## Responsibilities

1. Accept filtering context (timeframe, node, sensor, archive mode).
2. Fetch events from the store.
3. Pass events and `time.Now().UTC()` to the calculator.
4. Return the composed `ThreatVelocityProjection`.

---

# Frontend Integration

## Data Flow

### Initial Load / Filter Changes

1. Widget watches contextual state.
2. Widget triggers `eventsStore.fetchThreatVelocityProjection(...)`.
3. Store replaces `threatVelocityProjection` with a new immutable snapshot.
4. Widget reacts to snapshot replacement and updates Chart.js.

---

## Real-Time Synchronization (The Rollover Ticker)

Because Threat Velocity is a time-series chart, the buckets must shift forward at exact intervals, even if no new events arrive.

Instead of a naive `setInterval`, the widget dynamically calculates the precise delay until the next time boundary based on `bucket_size_ms`.

```js
const scheduleNextRollover = () => {
    // Clear any existing timeout so they don't pile up
    if (rolloverTimeout) clearTimeout(rolloverTimeout)
    if (!projection.value?.bucket_size_ms) return

    const bucketMs = projection.value.bucket_size_ms
    const now = Date.now()
    
    // Find the exact millisecond of the next bucket boundary
    const nextBoundary = Math.ceil(now / bucketMs) * bucketMs
    
    // Add a tiny 100ms buffer to ensure we safely crossed the time boundary 
    // before asking the backend for the new data.
    const delay = nextBoundary - now + 100

    rolloverTimeout = setTimeout(() => {
        // When the boundary hits, request fresh data
        fetchContextualProjection()
        
        // Note: We don't recursively call scheduleNextRollover() here.
        // Why? Because fetchContextualProjection() will fetch a new projection,
        // which will trigger the watcher below, which will safely schedule the NEXT tick.
    }, delay)
}

watch(
    () => projection.value?.generated_at,
    (newVal) => {
        if (newVal) {
            updateData();
            scheduleNextRollover();
        }
    }
)
```

## WebSocket Flow

### Event Arrival

1. New websocket event arrives.
2. Store determines if event affects current filter context.
3. If relevant, store calls `invalidateThreatVelocityProjection()`.
4. Widget watches invalidation state. If not viewing the archive, it triggers `fetchContextualProjection()`.

---

# Important Architectural Rule

The frontend does NOT:
- Calculate bucket sizes or relative minutes/hours.
- Iterate raw event arrays to build datasets.
- Maintain a local ticking clock for chart dataset shifting.

The backend is the authoritative source for both **data** and **time labels**.

---

# Store Responsibilities

## Events Store

The store owns:

- Network transport (using `AbortController`)
- `threatVelocityProjection` state
- Contextual fetching via query params.
- Blind invalidation marker (`lastVelocityInvalidation`).

---

# API Endpoint

### GET /api/v1/events/velocity

Returns time-bucketed velocity analytics for the fleet.

---

## Query Parameters

| Parameter | Description |
|---|---|
| `timeframe` | Timeframe filter (`1H`, `24H`, `7D`, `30D`) (default: `24H`) |
| `nodeId` | Filter by node ID (optional) |
| `sensorId` | Filter by sensor ID (optional) |
| `archived` | Include archived events (`true` or `false`) (default: `false`) |

---

## Example Response

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
