package uptime

import (
	"fmt"
	"testing"
	"time"

	"github.com/honeywire/hub/internal/store"
)

func TestCalculateParams_Timeframes(t *testing.T) {
	now := time.Now()
	cases := []struct {
		timeframe string
		blocks    int
		delta     time.Duration
	}{
		{"1H", 30, 2 * time.Minute},
		{"24H", 24, time.Hour},
		{"7D", 7, 24 * time.Hour},
		{"30D", 30, 24 * time.Hour},
	}

	for _, tc := range cases {
		t.Run(tc.timeframe, func(t *testing.T) {
			p := CalculateParams(tc.timeframe, now)
			if p.NumBlocks != tc.blocks {
				t.Errorf("Expected %d blocks, got %d", tc.blocks, p.NumBlocks)
			}
			if p.Delta != tc.delta {
				t.Errorf("Expected %v delta, got %v", tc.delta, p.Delta)
			}
		})
	}
}

func TestCalculateParams_CutoffAlignment(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 37, 12, 0, time.UTC)
	params := CalculateParams("24H", now)

	if params.Cutoff.Minute() != 0 {
		t.Errorf("Expected cutoff minute to be 0, got %d", params.Cutoff.Minute())
	}
	if params.Cutoff.Second() != 0 {
		t.Errorf("Expected cutoff second to be 0, got %d", params.Cutoff.Second())
	}
}

func TestBuildUptimeHistory(t *testing.T) {
	cutoff := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	now := cutoff.Add(24 * time.Hour)
	params := UptimeCalculationParams{
		NumBlocks: 24,
		Delta:     time.Hour,
		Cutoff:    cutoff,
	}

	sensors := []store.SensorUptimeData{
		{NodeID: "node1", SensorID: "sensor1"},
	}

	changes := []store.StatusChangeData{
		{NodeID: "node1", SensorID: "sensor1", Status: "online", Timestamp: cutoff.Add(30 * time.Minute).Format(time.RFC3339)}, // Block 0: 30 min uptime
		{NodeID: "node1", SensorID: "sensor1", Status: "offline", Timestamp: cutoff.Add(90 * time.Minute).Format(time.RFC3339)}, // Block 1: 30 min uptime
	}

	history := BuildUptimeHistory(sensors, changes, params, now, nil)
	key := "node1:sensor1"

	if len(history[key]) != 24 {
		t.Fatalf("Expected 24 blocks, got %d", len(history[key]))
	}

	if history[key][0] != 1800 { // 30 mins
		t.Errorf("Block 0 expected 1800 seconds, got %v", history[key][0])
	}
	if history[key][1] != 1800 { // 30 mins
		t.Errorf("Block 1 expected 1800 seconds, got %v", history[key][1])
	}
	if history[key][2] != 0 { // 0 mins
		t.Errorf("Block 2 expected 0 seconds, got %v", history[key][2])
	}
}

func TestCalculateBlockStatus(t *testing.T) {
	now := time.Now()
	params := UptimeCalculationParams{NumBlocks: 24}
	blockStart := now.Add(-time.Hour)
	blockEnd := now

	t.Run("NotDeployed", func(t *testing.T) {
		firstSeen := now.Add(time.Hour) // Deployed in the future
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, 0, params, 0, false)
		if status.Status != "nodata" {
			t.Errorf("Expected nodata, got %s", status.Status)
		}
	})

	t.Run("HistoricalDown", func(t *testing.T) {
		firstSeen := now.Add(-2 * time.Hour)
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, 0, params, 0, false)
		if status.Status != "down" {
			t.Errorf("Expected down, got %s", status.Status)
		}
	})

	t.Run("HistoricalDegraded", func(t *testing.T) {
		firstSeen := now.Add(-2 * time.Hour)
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, 1800, params, 0, false) // 50% uptime
		if status.Status != "degraded" {
			t.Errorf("Expected degraded, got %s", status.Status)
		}
	})

	t.Run("HistoricalUp", func(t *testing.T) {
		firstSeen := now.Add(-2 * time.Hour)
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, 3600, params, 0, false) // 100% uptime
		if status.Status != "up" {
			t.Errorf("Expected up, got %s", status.Status)
		}
	})
}

func TestGenerateBlocks(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	params := UptimeCalculationParams{
		NumBlocks: 3,
		Delta:     time.Hour,
		Cutoff:    now.Add(-3 * time.Hour), // 09:00
	}

	t.Run("PendingSensor", func(t *testing.T) {
		firstSeen := now.Add(-time.Hour) // Deployed at 11:00
		sensor := store.SensorUptimeData{FirstSeen: firstSeen.Format(time.RFC3339)}
		history := []float64{0, 0, 0}

		blocks := GenerateBlocks(sensor, history, params, "24H", now, "pending")

		// 09:00 - 10:00 -> nodata (before deployment)
		if blocks[0].Status != "nodata" {
			t.Errorf("Block 0 expected nodata, got %s", blocks[0].Status)
		}
		// 10:00 - 11:00 -> nodata (deployed exactly at 11:00)
		if blocks[1].Status != "nodata" {
			t.Errorf("Block 1 expected nodata, got %s", blocks[1].Status)
		}
		// 11:00 - 12:00 -> pending
		if blocks[2].Status != "pending" {
			t.Errorf("Block 2 expected pending, got %s", blocks[2].Status)
		}
	})
}

func TestResolveWorstStatus(t *testing.T) {
	cases := []struct {
		statuses []string
		expected string
	}{
		{[]string{"up", "up"}, "up"},
		{[]string{"up", "degraded"}, "degraded"},
		{[]string{"up", "down"}, "down"},
		{[]string{"pending", "pending"}, "pending"},
		{[]string{"nodata", "nodata"}, ""},
	}

	for _, tc := range cases {
		actual := ResolveWorstStatus(tc.statuses)
		if actual != tc.expected {
			t.Errorf("For %v, expected '%s', got '%s'", tc.statuses, tc.expected, actual)
		}
	}
}

func TestCalculateOverallUptime(t *testing.T) {
	now := time.Now()
	params := UptimeCalculationParams{
		NumBlocks: 24,
		Delta:     time.Hour,
		Cutoff:    now.Add(-24 * time.Hour),
	}

	t.Run("PerfectUptime", func(t *testing.T) {
		sensors := []store.SensorUptimeData{{NodeID: "n1", SensorID: "s1", FirstSeen: params.Cutoff.Format(time.RFC3339)}}
		history := map[string][]float64{
			"n1:s1": make([]float64, 24),
		}
		for i := 0; i < 24; i++ {
			history["n1:s1"][i] = 3600 // 1 hour uptime
		}
		liveMap := map[string]string{"n1:s1": "up"}

		uptime := CalculateOverallUptime(sensors, history, params, now, liveMap)
		if uptime != 100.0 {
			t.Errorf("Expected 100%% uptime, got %f", uptime)
		}
	})
}

func TestOverallUptime_AllGreenButNot100(t *testing.T) {
	// Reproduce: sensor deployed 6 days ago, always online, 7D timeframe.
	// All blocks should be green. Overall should be 100%.
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	params := CalculateParams("7D", now)

	// Sensor was first seen 6.5 days ago (noon, 6 days ago)
	firstSeen := now.Add(-6*24*time.Hour - 12*time.Hour)

	sensors := []store.SensorUptimeData{
		{NodeID: "n1", SensorID: "s1", FirstSeen: firstSeen.Format(time.RFC3339)},
	}

	// The first status change is "online" at firstSeen
	changes := []store.StatusChangeData{
		{NodeID: "n1", SensorID: "s1", Status: "online", Timestamp: firstSeen.Format(time.RFC3339)},
	}

	liveMap := map[string]string{"n1:s1": "up"}

	history := BuildUptimeHistory(sensors, changes, params, now, liveMap)

	// Now check what each block evaluates to
	key := "n1:s1"
	t.Logf("History: %v", history[key])
	t.Logf("Params: NumBlocks=%d, Delta=%v, Cutoff=%v", params.NumBlocks, params.Delta, params.Cutoff)

	for i := 0; i < params.NumBlocks; i++ {
		blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
		blockEnd := blockStart.Add(params.Delta)
		bs := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, history[key][i], params, i, false)
		t.Logf("Block %d: blockStart=%v uptimeSeconds=%.0f status=%s label=%s",
			i, blockStart.Format("2006-01-02 15:04"), history[key][i], bs.Status, bs.Label)
	}

	uptime := CalculateOverallUptime(sensors, history, params, now, liveMap)
	t.Logf("Overall uptime: %.2f%%", uptime)
	if uptime != 100.0 {
		t.Errorf("Expected 100%% uptime (all blocks green), got %.2f%%", uptime)
	}
}

func TestOverallUptime_NoChangesInWindow(t *testing.T) {
	// Sensor deployed 30 days ago, no status changes in the 7D window.
	// The sensor is currently "up". All blocks should be green.
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	params := CalculateParams("7D", now)

	firstSeen := now.Add(-30 * 24 * time.Hour)

	sensors := []store.SensorUptimeData{
		{NodeID: "n1", SensorID: "s1", FirstSeen: firstSeen.Format(time.RFC3339)},
	}

	// NO changes in the window — the sensor was continuously online
	changes := []store.StatusChangeData{}

	liveMap := map[string]string{"n1:s1": "up"}

	history := BuildUptimeHistory(sensors, changes, params, now, liveMap)

	key := "n1:s1"
	t.Logf("History: %v", history[key])

	for i := 0; i < params.NumBlocks; i++ {
		blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
		blockEnd := blockStart.Add(params.Delta)
		bs := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, history[key][i], params, i, false)
		t.Logf("Block %d: blockStart=%v uptimeSeconds=%.0f status=%s",
			i, blockStart.Format("2006-01-02 15:04"), history[key][i], bs.Status)
	}

	uptime := CalculateOverallUptime(sensors, history, params, now, liveMap)
	t.Logf("Overall uptime: %.2f%%", uptime)
	if uptime != 100.0 {
		t.Errorf("Expected 100%% uptime, got %.2f%%", uptime)
	}
}

func TestOverallUptime_MultiSensorDailyBlocks(t *testing.T) {
	// 10 sensors, all online, 7D timeframe. Overall should be 100%.
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	params := CalculateParams("7D", now)

	firstSeen := now.Add(-30 * 24 * time.Hour)

	sensors := make([]store.SensorUptimeData, 10)
	liveMap := make(map[string]string)
	for i := 0; i < 10; i++ {
		sid := fmt.Sprintf("s%d", i)
		sensors[i] = store.SensorUptimeData{NodeID: "n1", SensorID: sid, FirstSeen: firstSeen.Format(time.RFC3339)}
		liveMap["n1:"+sid] = "up"
	}

	changes := []store.StatusChangeData{}
	history := BuildUptimeHistory(sensors, changes, params, now, liveMap)

	uptime := CalculateOverallUptime(sensors, history, params, now, liveMap)
	t.Logf("Overall uptime (10 sensors, all up): %.2f%%", uptime)
	if uptime != 100.0 {
		t.Errorf("Expected 100%% uptime, got %.2f%%", uptime)
	}
}

func TestOverallUptime_SensorDeployedMidBlock30D(t *testing.T) {
	// 1 sensor, deployed 25 days ago (mid-block for 30D timeframe), always online.
	// The deployment block should be "up" and overall should be 100%.
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	params := CalculateParams("30D", now)

	// Deployed 25.5 days ago (noon)
	firstSeen := now.Add(-25*24*time.Hour - 12*time.Hour)

	sensors := []store.SensorUptimeData{
		{NodeID: "n1", SensorID: "s1", FirstSeen: firstSeen.Format(time.RFC3339)},
	}

	changes := []store.StatusChangeData{
		{NodeID: "n1", SensorID: "s1", Status: "online", Timestamp: firstSeen.Format(time.RFC3339)},
	}

	liveMap := map[string]string{"n1:s1": "up"}
	history := BuildUptimeHistory(sensors, changes, params, now, liveMap)

	key := "n1:s1"
	
	notUpCount := 0
	for i := 0; i < params.NumBlocks; i++ {
		blockStart := params.Cutoff.Add(time.Duration(i) * params.Delta)
		blockEnd := blockStart.Add(params.Delta)
		bs := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, history[key][i], params, i, false)
		if bs.Status != "up" && bs.Status != "nodata" {
			t.Logf("Block %d: blockStart=%v uptimeSeconds=%.0f status=%s label=%s",
				i, blockStart.Format("2006-01-02 15:04"), history[key][i], bs.Status, bs.Label)
			notUpCount++
		}
	}

	uptime := CalculateOverallUptime(sensors, history, params, now, liveMap)
	t.Logf("Overall uptime: %.2f%% (notUpBlocks=%d)", uptime, notUpCount)
	if uptime != 100.0 {
		t.Errorf("Expected 100%% uptime, got %.2f%%", uptime)
	}
}
