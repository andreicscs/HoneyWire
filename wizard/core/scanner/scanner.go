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

// MockScanner simulates discovering correlated services.
// For the MVP, it returns: postgres on 5432, nginx on 80
type MockScanner struct {
	hostState *HostState
}

// NewMockScanner creates a new mock scanner.
func NewMockScanner() *MockScanner {
	return &MockScanner{
		hostState: &HostState{
			Services: []Service{
				{ProcessName: "postgres", Port: 5432, PID: 1234},
				{ProcessName: "nginx", Port: 80, PID: 5678},
			},
		},
	}
}

// Scan returns the simulated host state.
// systemState is used to filter out already-managed services.
func (m *MockScanner) Scan(systemState *system.SystemState) (*HostState, error) {
	if systemState == nil {
		return m.hostState, nil
	}
	// Filter out any services on managed ports
	var filtered []Service
	managedPortMap := make(map[int]bool)
	for _, p := range systemState.ManagedPorts {
		managedPortMap[p] = true
	}
	for _, svc := range m.hostState.Services {
		if !managedPortMap[svc.Port] {
			filtered = append(filtered, svc)
		}
	}
	return &HostState{Services: filtered}, nil
}
