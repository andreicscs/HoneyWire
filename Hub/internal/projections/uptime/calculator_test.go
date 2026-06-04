package uptime

import (
	"testing"
	"time"

	"github.com/honeywire/hub/internal/store"
)

// ============================================================================
// 1. CalculateParams()
// ============================================================================

func TestCalculateParams_Timeframes(t *testing.T) {
	now := time.Now()
	cases := []struct {
		timeframe string
		blocks    int
		delta     time.Duration
		expected  float64
	}{
		{"1H", 30, 2 * time.Minute, 2},
		{"24H", 24, time.Hour, 60},
		{"7D", 7, 24 * time.Hour, 1440},
		{"30D", 30, 24 * time.Hour, 1440},
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
			if p.ExpectedPings != tc.expected {
				t.Errorf("Expected %f pings, got %f", tc.expected, p.ExpectedPings)
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

// ============================================================================
// 2. BuildHeartbeatHistory()
// ============================================================================

func TestBuildHeartbeatHistory_BucketsCorrectly(t *testing.T) {
	cutoff := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	params := UptimeCalculationParams{
		NumBlocks: 24,
		Delta:     time.Hour,
		Cutoff:    cutoff,
	}

	sensors := []store.SensorUptimeData{
		{NodeID: "node1", SensorID: "sensor1"},
	}

	heartbeats := []store.HeartbeatData{
		{NodeID: "node1", SensorID: "sensor1", TimeBucket: cutoff.Add(30 * time.Minute).Format(time.RFC3339)},             // Block 0
		{NodeID: "node1", SensorID: "sensor1", TimeBucket: cutoff.Add(90 * time.Minute).Format(time.RFC3339)},             // Block 1
		{NodeID: "node1", SensorID: "sensor1", TimeBucket: cutoff.Add(2*time.Hour + 10*time.Minute).Format(time.RFC3339)}, // Block 2
		{NodeID: "node1", SensorID: "sensor1", TimeBucket: cutoff.Add(2*time.Hour + 20*time.Minute).Format(time.RFC3339)}, // Block 2 (second ping)
	}

	history := BuildHeartbeatHistory(sensors, heartbeats, params)
	key := "node1:sensor1"

	if len(history[key]) != 24 {
		t.Fatalf("Expected 24 blocks, got %d", len(history[key]))
	}
	if history[key][0] != 1 || history[key][1] != 1 || history[key][2] != 2 || history[key][3] != 0 {
		t.Errorf("Buckets populated incorrectly: %v", history[key][:4])
	}
}

func TestBuildHeartbeatHistory_IgnoresOldData(t *testing.T) {
	cutoff := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	params := UptimeCalculationParams{
		NumBlocks: 24,
		Delta:     time.Hour,
		Cutoff:    cutoff,
	}

	sensors := []store.SensorUptimeData{{NodeID: "node1", SensorID: "sensor1"}}
	heartbeats := []store.HeartbeatData{
		{NodeID: "node1", SensorID: "sensor1", TimeBucket: cutoff.Add(-2 * time.Hour).Format(time.RFC3339)}, // Before cutoff
	}

	history := BuildHeartbeatHistory(sensors, heartbeats, params)
	if history["node1:sensor1"][0] != 0 {
		t.Errorf("Old data was not ignored")
	}
}

// ============================================================================
// 3. CalculateBlockStatus()
// ============================================================================

func TestCalculateBlockStatus(t *testing.T) {
	now := time.Now()
	params := UptimeCalculationParams{ExpectedPings: 60, NumBlocks: 24}
	blockStart := now.Add(-time.Hour)
	blockEnd := now

	t.Run("NotDeployed", func(t *testing.T) {
		firstSeen := now.Add(time.Hour) // Deployed in the future
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, time.Time{}, 0, params, 0, false)
		if status.Status != "nodata" {
			t.Errorf("Expected nodata, got %s", status.Status)
		}
	})

	t.Run("PendingBeforeFirstPing", func(t *testing.T) {
		firstSeen := now.Add(-2 * time.Hour) // Deployed in the past
		firstPing := now.Add(time.Hour)      // First ping hasn't happened yet
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, firstPing, 0, params, 0, false)
		if status.Status != "pending" {
			t.Errorf("Expected pending, got %s", status.Status)
		}
	})

	t.Run("HistoricalDown", func(t *testing.T) {
		firstSeen := now.Add(-2 * time.Hour)
		firstPing := now.Add(-90 * time.Minute) // Pinged long ago
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, firstPing, 0, params, 0, false)
		if status.Status != "down" {
			t.Errorf("Expected down, got %s", status.Status)
		}
	})

	t.Run("HistoricalDegraded", func(t *testing.T) {
		firstSeen := now.Add(-2 * time.Hour)
		firstPing := now.Add(-90 * time.Minute)
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, firstPing, 40, params, 0, false)
		if status.Status != "degraded" {
			t.Errorf("Expected degraded, got %s", status.Status)
		}
	})

	t.Run("HistoricalUp", func(t *testing.T) {
		firstSeen := now.Add(-2 * time.Hour)
		firstPing := now.Add(-90 * time.Minute)
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, firstPing, 60, params, 0, false)
		if status.Status != "up" {
			t.Errorf("Expected up, got %s", status.Status)
		}
	})

	t.Run("CurrentBlockOffline", func(t *testing.T) {
		firstSeen := now.Add(-2 * time.Hour)
		firstPing := now.Add(-90 * time.Minute)
		status := CalculateBlockStatus(blockStart, blockEnd, now, firstSeen, firstPing, 0, params, params.NumBlocks-1, true)
		if status.Status != "down" {
			t.Errorf("Expected down, got %s", status.Status)
		}
	})

	t.Run("CurrentBlockPartialProgress", func(t *testing.T) {
		// Current block just started 5 minutes ago
		bStart := now.Add(-5 * time.Minute)
		bEnd := bStart.Add(time.Hour)
		firstSeen := now.Add(-2 * time.Hour)
		firstPing := now.Add(-90 * time.Minute)

		// Only 5 pings so far, but it's only been 5 minutes!
		status := CalculateBlockStatus(bStart, bEnd, now, firstSeen, firstPing, 5, params, params.NumBlocks-1, false)
		if status.Status != "up" {
			t.Errorf("Expected up for early partial progress, got %s", status.Status)
		}
	})

	t.Run("FirstHeartbeatNotDegraded", func(t *testing.T) {
		bStart := now.Add(-5 * time.Minute)
		bEnd := bStart.Add(time.Hour)
		firstSeen := now.Add(-2 * time.Hour)
		firstPing := now // Just pinged right now

		// 1 ping so far
		status := CalculateBlockStatus(bStart, bEnd, now, firstSeen, firstPing, 1, params, params.NumBlocks-1, false)
		if status.Status != "up" {
			t.Errorf("Expected up for newly online sensor, got %s", status.Status)
		}
	})

	t.Run("FirstHeartbeatLateInBlock", func(t *testing.T) {
		// Current block started 55 minutes ago (only 5 mins left)
		bStart := now.Add(-55 * time.Minute)
		bEnd := bStart.Add(time.Hour)
		firstSeen := now.Add(-2 * time.Hour) // Deployed 2 hours ago
		firstPing := now                     // Just pinged right now

		status := CalculateBlockStatus(bStart, bEnd, now, firstSeen, firstPing, 1, params, params.NumBlocks-1, false)
		if status.Status != "up" {
			t.Errorf("Expected up for newly online sensor, got %s", status.Status)
		}
	})
}

// ============================================================================
// 4. GenerateBlocks()
// ============================================================================

func TestGenerateBlocks(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	params := UptimeCalculationParams{
		NumBlocks:     3,
		Delta:         time.Hour,
		ExpectedPings: 60,
		Cutoff:        now.Add(-3 * time.Hour), // 09:00
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

	t.Run("LiveSensor", func(t *testing.T) {
		firstSeen := now.Add(-4 * time.Hour) // Deployed well in the past
		sensor := store.SensorUptimeData{FirstSeen: firstSeen.Format(time.RFC3339)}
		history := []float64{60, 0, 60} // Up, Down, Up

		blocks := GenerateBlocks(sensor, history, params, "24H", now, "up")

		if blocks[0].Status != "up" || blocks[1].Status != "down" || blocks[2].Status != "up" {
			t.Errorf("Unexpected live sensor statuses: %s, %s, %s", blocks[0].Status, blocks[1].Status, blocks[2].Status)
		}
	})

	t.Run("FreshDeploymentAllPending", func(t *testing.T) {
		firstSeen := now.Add(-10 * time.Minute)
		sensor := store.SensorUptimeData{FirstSeen: firstSeen.Format(time.RFC3339)}
		history := []float64{0, 0, 0}

		blocks := GenerateBlocks(sensor, history, params, "24H", now, "pending")
		if blocks[2].Status != "pending" { // The block encompassing the last 10 mins
			t.Errorf("Expected block 2 to be pending, got %s", blocks[2].Status)
		}
	})

	t.Run("FirstHeartbeatArrives", func(t *testing.T) {
		// Timeline: Deployed at 10:00, first ping at 10:15
		firstSeen := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
		sensor := store.SensorUptimeData{FirstSeen: firstSeen.Format(time.RFC3339)}

		// 09:00-10:00 (0 pings), 10:00-11:00 (1 ping at 10:15), 11:00-12:00 (60 pings)
		history := []float64{0, 1, 60}

		blocks := GenerateBlocks(sensor, history, params, "24H", now, "up")

		// Before 10:00
		if blocks[0].Status != "nodata" {
			t.Errorf("Expected block 0 to be nodata, got %s", blocks[0].Status)
		}
		// 10:00-11:00: Contains the first ping. It's partially "up" depending on progress, but importantly not "down"
		if blocks[1].Status == "down" {
			t.Errorf("Block 1 incorrectly rewrote to down, should be up/degraded")
		}
		// After 10:15
		if blocks[2].Status != "up" {
			t.Errorf("Expected block 2 to be up, got %s", blocks[2].Status)
		}
	})

	t.Run("LiveSensorNoHistoricalPings", func(t *testing.T) {
		firstSeen := now.Add(-time.Hour)
		sensor := store.SensorUptimeData{FirstSeen: firstSeen.Format(time.RFC3339)}
		history := []float64{0, 0, 0} // DB hasn't flushed yet

		// But live status is "up" (just checked in)
		blocks := GenerateBlocks(sensor, history, params, "24H", now, "up")

		// It should assume the first ping is exactly 'now' and not rewrite the past hour to "down"
		if blocks[2].Status == "down" {
			t.Errorf("Block 2 incorrectly marked down for a brand new live sensor")
		}
	})
}

// ============================================================================
// 5. ResolveWorstStatus()
// ============================================================================

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

// ============================================================================
// 6. CalculateOverallUptime()
// ============================================================================

func TestCalculateOverallUptime(t *testing.T) {
	now := time.Now()
	params := UptimeCalculationParams{
		NumBlocks:     24,
		Delta:         time.Hour,
		ExpectedPings: 60,
		Cutoff:        now.Add(-24 * time.Hour),
	}

	t.Run("PerfectUptime", func(t *testing.T) {
		sensors := []store.SensorUptimeData{{NodeID: "n1", SensorID: "s1", FirstSeen: params.Cutoff.Format(time.RFC3339)}}
		history := map[string][]float64{
			"n1:s1": make([]float64, 24),
		}
		for i := 0; i < 24; i++ {
			history["n1:s1"][i] = 60
		}
		liveMap := map[string]string{"n1:s1": "up"}

		uptime := CalculateOverallUptime(sensors, history, params, now, liveMap)
		if uptime != 100.0 {
			t.Errorf("Expected 100%% uptime, got %f", uptime)
		}
	})

	t.Run("HalfUptime", func(t *testing.T) {
		sensors := []store.SensorUptimeData{{NodeID: "n1", SensorID: "s1", FirstSeen: params.Cutoff.Format(time.RFC3339)}}
		history := map[string][]float64{
			"n1:s1": make([]float64, 24),
		}
		// 12 up, 12 down
		for i := 0; i < 12; i++ {
			history["n1:s1"][i] = 60
		}
		liveMap := map[string]string{"n1:s1": "down"} // Current is down

		uptime := CalculateOverallUptime(sensors, history, params, now, liveMap)
		if uptime != 50.0 {
			t.Errorf("Expected 50%% uptime, got %f", uptime)
		}
	})

	t.Run("PendingSensorExcluded", func(t *testing.T) {
		// Sensor A = Perfect, Sensor B = Pending
		sensors := []store.SensorUptimeData{
			{NodeID: "n1", SensorID: "s1", FirstSeen: params.Cutoff.Format(time.RFC3339)},
			{NodeID: "n1", SensorID: "s2", FirstSeen: now.Add(-time.Hour).Format(time.RFC3339)},
		}

		history := map[string][]float64{
			"n1:s1": make([]float64, 24),
			"n1:s2": make([]float64, 24),
		}
		for i := 0; i < 24; i++ {
			history["n1:s1"][i] = 60
		}

		liveMap := map[string]string{
			"n1:s1": "up",
			"n1:s2": "pending",
		}

		uptime := CalculateOverallUptime(sensors, history, params, now, liveMap)
		if uptime != 100.0 {
			t.Errorf("Expected 100%% uptime when pending sensor is excluded, got %f", uptime)
		}
	})
}
