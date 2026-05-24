package velocity

import (
	"fmt"
	"strings"
	"time"

	"github.com/honeywire/hub/internal/models"
)

func BuildProjection(events []models.Event, timeframe string, now time.Time) ThreatVelocityProjection {
	var buckets int
	var bucketSizeMs int64

	switch timeframe {
	case "24H":
		buckets = 24
		bucketSizeMs = 3600000
	case "7D":
		buckets = 14
		bucketSizeMs = 43200000
	case "30D":
		buckets = 30
		bucketSizeMs = 86400000
	default:
		// 1H default
		buckets = 30
		bucketSizeMs = 120000
		timeframe = "1H"
	}

	labels := make([]string, buckets)
	exactTimes := make([]string, buckets)
	bucketTimestamps := make([]int64, buckets)

	for i := 0; i < buckets; i++ {
		stepsAgo := buckets - 1 - i
		d := now.Add(-time.Duration(int64(stepsAgo)*bucketSizeMs) * time.Millisecond)

		bucketTimestamps[i] = d.UnixMilli()
		exactTimes[i] = d.Format("Jan 2, 03:04 PM") // Equivalent to: month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'

		if stepsAgo == 0 {
			labels[i] = "Now"
		} else {
			if timeframe == "1H" {
				labels[i] = fmt.Sprintf("-%dm", stepsAgo*2)
			} else if timeframe == "24H" {
				labels[i] = fmt.Sprintf("-%dh", stepsAgo)
			} else if timeframe == "7D" {
				labels[i] = fmt.Sprintf("-%dh", stepsAgo*12)
			} else if timeframe == "30D" {
				labels[i] = fmt.Sprintf("-%dd", stepsAgo)
			}
		}
	}

	series := map[string][]int{
		"critical": make([]int, buckets),
		"high":     make([]int, buckets),
		"medium":   make([]int, buckets),
		"low":      make([]int, buckets),
		"info":     make([]int, buckets),
	}

	count := 0
	for _, e := range events {
		if e.Timestamp == "" {
			continue
		}
		eTime, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			continue
		}

		diffMs := now.Sub(eTime).Milliseconds()
		diffMins := int(diffMs / bucketSizeMs)

		if diffMins >= 0 && diffMins < buckets {
			sev := "info"
			if e.Severity != "" {
				sev = strings.ToLower(e.Severity)
			}
			// severity should be normalized, assume lowercase
			switch sev {
			case "critical", "high", "medium", "low", "info":
				// valid
			default:
				sev = "info"
			}
			series[sev][buckets-1-diffMins]++
			count++
		}
	}

	return ThreatVelocityProjection{
		Timeframe:        timeframe,
		BucketSizeMs:     bucketSizeMs,
		GeneratedAt:      now.UnixMilli(),
		BucketTimestamps: bucketTimestamps,
		Labels:           labels,
		ExactTimes:       exactTimes,
		Series:           series,
		RecentEventCount: count,
	}
}
