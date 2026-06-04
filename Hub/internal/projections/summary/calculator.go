package summary

import (
	"time"

	"github.com/honeywire/hub/internal/models"
)

// CalculateSummary iterates over raw events and aggregates the summary projection
func CalculateSummary(events []models.Event, timeframe string, since time.Time) *SummaryDTO {
	dto := &SummaryDTO{
		Timeframe: timeframe,
		BySensor:  make(map[string]int),
	}

	for _, e := range events {
		if !since.IsZero() {
			t, err := time.Parse(time.RFC3339, e.Timestamp)
			if err == nil && t.Before(since) {
				continue
			}
		}
		dto.TotalEvents++
		dto.BySensor[e.SensorID]++
	}

	return dto
}
