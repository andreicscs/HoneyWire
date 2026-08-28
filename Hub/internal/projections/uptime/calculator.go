package uptime

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/honeywire/hub/internal/store"
)

// Canonical uptime block status vocabulary
const (
	StatusUp       = "up"
	StatusDown     = "down"
	StatusDegraded = "degraded"
	StatusPending  = "pending"
	StatusNoData   = "nodata"
)

// IsLiveOnline normalizes live status string checks into a boolean
func IsLiveOnline(status string) bool {
	switch strings.ToLower(status) {
	case "up", "online", "alive":
		return true
	default:
		return false
	}
}

// IsLiveOffline checks if a live status represents an offline state
func IsLiveOffline(status string) bool {
	switch strings.ToLower(status) {
	case "down", "offline":
		return true
	default:
		return false
	}
}

// IsLivePending checks if a live status represents a pending state
func IsLivePending(status string) bool {
	return strings.ToLower(status) == "pending"
}

// UptimeCalculationParams holds parameters needed for uptime calculations
type UptimeCalculationParams struct {
	NumBlocks int
	Delta     time.Duration
	Cutoff    time.Time
}

// SensorBaselineMeta holds baseline timestamps and operational monitoring boundaries
type SensorBaselineMeta struct {
	FirstSeen        time.Time
	OperationalStart time.Time
	HasPriorChange   bool
	EarliestChange   time.Time
	EarliestOnline   time.Time
}

// ResolveSensorBaseline extracts baseline timestamps and determines the operational start boundary
func ResolveSensorBaseline(firstSeenStr string, changes []store.StatusChangeData, cutoff time.Time) SensorBaselineMeta {
	firstSeen, _ := time.Parse(time.RFC3339, firstSeenStr)
	meta := SensorBaselineMeta{
		FirstSeen:        firstSeen,
		OperationalStart: firstSeen,
	}

	if len(changes) == 0 {
		return meta
	}

	for _, c := range changes {
		t, err := time.Parse(time.RFC3339, c.Timestamp)
		if err != nil {
			continue
		}
		if meta.EarliestChange.IsZero() || t.Before(meta.EarliestChange) {
			meta.EarliestChange = t
		}
		if c.Status == "online" && (meta.EarliestOnline.IsZero() || t.Before(meta.EarliestOnline)) {
			meta.EarliestOnline = t
		}
		if !t.After(cutoff) {
			meta.HasPriorChange = true
		}
	}

	// If no status transition existed prior to cutoff, operational start begins at the first online check-in.
	if !meta.HasPriorChange && !meta.EarliestOnline.IsZero() && meta.EarliestOnline.After(meta.OperationalStart) {
		meta.OperationalStart = meta.EarliestOnline
	}

	return meta
}

// IsNoDataBlock evaluates whether a block represents an unrecorded telemetry period or pre-deployment window
func IsNoDataBlock(blockEnd time.Time, meta SensorBaselineMeta, hasChanges bool, liveStatus string) bool {
	// 1. Block ended before the sensor was created / deployed
	if !blockEnd.After(meta.FirstSeen) {
		return true
	}

	// 2. Zero-guessing rule: If no baseline exists prior to cutoff, blocks before the first recorded transition are No Data
	if !meta.HasPriorChange && hasChanges && !blockEnd.After(meta.EarliestChange) {
		return true
	}

	// 3. Sensor has zero transitions recorded and is not currently online
	if !hasChanges && !IsLiveOnline(liveStatus) {
		return true
	}

	return false
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
			if len(sensorChanges) == 0 && IsLiveOnline(sensorLiveStatusMap[key]) {
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
			if IsLiveOnline(sensorLiveStatusMap[key]) {
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
	Status   string  // "up", "down", "degraded", "nodata"
	Label    string  // Human-readable explanation
	Duration float64 // Effective seconds this block covers (bounded by operationalStart/now)
	Ratio    float64 // Effective uptime ratio used to derive Status (0 for down/nodata)
}

// CalculateBlockStatus determines the uptime status for a single time block
func CalculateBlockStatus(
	blockStart, blockEnd, now, operationalStart time.Time,
	uptimeSeconds float64,
	params UptimeCalculationParams,
	blockIndex int,
	isLiveOffline bool,
) BlockStatus {
	if !blockEnd.After(operationalStart) {
		return BlockStatus{Status: StatusNoData, Label: "No Data (Not Deployed Yet)"}
	}

	effectiveStart := blockStart
	if operationalStart.After(effectiveStart) {
		effectiveStart = operationalStart
	}

	effectiveEnd := blockEnd
	if now.Before(effectiveEnd) {
		effectiveEnd = now
	}

	blockDuration := effectiveEnd.Sub(effectiveStart).Seconds()
	if blockDuration <= 0 {
		return BlockStatus{Status: StatusNoData, Label: "No Data"}
	}

	if blockIndex == params.NumBlocks-1 && isLiveOffline {
		return BlockStatus{Status: StatusDown, Label: "Offline", Duration: blockDuration, Ratio: 0}
	}

	ratio := uptimeSeconds / blockDuration
	if ratio > 1.0 {
		ratio = 1.0
	}

	if ratio >= 0.99 {
		return BlockStatus{Status: StatusUp, Label: "Online", Duration: blockDuration, Ratio: ratio}
	} else if ratio > 0 {
		return BlockStatus{Status: StatusDegraded, Label: fmt.Sprintf("Degraded (%.1f%% uptime)", ratio*100), Duration: blockDuration, Ratio: ratio}
	}
	return BlockStatus{Status: StatusDown, Label: "Offline", Duration: blockDuration, Ratio: 0}
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
	blocks := make([]UptimeBlock, params.NumBlocks)
	meta := ResolveSensorBaseline(sensorData.FirstSeen, sensorChanges, params.Cutoff)
	hasChanges := len(sensorChanges) > 0
	isPending := IsLivePending(liveStatus)
	isOffline := IsLiveOffline(liveStatus)

	for i := 0; i < params.NumBlocks; i++ {
		blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
		blockEnd := blockStart.Add(params.Delta)
		stepsAgo := params.NumBlocks - 1 - i
		timeLabel := formatTimeLabel(stepsAgo, params.Delta, timeframe)

		if isPending {
			if !blockEnd.After(meta.FirstSeen) {
				blocks[i] = UptimeBlock{
					Status:    StatusNoData,
					Label:     "No Data (Not Deployed Yet)",
					TimeLabel: timeLabel,
				}
			} else {
				blocks[i] = UptimeBlock{
					Status:    StatusPending,
					Label:     "Awaiting Initial Check-in",
					TimeLabel: timeLabel,
				}
			}
			continue
		}

		if IsNoDataBlock(blockEnd, meta, hasChanges, liveStatus) {
			label := "No Data"
			if !blockEnd.After(meta.FirstSeen) {
				label = "No Data (Not Deployed Yet)"
			}
			blocks[i] = UptimeBlock{
				Status:    StatusNoData,
				Label:     label,
				TimeLabel: timeLabel,
			}
			continue
		}

		blockStatus := CalculateBlockStatus(blockStart, blockEnd, now, meta.OperationalStart, history[i], params, i, isOffline)
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
		if status == StatusDown {
			return StatusDown
		}
	}
	for _, status := range statuses {
		if status == StatusDegraded {
			return StatusDegraded
		}
	}
	for _, status := range statuses {
		if status == StatusUp {
			return StatusUp
		}
	}
	for _, status := range statuses {
		if status == StatusPending {
			return StatusPending
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

	var totalSeconds, upSeconds float64

	for _, sensor := range sensors {
		historyKey := sensor.NodeID + ":" + sensor.SensorID
		sensorHistory := history[historyKey]
		if sensorHistory == nil {
			continue
		}

		liveStatus := sensorLiveStatusMap[historyKey]
		if IsLivePending(liveStatus) {
			continue // Pending sensors don't count against overall uptime
		}

		sensorChanges := changesBySensor[historyKey]
		meta := ResolveSensorBaseline(sensor.FirstSeen, sensorChanges, params.Cutoff)
		hasChanges := len(sensorChanges) > 0
		isOffline := IsLiveOffline(liveStatus)

		for i := 0; i < params.NumBlocks; i++ {
			blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
			blockEnd := blockStart.Add(params.Delta)

			if IsNoDataBlock(blockEnd, meta, hasChanges, liveStatus) {
				continue
			}

			blockStatus := CalculateBlockStatus(blockStart, blockEnd, now, meta.OperationalStart, sensorHistory[i], params, i, isOffline)
			if blockStatus.Status == StatusNoData || blockStatus.Status == StatusPending || blockStatus.Duration <= 0 {
				continue
			}

			totalSeconds += blockStatus.Duration
			upSeconds += blockStatus.Duration * blockStatus.Ratio
		}
	}

	if totalSeconds == 0 {
		return 100.0
	}

	return (upSeconds / totalSeconds) * 100.0
}
