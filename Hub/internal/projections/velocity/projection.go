package velocity

import (
	"context"
	"fmt"
	"time"

	"github.com/honeywire/hub/internal/models"
)

// ProjectionStore defines the minimal data access needed for velocity projections
type ProjectionStore interface {
	GetEvents(isArchived int, nodeID string, sensorID string) ([]models.Event, error)
}

// Projector is responsible for building velocity projections
type Projector struct {
	Store ProjectionStore
}

// NewProjector creates a new velocity projector
func NewProjector(s ProjectionStore) *Projector {
	return &Projector{Store: s}
}

// BuildThreatVelocityProjection constructs a threat velocity projection
func (p *Projector) BuildThreatVelocityProjection(ctx context.Context, timeframe string, nodeID string, sensorID string, viewingArchive int) (ThreatVelocityProjection, error) {
	events, err := p.Store.GetEvents(viewingArchive, nodeID, sensorID)
	if err != nil {
		return ThreatVelocityProjection{}, fmt.Errorf("failed to fetch raw events for velocity projection: %w", err)
	}

	return BuildProjection(events, timeframe, time.Now().UTC()), nil
}
