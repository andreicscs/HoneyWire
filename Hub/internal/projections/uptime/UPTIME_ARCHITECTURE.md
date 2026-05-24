# Uptime Projection Architecture Guide

This document describes the refactored uptime analytics architecture, which moves from frontend-computed UI to backend-generated projections.

## Architecture Overview

### Backend Projection Model
```
Database → Backend Calculations → Typed DTOs → Frontend Strict Rendering
```

**Benefits:**
- Strong type contracts between backend and frontend
- Single source of truth for all uptime logic
- Pure business logic separated from HTTP handlers
- Testable calculations with zero external dependencies
- Frontend only does shallow hydration for real-time updates

## Directory Structure

```
internal/
├── projections/           # New domain: analytics/UI projection layer
│   └── uptime/
│       ├── dto.go         # API contracts (zero logic)
│       ├── calculator.go  # Pure business logic
│       └── projection.go  # Orchestration & mapping
├── api/
│   ├── analytics.go  # Thin HTTP handler
│   └── ... (other handlers)
└── ... (other packages)
```

## Component Responsibilities

### 1. DTOs (`dto.go`)

**Purpose:** Define the exact JSON structure the frontend expects.

**Rules:**
- Zero business logic or methods
- Explicit JSON tags on all fields
- Immutable once defined (breaking changes require versioning)

**Types:**
- `UptimeResponse`: Root response object
- `UptimeSummary`: Fleet-wide statistics
- `UptimeGroup`: Sensors grouped by node
- `UptimeSensor`: Individual sensor with blocks
- `UptimeBlock`: Single time-bucket heatmap cell

### 2. Calculator (`calculator.go`)

**Purpose:** Pure business logic isolated from HTTP/database concerns.

**Rules:**
- No database access (data passed as parameters)
- No HTTP request context
- Functions should be deterministic and testable
- Descriptive names that explain intent

**Key Functions:**
- `CalculateParams()`: Determine block count, delta, expected pings per timeframe
- `BuildHeartbeatHistory()`: Aggregate heartbeats into time-bucketed map
- `CalculateBlockStatus()`: Determine up/down/degraded for a single block
- `GenerateBlocks()`: Build heatmap for a sensor
- `ResolveWorstStatus()`: Determine worst status from list (down > degraded > up)
- `CalculateOverallUptime()`: Fleet-wide uptime percentage

**Testing Strategy:**
```go
// All calculator functions can be tested directly without mocking
func TestCalculateBlockStatus(t *testing.T) {
    status := CalculateBlockStatus(start, end, now, firstSeen, pings, params, idx)
    assert.Equal(t, "up", status.Status)
}
```

### 3. Projection (`projection.go`)

**Purpose:** Orchestrate data flow from storage to DTOs.

**Responsibilities:**
1. Accept `FilterCriteria` (timeframe, now)
2. Fetch raw data from store
3. Invoke calculators on raw data
4. Map results into DTOs
5. Return complete `UptimeResponse`

**Interface Design:**
```go
type ProjectionStore interface {
    GetNodes() ([]models.Node, error)
    GetSensorsForUptime(cutoffStr string) ([]store.SensorUptimeData, error)
    GetHeartbeatsSince(cutoffStr string) ([]store.HeartbeatData, error)
    IsSensorSilenced(nodeID, sensorID string) (bool, error)
}
```

The minimal interface allows easy testing with mocks.

### 4. HTTP Handler (`uptime_handler.go`)

**Purpose:** Parse HTTP request → Call projector → Serialize response.

**Rules:**
- Zero business logic
- Validate input early
- Delegate to projector
- Handle HTTP-specific concerns (status codes, error messages)

```go
func (h *Handler) GetUptime(w http.ResponseWriter, r *http.Request) {
    // 1. Parse & validate
    timeframe := r.URL.Query().Get("timeframe")
    if !isValidTimeframe(timeframe) {
        RespondError(w, "Invalid timeframe", http.StatusBadRequest)
        return
    }

    // 2. Delegate
    projector := uptime.NewProjector(h.Store)
    projection, err := projector.BuildUptimeProjection(uptime.FilterCriteria{...})
    if err != nil {
        RespondError(w, "Failed to build projection", http.StatusInternalServerError)
        return
    }

    // 3. Serialize
    SendJSON(w, http.StatusOK, projection)
}
```

## Frontend Integration

### Data Flow

**Initial Load:**
1. Component mounts → Fleet store's `fetchUptime()` calls `GET /api/v1/uptime?timeframe=24H`
2. API returns `UptimeResponse` → Stored in `uptimeData`
3. Component computes `hydratedGroups` → Shallow hydration with live status from fleet store
4. Template renders from `hydratedGroups.groups` (already grouped, no computation)

**Real-time Updates (WebSocket):**
1. `heartbeat` event updates `fleet.nodes[nodeId].installedSensors[sensorId].status`
2. Vue reactivity triggers `hydratedGroups` recomputation
3. Only the "Current" block's status is updated (shallow hydration)
4. Historical blocks are unchanged (important!)

### Frontend Responsibilities

**Allowed:**
- Rendering the nested structure
- Shallow hydration for live status
- Filtering/sorting on UI view
- Animation/transition effects

**Forbidden:**
- Grouping sensors by node (backend does this)
- Calculating worst_status
- Recalculating overall_uptime
- Inferring historical downtime blocks
- Joining data from multiple sources

### Hydration Function

```typescript
// Only update the "Current" block's live status
const hydrateGroupsWithLiveStatus = (groups) => {
  return groups.map(group => ({
    ...group,
    sensors: group.sensors.map(sensor => {
      const blocks = [...sensor.blocks]
      if (blocks.length > 0) {
        const lastBlock = blocks[blocks.length - 1]
        lastBlock.status = isLiveOnline ? 'up' : 'down'
      }
      return { ...sensor, blocks }
    })
  }))
}
```

## API Endpoints

### GET /api/v1/uptime

**Query Parameters:**
- `timeframe` (string): `1H`, `24H`, `7D`, `30D` (default: `24H`)

**Response:**
```json
{
  "timeframe": "24H",
  "generated_at": "2026-05-23T14:30:00Z",
  "summary": {
    "overall_uptime": 99.52
  },
  "groups": [
    {
      "node_id": "prod-server-1",
      "node_alias": "Production Primary",
      "worst_status": "up",
      "sensors": [
        {
          "sensor_id": "hw-tcp-tarpit",
          "display_name": "TCP Tarpit",
          "status": "up",
          "is_silenced": false,
          "blocks": [
            {
              "status": "up",
              "label": "Online",
              "time_label": "Current"
            },
            {
              "status": "up",
              "label": "Online",
              "time_label": "1 hours ago"
            }
          ]
        }
      ]
    }
  ]
}
```

## Extending the Architecture

### Adding a New Calculation

1. Add function to `calculator.go`:
   ```go
   func CalculateSLABreach(blocks []UptimeBlock) bool {
       // Pure logic only
   }
   ```

2. Add field to DTO in `dto.go`:
   ```go
   type UptimeSensor struct {
       // ... existing fields
       SLABreached bool `json:"sla_breached"`
   }
   ```

3. Invoke calculator in `projection.go`:
   ```go
   sensorDTO.SLABreached = CalculateSLABreach(blocks)
   ```

4. Update frontend component to render new field (if needed)

### Adding a New Timeframe

1. Add case to `CalculateParams()` in `calculator.go`
2. Update `formatTimeLabel()` to handle new granularity
3. Update frontend's timeframe dropdown (if needed)
4. No handler changes needed!

### Testing the Projection Layer

```go
// Create a test store mock
type MockStore struct {
    nodes     []models.Node
    sensors   []store.SensorUptimeData
    heartbeats []store.HeartbeatData
}

func (m *MockStore) GetNodes() ([]models.Node, error) {
    return m.nodes, nil
}

// Run test
func TestBuildUptimeProjection(t *testing.T) {
    store := &MockStore{...}
    projector := uptime.NewProjector(store)
    result, err := projector.BuildUptimeProjection(uptime.FilterCriteria{...})
    assert.NoError(t, err)
    assert.Equal(t, "up", result.Groups[0].WorstStatus)
}
```

## Common Pitfalls

### ❌ Adding business logic to the handler
```go
// WRONG
func (h *Handler) GetUptime(w http.ResponseWriter, r *http.Request) {
    worst := "" // Don't calculate here!
}

// RIGHT
func (h *Handler) GetUptime(w http.ResponseWriter, r *http.Request) {
    projection, _ := projector.BuildUptimeProjection(criteria)
    SendJSON(w, http.StatusOK, projection)
}
```

### ❌ Frontend re-computing projections
```javascript
// WRONG
const groups = computed(() => {
    return flatten(uptimeData).map(s => ({
        worst: calculateWorst(s.blocks) // Don't do this!
    }))
})

// RIGHT
const hydratedGroups = computed(() => {
    return hydrateWithLiveStatus(uptimeData?.groups)
})
```

### ❌ Changing DTOs without versioning
```go
// WRONG: Existing frontend breaks
type UptimeSensor struct {
    // Removed: SensorID string
    Identifier string // Use new name
}

// RIGHT: Add new field, deprecate old
type UptimeSensor struct {
    SensorID string `json:"sensor_id"` // Keep for compatibility
    Identifier string `json:"identifier"` // New field
}
```

## Maintenance Guidelines

### When to Update Each Layer

| Change | Where | Why |
|--------|-------|-----|
| Fix uptime calculation bug | `calculator.go` | Isolated, testable |
| Add time range filter | `projection.go` | Doesn't touch DTOs |
| New status type | `calculator.go` + `dto.go` | Pure logic + contract |
| Style changes | Frontend component | No backend impact |
| Fetch different data | `projection.go` store interface | Fetch layer only |