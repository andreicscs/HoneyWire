package velocity

type ThreatVelocityProjection struct {
	Timeframe        string           `json:"timeframe"`
	BucketSizeMs     int64            `json:"bucketSizeMs"`
	GeneratedAt      int64            `json:"generatedAt"`

	BucketTimestamps []int64          `json:"bucketTimestamps"` // Raw epoch timestamps
	Labels           []string         `json:"labels"`
	ExactTimes       []string         `json:"exactTimes"`

	Series           map[string][]int `json:"series"`
	RecentEventCount int              `json:"recentEventCount"`
}
