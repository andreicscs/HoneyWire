package uptime

import (
	"fmt"
	"sort"
	"time"

	"github.com/honeywire/hub/internal/store"
)

// UptimeCalculationParams holds parameters needed for uptime calculations
type UptimeCalculationParams struct {
	NumBlocks int
	Delta     time.Duration
	Cutoff    time.Time
}

// CalculateParams determines the calculation parameters based on timeframe
func CalculateParams(timeframe string, now time.Time) UptimeCalculationParams {
	var numBlocks int
	var delta time.Duration

	switch timeframe {
	case "1H":
		numBlocks, delta = 30, 2*time.Minute
	case "7D":
		numBlocks, delta = 7, 24*time.Hour
	case "30D":
		numBlocks, delta = 30, 24*time.Hour
	case "24H":
		fallthrough
	default:
		numBlocks, delta = 24, time.Hour
	}

	// Align the grid to the delta boundary to prevent phase-shift visual jitter,
	// and ensure the final block always encompasses 'now'.
	cutoff := now.Truncate(delta).Add(-delta * time.Duration(numBlocks-1))

	return UptimeCalculationParams{
		NumBlocks: numBlocks,
		Delta:     delta,
		Cutoff:    cutoff,
	}
}

// BuildUptimeHistory aggregates status changes to compute seconds spent 'online' per block.
func BuildUptimeHistory(
	sensors []store.SensorUptimeData,
	changes []store.StatusChangeData,
	params UptimeCalculationParams,
	now time.Time,
	sensorLiveStatusMap map[string]string,
) map[string][]float64 {
	history := make(map[string][]float64)
	for _, s := range sensors {
		historyKey := s.NodeID + ":" + s.SensorID
		history[historyKey] = make([]float64, params.NumBlocks)
	}

	// Group changes by sensor
	changesBySensor := make(map[string][]store.StatusChangeData)
	for _, c := range changes {
		key := c.NodeID + ":" + c.SensorID
		changesBySensor[key] = append(changesBySensor[key], c)
	}

	for key, sensorChanges := range changesBySensor {
		if _, ok := history[key]; !ok {
			continue // Skip unknown sensors
		}

		// Sort changes by timestamp
		sort.Slice(sensorChanges, func(i, j int) bool {
			return sensorChanges[i].Timestamp < sensorChanges[j].Timestamp
		})

		// Determine initial status at cutoff.
		isOnline := false
		changeIdx := 0
		hasPriorChange := false

		for changeIdx < len(sensorChanges) {
			t, err := time.Parse(time.RFC3339, sensorChanges[changeIdx].Timestamp)
			if err == nil && !t.After(params.Cutoff) {
				isOnline = (sensorChanges[changeIdx].Status == "online")
				hasPriorChange = true
				changeIdx++
			} else {
				break
			}
		}

		if !hasPriorChange {
			if len(sensorChanges) == 0 && sensorLiveStatusMap[key] == "up" {
				isOnline = true
			}
		}

		for i := 0; i < params.NumBlocks; i++ {
			blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
			blockEnd := blockStart.Add(params.Delta)
			if blockEnd.After(now) {
				blockEnd = now
			}

			if blockStart.After(now) || blockStart.Equal(now) {
				break
			}

			uptimeSeconds := 0.0
			blockCurrentTime := blockStart

			for blockCurrentTime.Before(blockEnd) {
				var nextChangeTime time.Time
				var nextStatus string
				hasChange := false

				if changeIdx < len(sensorChanges) {
					t, err := time.Parse(time.RFC3339, sensorChanges[changeIdx].Timestamp)
					if err == nil {
						nextChangeTime = t
						nextStatus = sensorChanges[changeIdx].Status
						hasChange = true
					}
				}

				if hasChange && !nextChangeTime.After(blockCurrentTime) {
					// Change happened before or at the current time we are tracking
					if nextStatus == "online" {
						isOnline = true
					} else {
						isOnline = false
					}
					changeIdx++
					continue
				}

				endTime := blockEnd
				if hasChange && nextChangeTime.Before(blockEnd) {
					endTime = nextChangeTime
				}

				duration := endTime.Sub(blockCurrentTime).Seconds()
				if isOnline {
					uptimeSeconds += duration
				}

				blockCurrentTime = endTime
				if hasChange && blockCurrentTime.Equal(nextChangeTime) {
					if nextStatus == "online" {
						isOnline = true
					} else {
						isOnline = false
					}
					changeIdx++
				}
			}

			history[key][i] = uptimeSeconds
		}
	}

	// For sensors with NO changes in the period, fill their history based on live status.
	for _, s := range sensors {
		key := s.NodeID + ":" + s.SensorID
		if len(changesBySensor[key]) == 0 {
			if sensorLiveStatusMap[key] == "up" {
				for i := 0; i < params.NumBlocks; i++ {
					blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
					blockEnd := blockStart.Add(params.Delta)
					if blockEnd.After(now) {
						blockEnd = now
					}
					if blockStart.Before(now) {
						history[key][i] = blockEnd.Sub(blockStart).Seconds()
					}
				}
			}
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
	blockStart, blockEnd, now, firstSeen time.Time,
	uptimeSeconds float64,
	params UptimeCalculationParams,
	blockIndex int,
	isLiveOffline bool,
) BlockStatus {
	if !blockEnd.After(firstSeen) {
		return BlockStatus{Status: "nodata", Label: "No Data (Not Deployed Yet)"}
	}

	effectiveStart := blockStart
	if firstSeen.After(effectiveStart) {
		effectiveStart = firstSeen
	}

	effectiveEnd := blockEnd
	if now.Before(effectiveEnd) {
		effectiveEnd = now
	}

	blockDuration := effectiveEnd.Sub(effectiveStart).Seconds()
	if blockDuration <= 0 {
		return BlockStatus{Status: "nodata", Label: "No Data"}
	}

	if blockIndex == params.NumBlocks-1 && isLiveOffline {
		return BlockStatus{Status: "down", Label: "Offline"}
	}

	ratio := uptimeSeconds / blockDuration

	if ratio >= 0.99 {
		return BlockStatus{Status: "up", Label: "Online"}
	} else if ratio > 0 {
		return BlockStatus{Status: "degraded", Label: fmt.Sprintf("Degraded (%.1f%% uptime)", ratio*100)}
	}
	return BlockStatus{Status: "down", Label: "Offline"}
}

// GenerateBlocks creates the heatmap blocks for a sensor
func GenerateBlocks(
	sensorData store.SensorUptimeData,
	sensorChanges []store.StatusChangeData,
	history []float64,
	params UptimeCalculationParams,
	timeframe string,
	now time.Time,
	liveStatus string,
) []UptimeBlock {
	firstSeenParsed, _ := time.Parse(time.RFC3339, sensorData.FirstSeen)
	blocks := make([]UptimeBlock, params.NumBlocks)

	hasPriorChange := false
	var earliestChange time.Time
	if len(sensorChanges) > 0 {
		for _, c := range sensorChanges {
			t, err := time.Parse(time.RFC3339, c.Timestamp)
			if err == nil {
				if earliestChange.IsZero() || t.Before(earliestChange) {
					earliestChange = t
				}
				if !t.After(params.Cutoff) {
					hasPriorChange = true
				}
			}
		}
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

		if !blockEnd.After(firstSeenParsed) {
			blocks[i] = UptimeBlock{
				Status:    "nodata",
				Label:     "No Data (Not Deployed Yet)",
				TimeLabel: timeLabel,
			}
			continue
		}

		// Zero-guessing rule: If no baseline exists prior to cutoff, blocks before the first recorded transition are "No Data"
		if !hasPriorChange && len(sensorChanges) > 0 && !blockEnd.After(earliestChange) {
			blocks[i] = UptimeBlock{
				Status:    "nodata",
				Label:     "No Data",
				TimeLabel: timeLabel,
			}
			continue
		}

		// If sensor has zero transitions recorded and is not live online, mark as no data
		if len(sensorChanges) == 0 && liveStatus != "up" && liveStatus != "online" {
			blocks[i] = UptimeBlock{
				Status:    "nodata",
				Label:     "No Data",
				TimeLabel: timeLabel,
			}
			continue
		}

		isLiveOffline := liveStatus == "down"
		blockStatus := CalculateBlockStatus(blockStart, blockEnd, now, firstSeenParsed, history[i], params, i, isLiveOffline)
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
	return ""
}

// CalculateOverallUptime computes the fleet-wide uptime percentage
func CalculateOverallUptime(
	sensors []store.SensorUptimeData,
	changesBySensor map[string][]store.StatusChangeData,
	history map[string][]float64,
	params UptimeCalculationParams,
	now time.Time,
	sensorLiveStatusMap map[string]string,
) float64 {
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
		sensorChanges := changesBySensor[historyKey]

		hasPriorChange := false
		var earliestChange time.Time
		if len(sensorChanges) > 0 {
			for _, c := range sensorChanges {
				t, err := time.Parse(time.RFC3339, c.Timestamp)
				if err == nil {
					if earliestChange.IsZero() || t.Before(earliestChange) {
						earliestChange = t
					}
					if !t.After(params.Cutoff) {
						hasPriorChange = true
					}
				}
			}
		}

		isLiveOffline := liveStatus == "down"

		for i := 0; i < params.NumBlocks; i++ {
			blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
			blockEnd := blockStart.Add(params.Delta)

			if !blockEnd.After(firstSeenParsed) {
				continue
			}

			if !hasPriorChange && len(sensorChanges) > 0 && !blockEnd.After(earliestChange) {
				continue // No data
			}

			if len(sensorChanges) == 0 && liveStatus != "up" && liveStatus != "online" {
				continue // No data
			}

			blockStatus := CalculateBlockStatus(blockStart, blockEnd, now, firstSeenParsed, sensorHistory[i], params, i, isLiveOffline)
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
