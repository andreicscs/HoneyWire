package api

import (
	"net/http"
	"time"

	"github.com/honeywire/hub/internal/projections/severity"
	"github.com/honeywire/hub/internal/projections/summary"
	"github.com/honeywire/hub/internal/projections/uptime"
	"github.com/honeywire/hub/internal/projections/velocity"
	"github.com/honeywire/hub/internal/store"
)

type AnalyticsHandler struct {
	Store *store.SQLiteStore
}

func NewAnalyticsHandler(store *store.SQLiteStore) *AnalyticsHandler {
	return &AnalyticsHandler{Store: store}
}

func (h *AnalyticsHandler) GetVelocityAnalytics(w http.ResponseWriter, r *http.Request) {
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "24H"
	}

	nodeID := r.URL.Query().Get("nodeId")
	sensorID := r.URL.Query().Get("sensorId")
	viewingArchiveStr := r.URL.Query().Get("archived")
	viewingArchive := 0
	if viewingArchiveStr == "true" || viewingArchiveStr == "1" {
		viewingArchive = 1
	}

	projector := velocity.NewProjector(h.Store)
	projection, err := projector.BuildThreatVelocityProjection(r.Context(), timeframe, nodeID, sensorID, viewingArchive)
	if err != nil {
		RespondError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, projection)
}

func (h *AnalyticsHandler) GetSummaryAnalytics(w http.ResponseWriter, r *http.Request) {
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "24H"
	}
	nodeID := r.URL.Query().Get("nodeId")
	sensorID := r.URL.Query().Get("sensorId")

	viewingArchiveStr := r.URL.Query().Get("archived")
	viewingArchive := 0
	if viewingArchiveStr == "true" || viewingArchiveStr == "1" {
		viewingArchive = 1
	}

	projector := summary.NewProjector(h.Store)
	projection, err := projector.BuildSummaryProjection(r.Context(), timeframe, nodeID, sensorID, viewingArchive)
	if err != nil {
		if err.Error() == "not_found" {
			RespondError(w, "Node not found", http.StatusNotFound)
			return
		}
		RespondError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, projection)
}

func (h *AnalyticsHandler) GetSeverityAnalytics(w http.ResponseWriter, r *http.Request) {
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "alltime"
	}

	nodeID := r.URL.Query().Get("node")
	sensorID := r.URL.Query().Get("sensor")
	viewingArchiveStr := r.URL.Query().Get("viewingArchive")
	viewingArchive := 0
	if viewingArchiveStr == "true" || viewingArchiveStr == "1" {
		viewingArchive = 1
	}

	projector := severity.NewProjector(h.Store)
	projection, err := projector.BuildSeverityProjection(r.Context(), timeframe, nodeID, sensorID, viewingArchive)
	if err != nil {
		RespondError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, projection)
}

// GetUptime handles GET /api/v1/uptime and returns the fleet uptime projection
func (h *AnalyticsHandler) GetUptime(w http.ResponseWriter, r *http.Request) {
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
