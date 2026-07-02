package severity

// SeverityProjection represents a flat DTO for the frontend.
type SeverityProjection struct {
	Timeframe string `json:"timeframe"`

	Total    int `json:"total"`
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}
