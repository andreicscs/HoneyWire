package api

import (
	"fmt"
	"time"

	"github.com/honeywire/hub/internal/store"
)

// UptimeParams holds the parameters needed to fetch data from the database
type UptimeParams struct {
	NumBlocks     int
	Delta         time.Duration
	ExpectedPings float64
	Cutoff        time.Time
	CutoffStr     string
}

func CalculateUptimeParams(timeframe string, now time.Time) UptimeParams {
	var numBlocks int
	var delta time.Duration
	var expectedPings float64

	switch timeframe {
	case "1H":
		numBlocks, delta, expectedPings = 30, 2*time.Minute, 2
	case "7D":
		numBlocks, delta, expectedPings = 7, 24*time.Hour, 1440
	case "30D":
		numBlocks, delta, expectedPings = 30, 24*time.Hour, 1440
	case "24H":
		fallthrough
	default:
		numBlocks, delta, expectedPings = 24, time.Hour, 60
	}

	cutoff := now.Add(-delta * time.Duration(numBlocks)).Truncate(time.Minute)
	return UptimeParams{
		NumBlocks:     numBlocks,
		Delta:         delta,
		ExpectedPings: expectedPings,
		Cutoff:        cutoff,
		CutoffStr:     cutoff.Format(time.RFC3339),
	}
}

func GenerateUptimeResult(timeframe string, now time.Time, params UptimeParams, sensors []store.SensorUptimeData, hbs []store.HeartbeatData) []map[string]interface{} {
	history := make(map[string][]float64)
	for _, s := range sensors {
		historyKey := s.NodeID + ":" + s.SensorID
		history[historyKey] = make([]float64, params.NumBlocks)
	}

	for _, hb := range hbs {
		parsedBucket, err := time.Parse(time.RFC3339, hb.TimeBucket)
		if err != nil {
			continue
		}

		if parsedBucket.Before(params.Cutoff) {
			continue
		}

		idx := int(parsedBucket.Sub(params.Cutoff) / params.Delta)
		if idx >= params.NumBlocks {
			idx = params.NumBlocks - 1
		}

		historyKey := hb.NodeID + ":" + hb.SensorID
		if idx >= 0 && history[historyKey] != nil {
			history[historyKey][idx]++
		}
	}

	var result []map[string]interface{}
	for _, s := range sensors {
		firstSeenParsed, _ := time.Parse(time.RFC3339, s.FirstSeen)
		var blocks []map[string]string

		historyKey := s.NodeID + ":" + s.SensorID

		for i := 0; i < params.NumBlocks; i++ {
			blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
			blockEnd := blockStart.Add(params.Delta)

			stepsAgo := params.NumBlocks - 1 - i
			timeLabel := "Current"
			if stepsAgo > 0 {
				switch timeframe {
				case "1H":
					timeLabel = fmt.Sprintf("%d mins ago", stepsAgo*int(params.Delta.Minutes()))
				case "24H":
					timeLabel = fmt.Sprintf("%d hours ago", stepsAgo)
				case "7D", "30D":
					timeLabel = fmt.Sprintf("%d days ago", stepsAgo)
				default:
					timeLabel = fmt.Sprintf("%d ago", stepsAgo)
				}
			}

			status, label := "", ""

			if blockEnd.Before(firstSeenParsed) {
				status, label = "nodata", "No Data (Not Deployed Yet)"
			} else {
				pings := history[historyKey][i]
				targetPings := params.ExpectedPings

				if firstSeenParsed.After(blockStart) && firstSeenParsed.Before(blockEnd) {
					activeDuration := blockEnd.Sub(firstSeenParsed)
					targetPings = activeDuration.Minutes()
					if targetPings > params.ExpectedPings {
						targetPings = params.ExpectedPings
					}
					if targetPings < 1 && activeDuration > 0 {
						targetPings = 1
					}
				} else if i == params.NumBlocks-1 {
					activeDuration := now.Sub(blockStart)
					targetPings = activeDuration.Minutes()
					if targetPings > params.ExpectedPings {
						targetPings = params.ExpectedPings
					}
					if targetPings < 1 && activeDuration > 0 {
						targetPings = 1
					}
				}

				if pings == 0 && targetPings >= 1 {
					status, label = "down", "Offline"
				} else if targetPings > 0 && pings < (targetPings*0.85) {
					status, label = "degraded", fmt.Sprintf("Degraded (%.0f/%.0f pings)", pings, targetPings)
				} else {
					status, label = "up", "Online"
				}
			}

			blocks = append(blocks, map[string]string{
				"status":    status,
				"timeLabel": timeLabel,
				"label":     label,
			})
		}

		isLive := now.Sub(s.LastSeen) < 60*time.Second
		if isLive {
			blocks[len(blocks)-1]["status"] = "up"
			blocks[len(blocks)-1]["label"] = "Online (Live)"
		} else {
			blocks[len(blocks)-1]["status"] = "down"
			blocks[len(blocks)-1]["label"] = "Offline (Live)"
		}

		result = append(result, map[string]interface{}{
			"id":       s.SensorID,
			"node_id":  s.NodeID,
			"name":     s.SensorID,
			"isOnline": isLive,
			"blocks":   blocks,
		})
	}

	if result == nil {
		result = []map[string]interface{}{}
	}
	
	return result
}
