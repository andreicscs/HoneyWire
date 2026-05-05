package state

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

// SystemState represents the current honeywire deployment state.
type SystemState struct {
	DeployedImages []string // Images currently deployed via honeywire-compose.yml
	ManagedPorts   []int    // Ports managed by deployed honeywire sensors
}

// CheckRoot returns a warning if the current user is not root.
func CheckRoot() (string, error) {
	if os.Geteuid() != 0 {
		return "⚠️ Wizard is not running as root (UID 0). Some processes or sockets may be hidden from /proc analysis.", nil
	}
	return "", nil
}

// CheckLoad returns a warning if the 1-minute load average is exceptionally high (e.g., > 4.0).
func CheckLoad() (string, error) {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/loadavg: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			load1, err := strconv.ParseFloat(fields[0], 64)
			if err == nil && load1 > 4.0 {
				return fmt.Sprintf("⚠️ High CPU load detected (%.2f). Deploying containers may impact host stability.", load1), nil
			}
		}
	}
	return "", nil
}

// CheckDiskSpace returns a warning if the root filesystem has less than 1GB of free space.
func CheckDiskSpace() (string, error) {
	var stat syscall.Statfs_t
	// Check /var/lib/docker if it exists, otherwise check /
	path := "/var/lib/docker"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = "/"
	}

	if err := syscall.Statfs(path, &stat); err != nil {
		return "", fmt.Errorf("failed to statfs %s: %w", path, err)
	}

	// Available space in bytes: blocks available * block size
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	freeGB := float64(freeBytes) / (1024 * 1024 * 1024)

	if freeGB < 1.0 {
		return fmt.Sprintf("⚠️ Low disk space detected (%.2f GB free on %s). Docker deployment requires at least 1GB.", freeGB, path), nil
	}
	return "", nil
}

// CheckMemory returns a warning if available memory is below 500MB.
func CheckMemory() (string, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemAvailable:") {
			// Parse "MemAvailable:  12345678 kB" -> extract the number
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				memKB, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					continue
				}
				memMB := memKB / 1024

				if memMB < 500 {
					return "⚠️ Low memory detected (<500MB). Deploying multiple sensors may impact host performance.", nil
				}
			}
			break
		}
	}

	return "", nil
}

// LoadCurrentState reads honeywire-compose.yml and extracts deployed state.
// If the file doesn't exist, it returns an empty SystemState.
func LoadCurrentState() (*SystemState, error) {
	state := &SystemState{
		DeployedImages: []string{},
		ManagedPorts:   []int{},
	}

	// Try to read honeywire-compose.yml
	composeFile := "honeywire-compose.yml"
	data, err := os.ReadFile(composeFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return empty state gracefully
			return state, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", composeFile, err)
	}

	// Parse YAML
	var compose map[string]interface{}
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", composeFile, err)
	}

	// Extract services
	services, ok := compose["services"].(map[string]interface{})
	if !ok {
		// No services key, return empty state
		return state, nil
	}

	deployedImageSet := make(map[string]bool)
	managedPortSet := make(map[int]bool)

	// Iterate through each service
	for _, svcInterface := range services {
		svc, ok := svcInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract image
		if image, ok := svc["image"].(string); ok {
			deployedImageSet[image] = true
		}

		// Extract ports
		if ports, ok := svc["ports"].([]interface{}); ok {
			for _, portInterface := range ports {
				portStr, ok := portInterface.(string)
				if !ok {
					continue
				}

				// Parse "8888:80" or "8888:80/tcp" format -> extract host port (8888)
				parts := strings.Split(portStr, ":")
				if len(parts) >= 1 {
					hostPortStr := strings.Split(parts[0], "/")[0] // Remove protocol if present
					if hostPort, err := strconv.Atoi(hostPortStr); err == nil {
						managedPortSet[hostPort] = true
					}
				}
			}
		}
	}

	// Convert sets to slices
	for image := range deployedImageSet {
		state.DeployedImages = append(state.DeployedImages, image)
	}

	for port := range managedPortSet {
		state.ManagedPorts = append(state.ManagedPorts, port)
	}

	return state, nil
}
