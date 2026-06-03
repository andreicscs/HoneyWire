package uptime

import (
	"fmt"
	"time"

	"github.com/honeywire/hub/internal/store"
)

// UptimeCalculationParams holds parameters needed for uptime calculations
type UptimeCalculationParams struct {
	NumBlocks     int
	Delta         time.Duration
	ExpectedPings float64
	Cutoff        time.Time
}

// CalculateParams determines the calculation parameters based on timeframe
func CalculateParams(timeframe string, now time.Time) UptimeCalculationParams {
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

	// Align the grid to the delta boundary to prevent phase-shift visual jitter,
	// and ensure the final block always encompasses 'now'.
	cutoff := now.Truncate(delta).Add(-delta * time.Duration(numBlocks-1))

	return UptimeCalculationParams{
		NumBlocks:     numBlocks,
		Delta:         delta,
		ExpectedPings: expectedPings,
		Cutoff:        cutoff,
	}
}

// HistoryBucket represents aggregated heartbeat data for a sensor in a time period
type HistoryBucket struct {
	SensorKey string
	Pings     []float64
}

// BuildHeartbeatHistory aggregates heartbeat data into time-based buckets
func BuildHeartbeatHistory(
	sensors []store.SensorUptimeData,
	heartbeats []store.HeartbeatData,
	params UptimeCalculationParams,
) map[string][]float64 {
	history := make(map[string][]float64)
	for _, s := range sensors {
		historyKey := s.NodeID + ":" + s.SensorID
		history[historyKey] = make([]float64, params.NumBlocks)
	}

	for _, hb := range heartbeats {
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

	return history
}

// BlockStatus represents the computed status of a time block
type BlockStatus struct {
	Status string // "up", "down", "degraded", "nodata"
	Label  string // Human-readable explanation
}

// CalculateBlockStatus determines the uptime status for a single time block
func CalculateBlockStatus(
	blockStart, blockEnd, now, firstSeen, firstPingTime time.Time,
	pings float64,
	params UptimeCalculationParams,
	blockIndex int,
	isLiveOffline bool,
) BlockStatus {
	status, label := "", ""

	if !blockEnd.After(firstSeen) {
		// Sensor not yet deployed at this time
		status, label = "nodata", "No Data (Not Deployed Yet)"
	} else if !firstPingTime.IsZero() && !blockEnd.After(firstPingTime) {
		// Deployed, but before the first successful heartbeat. This is the "pending" phase.
		status, label = "pending", "Awaiting Initial Check-in"
	} else if firstPingTime.IsZero() && !isLiveOffline {
		// Deployed, but no heartbeats ever recorded, and it's not currently considered offline.
		status, label = "pending", "Awaiting Initial Check-in"
	} else {
		targetPings := params.ExpectedPings

		// Adjust expected pings if deployment occurred mid-block
		if firstSeen.After(blockStart) && firstSeen.Before(blockEnd) {
			activeDuration := blockEnd.Sub(firstSeen)
			targetPings = activeDuration.Minutes()
			if targetPings > params.ExpectedPings {
				targetPings = params.ExpectedPings
			}
			if targetPings < 1 && activeDuration > 0 {
				targetPings = 1
			}
		}

		// Strict 85% SLA threshold
		acceptablePings := targetPings * 0.85

		if blockIndex == params.NumBlocks-1 {
			if isLiveOffline {
				status, label = "down", "Offline"
			} else {
				remainingDuration := blockEnd.Sub(now)
				if remainingDuration < 0 {
					remainingDuration = 0
				}
				maxPossiblePings := pings + remainingDuration.Minutes()

				if maxPossiblePings < acceptablePings {
					status, label = "degraded", fmt.Sprintf("Degraded (%.0f/%.0f pings)", pings, targetPings)
				} else {
					status, label = "up", "Online"
				}
			}
		} else {
			if pings == 0 && targetPings >= 1 {
				status, label = "down", "Offline"
			} else if targetPings > 0 && pings < acceptablePings {
				status, label = "degraded", fmt.Sprintf("Degraded (%.0f/%.0f pings)", pings, targetPings)
			} else {
				status, label = "up", "Online"
			}
		}
	}

	return BlockStatus{Status: status, Label: label}
}

// GenerateBlocks creates the heatmap blocks for a sensor
func GenerateBlocks(
	sensorData store.SensorUptimeData,
	history []float64,
	params UptimeCalculationParams,
	timeframe string,
	now time.Time,
	liveStatus string,
) []UptimeBlock {
	firstSeenParsed, _ := time.Parse(time.RFC3339, sensorData.FirstSeen)
	blocks := make([]UptimeBlock, params.NumBlocks)

	var firstPingTime time.Time
	// Find the start time of the first block that has pings.
	for i, pings := range history {
		if pings > 0 {
			firstPingTime = params.Cutoff.Add(time.Duration(i) * params.Delta)
			break
		}
	}
	// If no pings in history, but sensor is live, the first ping is now.
	// This handles the case where the very first ping just occurred.
	if firstPingTime.IsZero() && liveStatus == "up" {
		firstPingTime = now
	}

	for i := 0; i < params.NumBlocks; i++ {
		blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
		blockEnd := blockStart.Add(params.Delta)

		stepsAgo := params.NumBlocks - 1 - i
		timeLabel := formatTimeLabel(stepsAgo, params.Delta, timeframe)

		if liveStatus == "pending" {
			if !blockEnd.After(firstSeenParsed) {
				blocks[i] = UptimeBlock{
					Status:    "nodata",
					Label:     "No Data (Not Deployed Yet)",
					TimeLabel: timeLabel,
				}
			} else {
				blocks[i] = UptimeBlock{
					Status:    "pending",
					Label:     "Awaiting Initial Check-in",
					TimeLabel: timeLabel,
				}
			}
			continue
		}

		isLiveOffline := liveStatus == "down"
		blockStatus := CalculateBlockStatus(blockStart, blockEnd, now, firstSeenParsed, firstPingTime, history[i], params, i, isLiveOffline)
		blocks[i] = UptimeBlock{
			Status:    blockStatus.Status,
			Label:     blockStatus.Label,
			TimeLabel: timeLabel,
		}
	}

	return blocks
}

// formatTimeLabel creates a human-readable time reference
func formatTimeLabel(stepsAgo int, delta time.Duration, timeframe string) string {
	if stepsAgo == 0 {
		return "Current"
	}

	switch timeframe {
	case "1H":
		return fmt.Sprintf("%d mins ago", stepsAgo*int(delta.Minutes()))
	case "24H":
		return fmt.Sprintf("%d hours ago", stepsAgo)
	case "7D", "30D":
		return fmt.Sprintf("%d days ago", stepsAgo)
	default:
		return fmt.Sprintf("%d ago", stepsAgo)
	}
}

// ResolveWorstStatus determines the worst status among a list of statuses
func ResolveWorstStatus(statuses []string) string {
	for _, status := range statuses {
		if status == "down" {
			return "down"
		}
	}
	for _, status := range statuses {
		if status == "degraded" {
			return "degraded"
		}
	}
	// All are "up" or "nodata"
	for _, status := range statuses {
		if status == "up" {
			return "up"
		}
	}
	for _, status := range statuses {
		if status == "pending" {
			return "pending"
		}
	}
	// All are "nodata"
	return ""
}

// CalculateOverallUptime computes the fleet-wide uptime percentage
func CalculateOverallUptime(sensors []store.SensorUptimeData, history map[string][]float64, params UptimeCalculationParams, now time.Time, sensorLiveStatusMap map[string]string) float64 {
	if len(sensors) == 0 {
		return 100.0
	}

	totalBlocks := 0
	upBlocks := 0

	for _, sensor := range sensors {
		historyKey := sensor.NodeID + ":" + sensor.SensorID
		sensorHistory := history[historyKey]
		if sensorHistory == nil {
			continue
		}

		liveStatus := sensorLiveStatusMap[historyKey]
		if liveStatus == "pending" {
			continue // Pending sensors don't count against overall uptime
		}

		firstSeenParsed, _ := time.Parse(time.RFC3339, sensor.FirstSeen)

		var firstPingTime time.Time
		for i, pings := range sensorHistory {
			if pings > 0 {
				firstPingTime = params.Cutoff.Add(time.Duration(i) * params.Delta)
				break
			}
		}
		if firstPingTime.IsZero() && liveStatus == "up" {
			firstPingTime = now
		}

		isLiveOffline := liveStatus == "down"

		for i := 0; i < params.NumBlocks; i++ {
			blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
			blockEnd := blockStart.Add(params.Delta)

			blockStatus := CalculateBlockStatus(blockStart, blockEnd, now, firstSeenParsed, firstPingTime, sensorHistory[i], params, i, isLiveOffline)
			if blockStatus.Status == "nodata" || blockStatus.Status == "pending" {
				continue
			}

			totalBlocks++
			if blockStatus.Status == "up" {
				upBlocks++
			}
		}
	}

	if totalBlocks == 0 {
		return 100.0
	}

	percentage := (float64(upBlocks) / float64(totalBlocks)) * 100.0
	return percentage
}
