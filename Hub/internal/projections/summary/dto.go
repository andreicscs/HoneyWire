package summary

// SummaryDTO is the flat read-model for event counts
type SummaryDTO struct {
	Timeframe   string         `json:"timeframe"`
	TotalEvents int            `json:"totalEvents"`
	BySensor    map[string]int `json:"bySensor"`
}
