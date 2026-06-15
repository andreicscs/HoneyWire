package discovery

import (
	"github.com/honeywire/wizard/core/scanner"
	"github.com/honeywire/wizard/core/schema"
	"github.com/honeywire/wizard/internal/system"
)

type MatchedService struct {
	ProcessName string
	Port        int
	PID         int
}

// Recommendation represents a suggested sensor deployment.
type Recommendation struct {
	SensorID           string
	SensorName         string
	Reason             string
	MatchedServices    []MatchedService // Correlated process-port bindings
	DeploymentTemplate *schema.Deployment
	Manifest           *schema.SensorManifest
}

// Engine performs matching between host state and sensor manifests.
type Engine struct {
	manifests []*schema.SensorManifest
}

// NewEngine creates a new discovery engine.
func NewEngine(manifests []*schema.SensorManifest) *Engine {
	return &Engine{
		manifests: manifests,
	}
}

// GetRecommendations analyzes the host state and returns matching sensor recommendations.
// Uses correlated discovery: a trigger requires both process AND port (if both specified).
// systemState is used to filter out already-deployed sensors (idempotency).
func (e *Engine) GetRecommendations(hostState *scanner.HostState, systemState *system.SystemState) []*Recommendation {
	var recommendations []*Recommendation

	// Build deployed image set for idempotency checks
	deployedImageSet := make(map[string]bool)
	if systemState != nil {
		for _, image := range systemState.DeployedImages {
			deployedImageSet[image] = true
		}
	}

	// Iterate through manifests and find correlated matches
	for _, manifest := range e.manifests {
		imageStr := manifest.Deployment.ImageRepository
		if manifest.Deployment.ImageTag != "" {
			imageStr += ":" + manifest.Deployment.ImageTag
		} else {
			imageStr += ":latest"
		}
		if manifest.Deployment.ImageDigest != "" {
			imageStr += "@" + manifest.Deployment.ImageDigest
		}

		// Idempotency check: skip if this sensor is already deployed
		if deployedImageSet[imageStr] {
			continue
		}

		matchedServices := []MatchedService{}

		// Check for universal trigger (empty processes AND empty ports)
		isUniversalTrigger := len(manifest.Heuristics.Triggers.Processes) == 0 &&
			len(manifest.Heuristics.Triggers.Ports) == 0

		if isUniversalTrigger {
			// Universal trigger: always recommend this sensor
			rec := &Recommendation{
				SensorID:           manifest.ID,
				SensorName:         manifest.Name,
				Reason:             manifest.Heuristics.RecommendationReason,
				MatchedServices:    []MatchedService{}, // No specific services matched
				DeploymentTemplate: &manifest.Deployment,
				Manifest:           manifest,
			}
			recommendations = append(recommendations, rec)
			continue
		}

		// Regular matching: iterate through each discovered service
		for _, svc := range hostState.Services {
			if e.matches(manifest, svc) {
				matchedServices = append(matchedServices, MatchedService{
					ProcessName: svc.ProcessName,
					Port:        svc.Port,
					PID:         svc.PID,
				})
			}
		}

		// If we found matching services, create a recommendation
		if len(matchedServices) > 0 {
			rec := &Recommendation{
				SensorID:           manifest.ID,
				SensorName:         manifest.Name,
				Reason:             manifest.Heuristics.RecommendationReason,
				MatchedServices:    matchedServices,
				DeploymentTemplate: &manifest.Deployment,
				Manifest:           manifest,
			}
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// matches checks if a service matches a manifest's heuristics using AND logic.
// If manifest specifies both processes and ports: both must match.
// If manifest specifies only processes: process must match.
// If manifest specifies only ports: port must match.
func (e *Engine) matches(manifest *schema.SensorManifest, svc scanner.Service) bool {
	triggers := manifest.Heuristics.Triggers

	hasProcessTriggers := len(triggers.Processes) > 0
	hasPortTriggers := len(triggers.Ports) > 0

	// If no triggers at all, don't match
	if !hasProcessTriggers && !hasPortTriggers {
		return false
	}

	// Check process match
	processMatches := false
	if hasProcessTriggers {
		for _, triggerProc := range triggers.Processes {
			if svc.ProcessName == triggerProc {
				processMatches = true
				break
			}
		}
	} else {
		// No process triggers specified, so process check passes
		processMatches = true
	}

	// Check port match
	portMatches := false
	if hasPortTriggers {
		for _, triggerPort := range triggers.Ports {
			if svc.Port == triggerPort {
				portMatches = true
				break
			}
		}
	} else {
		// No port triggers specified, so port check passes
		portMatches = true
	}

	// Both must be true (AND logic)
	return processMatches && portMatches
}
