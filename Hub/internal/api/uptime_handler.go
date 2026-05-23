package api

import (
	"net/http"
	"time"

	"github.com/honeywire/hub/internal/projections/uptime"
)

// GetUptime handles GET /api/v1/uptime and returns the fleet uptime projection
func (h *Handler) GetUptime(w http.ResponseWriter, r *http.Request) {
	// Parse timeframe from query string (default to 24H)
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "24H"
	}

	// Validate timeframe
	validTimeframes := map[string]bool{
		"1H":  true,
		"24H": true,
		"7D":  true,
		"30D": true,
	}
	if !validTimeframes[timeframe] {
		RespondError(w, "Invalid timeframe. Valid values: 1H, 24H, 7D, 30D", http.StatusBadRequest)
		return
	}

	// Create projector
	projector := uptime.NewProjector(h.Store)

	// Build projection
	criteria := uptime.FilterCriteria{
		Timeframe: timeframe,
		Now:       time.Now().UTC(),
	}

	projection, err := projector.BuildUptimeProjection(criteria)
	if err != nil {
		RespondError(w, "Failed to build uptime projection", http.StatusInternalServerError)
		return
	}

	// Return the projection as JSON
	SendJSON(w, http.StatusOK, projection)
}
