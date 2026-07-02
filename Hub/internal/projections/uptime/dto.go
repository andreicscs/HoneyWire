package uptime

import "time"

// UptimeResponse is the frontend-facing DTO for uptime data.
type UptimeResponse struct {
    Timeframe   string        `json:"timeframe"`
    GeneratedAt time.Time     `json:"generatedAt"`
    Summary     UptimeSummary `json:"summary"`
    Groups      []UptimeGroup `json:"groups"`
}

type UptimeSummary struct {
    OverallUptime float64 `json:"overallUptime"`
}

type UptimeGroup struct {
    NodeID      string         `json:"nodeId"`     
    NodeAlias   string         `json:"nodeAlias"`  
    WorstStatus string         `json:"worstStatus"`
    Sensors     []UptimeSensor `json:"sensors"`
}

type UptimeSensor struct {
    SensorID    string        `json:"sensorId"`   
    DisplayName string        `json:"displayName"`
    Status      string        `json:"status"`
    IsSilenced  bool          `json:"isSilenced"` 
    Blocks      []UptimeBlock `json:"blocks"`
}

type UptimeBlock struct {
    Status    string `json:"status"`
    Label     string `json:"label"`
    TimeLabel string `json:"timeLabel"`
}