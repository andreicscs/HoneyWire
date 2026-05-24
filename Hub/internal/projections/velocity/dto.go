package velocity

type ThreatVelocityProjection struct {
	Timeframe        string           `json:"timeframe"`
	BucketSizeMs     int64            `json:"bucket_size_ms"`
	GeneratedAt      int64            `json:"generated_at"`

	BucketTimestamps []int64          `json:"bucket_timestamps"` // Raw epoch timestamps
	Labels           []string         `json:"labels"`
	ExactTimes       []string         `json:"exact_times"`

	Series           map[string][]int `json:"series"`
	RecentEventCount int              `json:"recent_event_count"`
}
