package uptime

import "time"

// UptimeResponse is the frontend-facing DTO for uptime data.
// This is the strict API contract - all UI rendering must derive from this structure.
type UptimeResponse struct {
	Timeframe   string        `json:"timeframe"`
	GeneratedAt time.Time     `json:"generated_at"`
	Summary     UptimeSummary `json:"summary"`
	Groups      []UptimeGroup `json:"groups"`
}

// UptimeSummary provides high-level fleet statistics.
type UptimeSummary struct {
	OverallUptime float64 `json:"overall_uptime"`
}

// UptimeGroup represents sensors grouped by a node.
type UptimeGroup struct {
	NodeID      string         `json:"node_id"`
	NodeAlias   string         `json:"node_alias"`
	WorstStatus string         `json:"worst_status"` // "up", "degraded", "down", or "" if all are nodata
	Sensors     []UptimeSensor `json:"sensors"`
}

// UptimeSensor represents a single sensor's uptime history.
type UptimeSensor struct {
	SensorID    string        `json:"sensor_id"`
	DisplayName string        `json:"display_name"`
	Status      string        `json:"status"` // "up", "down", "degraded"
	IsSilenced  bool          `json:"is_silenced"`
	Blocks      []UptimeBlock `json:"blocks"`
}

// UptimeBlock represents a single time bucket in the heatmap.
type UptimeBlock struct {
	Status    string `json:"status"`     // "up", "down", "degraded", "nodata"
	Label     string `json:"label"`      // Human-readable status (e.g., "Offline", "Online", etc.)
	TimeLabel string `json:"time_label"` // Time reference (e.g., "5 hours ago", "Current")
}
