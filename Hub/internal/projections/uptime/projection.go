package uptime

import (
	"time"

	"github.com/honeywire/hub/internal/models"
	"github.com/honeywire/hub/internal/store"
)

// FilterCriteria holds the parameters for building an uptime projection
type FilterCriteria struct {
	Timeframe string
	Now       time.Time
}

// ProjectionStore defines the minimal data access needed for uptime projections
type ProjectionStore interface {
	GetNodes() ([]models.Node, error)
	GetSensorsForUptime(cutoffStr string) ([]store.SensorUptimeData, error)
	GetStatusChangesSince(cutoffStr string) ([]store.StatusChangeData, error)
	IsSensorSilenced(nodeID, sensorID string) (bool, error)
}

// Projector is responsible for building complete uptime projections
type Projector struct {
	Store ProjectionStore
}

// NewProjector creates a new uptime projector
func NewProjector(s ProjectionStore) *Projector {
	return &Projector{Store: s}
}

// BuildUptimeProjection constructs a complete uptime projection from raw backend data
func (p *Projector) BuildUptimeProjection(criteria FilterCriteria) (*UptimeResponse, error) {
	// 1. Calculate parameters based on timeframe
	params := CalculateParams(criteria.Timeframe, criteria.Now)

	// 2. Fetch raw data from store
	sensors, err := p.Store.GetSensorsForUptime(params.Cutoff.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	changes, err := p.Store.GetStatusChangesSince(params.Cutoff.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	nodes, err := p.Store.GetNodes()
	if err != nil {
		return nil, err
	}

	// 3. Build a map for fast node lookup by ID and live status
	nodesMap := make(map[string]models.Node)
	sensorLiveStatusMap := make(map[string]string)
	for _, node := range nodes {
		nodesMap[node.ID] = node
		for _, ns := range node.InstalledSensors {
			sensorLiveStatusMap[node.ID+":"+ns.ID] = ns.Status
		}
	}

	// 4. Build uptime history
	history := BuildUptimeHistory(sensors, changes, params, criteria.Now, sensorLiveStatusMap)

	// Group changes by sensor
	rawChangesBySensor := make(map[string][]store.StatusChangeData)
	for _, c := range changes {
		key := c.NodeID + ":" + c.SensorID
		rawChangesBySensor[key] = append(rawChangesBySensor[key], c)
	}

	// 5. Group sensors by NodeID and build DTOs
	groupsMap := make(map[string]*UptimeGroup)
	groupStatusesMap := make(map[string][]string)

	for _, sensor := range sensors {
		nodeID := sensor.NodeID
		historyKey := nodeID + ":" + sensor.SensorID

		// Get or create group for this node
		if _, exists := groupsMap[nodeID]; !exists {
			nodeAlias := nodeID
			if node, ok := nodesMap[nodeID]; ok {
				nodeAlias = node.Alias
			}
			groupsMap[nodeID] = &UptimeGroup{
				NodeID:    nodeID,
				NodeAlias: nodeAlias,
				Sensors:   make([]UptimeSensor, 0),
			}
		}

		sensorHistory := history[historyKey]
		if sensorHistory == nil {
			sensorHistory = make([]float64, params.NumBlocks)
		}

		liveStatus := sensorLiveStatusMap[historyKey]
		sensorChanges := rawChangesBySensor[historyKey]
		blocks := GenerateBlocks(sensor, sensorChanges, sensorHistory, params, criteria.Timeframe, criteria.Now, liveStatus)
		for _, block := range blocks {
			groupStatusesMap[nodeID] = append(groupStatusesMap[nodeID], block.Status)
		}

		// Determine sensor status from the most recent block or live status
		sensorStatus := sensorLiveStatusMap[historyKey]
		if sensorStatus == "" {
			if len(blocks) > 0 {
				lastBlock := blocks[len(blocks)-1]
				sensorStatus = lastBlock.Status
				if sensorStatus == "nodata" {
					sensorStatus = "up" // Treat nodata as up for status display
				}
			} else {
				sensorStatus = "up"
			}
		}

		// Check if sensor is silenced
		isSilenced, _ := p.Store.IsSensorSilenced(nodeID, sensor.SensorID)

		sensorDTO := UptimeSensor{
			SensorID:    sensor.SensorID,
			DisplayName: sensor.SensorID,
			Status:      sensorStatus,
			IsSilenced:  isSilenced,
			Blocks:      blocks,
		}

		groupsMap[nodeID].Sensors = append(groupsMap[nodeID].Sensors, sensorDTO)
	}

	// 6. Calculate worst status per group
	for _, group := range groupsMap {
		group.WorstStatus = ResolveWorstStatus(groupStatusesMap[group.NodeID])
	}

	// 7. Convert map to sorted slice
	groups := make([]UptimeGroup, 0, len(groupsMap))
	for _, group := range groupsMap {
		groups = append(groups, *group)
	}

	// Sort groups: unassigned last, others alphabetically
	sortGroups(groups)

	// 8. Calculate overall uptime
	overallUptime := CalculateOverallUptime(sensors, rawChangesBySensor, history, params, criteria.Now, sensorLiveStatusMap)

	// 9. Build response
	response := &UptimeResponse{
		Timeframe:   criteria.Timeframe,
		GeneratedAt: criteria.Now,
		Summary: UptimeSummary{
			OverallUptime: overallUptime,
		},
		Groups: groups,
	}

	return response, nil
}

// sortGroups sorts groups with unassigned last and others alphabetically
func sortGroups(groups []UptimeGroup) {
	// Simple bubble sort for small datasets
	for i := 0; i < len(groups); i++ {
		for j := i + 1; j < len(groups); j++ {
			if shouldSwap(groups[i], groups[j]) {
				groups[i], groups[j] = groups[j], groups[i]
			}
		}
	}
}

// shouldSwap determines if two groups should be swapped during sorting
func shouldSwap(a, b UptimeGroup) bool {
	if a.NodeID == "unassigned" {
		return false // unassigned stays at end
	}
	if b.NodeID == "unassigned" {
		return true // move other groups before unassigned
	}
	return a.NodeID > b.NodeID
}
