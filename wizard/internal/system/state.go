package system

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

func CheckRoot() (string, error) {
	if os.Geteuid() != 0 {
		return "⚠️ Wizard is not running as root (UID 0). Some processes or sockets may be hidden from /proc analysis.", nil
	}
	return "", nil
}

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

func CheckDiskSpace() (string, error) {
	var stat syscall.Statfs_t
	path := "/var/lib/docker"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = "/"
	}

	if err := syscall.Statfs(path, &stat); err != nil {
		return "", fmt.Errorf("failed to statfs %s: %w", path, err)
	}

	freeBytes := stat.Bavail * uint64(stat.Bsize)
	freeGB := float64(freeBytes) / (1024 * 1024 * 1024)

	if freeGB < 1.0 {
		return fmt.Sprintf("⚠️ Low disk space detected (%.2f GB free on %s). Docker deployment requires at least 1GB.", freeGB, path), nil
	}
	return "", nil
}

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
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				memKB, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					continue
				}
				memMB := memKB / 1024

				if memMB < 500 {
					return "⚠️  Low memory detected (<500MB). Deploying multiple sensors may impact host performance.", nil
				}
			}
			break
		}
	}

	return "", nil
}

func LoadCurrentState() (*SystemState, error) {
	state := &SystemState{
		DeployedImages: []string{},
		ManagedPorts:   []int{},
	}

	composeFile := "honeywire-compose.yml"
	data, err := os.ReadFile(composeFile)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", composeFile, err)
	}

	var compose map[string]interface{}
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", composeFile, err)
	}

	services, ok := compose["services"].(map[string]interface{})
	if !ok {
		return state, nil
	}

	deployedImageSet := make(map[string]bool)
	managedPortSet := make(map[int]bool)

	for _, svcInterface := range services {
		svc, ok := svcInterface.(map[string]interface{})
		if !ok {
			continue
		}

		if image, ok := svc["image"].(string); ok {
			deployedImageSet[image] = true
		}

		if ports, ok := svc["ports"].([]interface{}); ok {
			for _, portInterface := range ports {
				portStr, ok := portInterface.(string)
				if !ok {
					continue
				}

				parts := strings.Split(portStr, ":")
				if len(parts) >= 1 {
					hostPortStr := strings.Split(parts[0], "/")[0]
					if hostPort, err := strconv.Atoi(hostPortStr); err == nil {
						managedPortSet[hostPort] = true
					}
				}
			}
		}
	}

	for image := range deployedImageSet {
		state.DeployedImages = append(state.DeployedImages, image)
	}

	for port := range managedPortSet {
		state.ManagedPorts = append(state.ManagedPorts, port)
	}

	return state, nil
}
