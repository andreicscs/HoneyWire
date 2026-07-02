package severity

import (
	"context"
	"fmt"
	"time"

	"github.com/honeywire/hub/internal/models"
)

// ProjectionStore defines the minimal data access needed for severity projections
type ProjectionStore interface {
	GetEvents(isArchived int, nodeID string, sensorID string) ([]models.Event, error)
}

// Projector is responsible for building severity projections
type Projector struct {
	Store ProjectionStore
}

// NewProjector creates a new severity projector
func NewProjector(s ProjectionStore) *Projector {
	return &Projector{Store: s}
}

// BuildSeverityProjection constructs a severity projection
func (p *Projector) BuildSeverityProjection(ctx context.Context, timeframe string, nodeID string, sensorID string, viewingArchive int) (SeverityProjection, error) {
	events, err := p.Store.GetEvents(viewingArchive, nodeID, sensorID)
	if err != nil {
		return SeverityProjection{}, fmt.Errorf("failed to fetch raw events for severity projection: %w", err)
	}

	filteredEvents := filterByTimeframe(events, timeframe)

	counts := CalculateDistribution(filteredEvents)

	return SeverityProjection{
		Timeframe: timeframe,

		Total:    counts.Total,
		Critical: counts.Critical,
		High:     counts.High,
		Medium:   counts.Medium,
		Low:      counts.Low,
		Info:     counts.Info,
	}, nil
}

func filterByTimeframe(events []models.Event, timeframe string) []models.Event {
	if timeframe == "alltime" || timeframe == "" {
		return events
	}

	now := time.Now().UTC()
	var cutoff time.Time

	switch timeframe {
	case "24h":
		cutoff = now.Add(-24 * time.Hour)
	case "7d":
		cutoff = now.Add(-7 * 24 * time.Hour)
	case "30d":
		cutoff = now.Add(-30 * 24 * time.Hour)
	default:
		return events
	}

	var filtered []models.Event
	for _, e := range events {
		t, err := time.Parse(time.RFC3339, e.Timestamp)
		if err == nil && t.After(cutoff) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
