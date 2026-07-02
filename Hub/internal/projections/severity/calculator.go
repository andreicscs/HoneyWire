package severity

import "github.com/honeywire/hub/internal/models"

type SeverityCounts struct {
	Total    int
	Critical int
	High     int
	Medium   int
	Low      int
	Info     int
}

func CalculateDistribution(events []models.Event) SeverityCounts {
	counts := SeverityCounts{}
	for _, e := range events {
		counts.Total++
		switch e.Severity {
		case "critical":
			counts.Critical++
		case "high":
			counts.High++
		case "medium":
			counts.Medium++
		case "low":
			counts.Low++
		case "info":
			counts.Info++
		}
	}
	return counts
}
