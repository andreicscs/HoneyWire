package summary

import (
	"context"
	"time"

	"github.com/honeywire/hub/internal/models"
)

type ProjectionStore interface {
	GetEvents(isArchived int, nodeID, sensorID string) ([]models.Event, error)
}

type Projector struct {
	Store ProjectionStore
}

func NewProjector(s ProjectionStore) *Projector {
	return &Projector{Store: s}
}

func (p *Projector) BuildSummaryProjection(ctx context.Context, timeframe, nodeID, sensorID string, viewingArchive int) (*SummaryDTO, error) {
	now := time.Now().UTC()
	var since time.Time

	switch timeframe {
	case "1H":
		since = now.Add(-1 * time.Hour)
	case "7D":
		since = now.Add(-7 * 24 * time.Hour)
	case "30D":
		since = now.Add(-30 * 24 * time.Hour)
	case "alltime":
		since = time.Time{}
	case "24H":
		fallthrough
	default:
		since = now.Add(-24 * time.Hour)
		timeframe = "24H"
	}

	events, err := p.Store.GetEvents(viewingArchive, nodeID, sensorID)
	if err != nil {
		return nil, err
	}

	return CalculateSummary(events, timeframe, since), nil
}
