package scanner

import "github.com/honeywire/wizard/internal/system"

// Service represents a correlated process-port binding on the host.
type Service struct {
	ProcessName string // e.g., "nginx", "postgres"
	Port        int    // e.g., 80, 5432
	PID         int    // Process ID
}

// HostState represents the current state of the host with correlated services.
type HostState struct {
	Services []Service
}

// Scanner is the interface for discovering host state.
// Implementations should be read-only and not modify host state.
// systemState allows filtering out already-deployed sensors to prevent rescanning own services.
type Scanner interface {
	Scan(systemState *system.SystemState) (*HostState, error)
}