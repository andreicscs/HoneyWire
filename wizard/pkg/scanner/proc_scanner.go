package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/honeywire/wizard/pkg/state"
)

// ProcScanner implements the Scanner interface by reading from /proc.
// It correlates processes to ports via inode matching.
type ProcScanner struct {
	ignoreList map[string]bool
}

// NewProcScanner creates a new proc-based scanner.
func NewProcScanner() *ProcScanner {
	// Ignore HoneyWire and noisy OS processes to reduce false positives
	ignoreList := map[string]bool{
		"honeywire-hub":   true,
		"tcp-tarpit":      true,
		"web-decoy":       true,
		"file-canary":     true,
		"icmp-canary":     true,
		"scan-detector":   true,
		"wizard":          true,
		"bash":            true,
		"systemd":         true,
		"init":            true,
		"systemd-journal": true,
		"systemd-udevd":   true,
		"systemd-resolve": true,
		"systemd-logind":  true,
		"systemd-timesyn": true,
		"rsyslogd":        true,
		"dbus-daemon":     true,
		"cron":            true,
		"agetty":          true,
		"login":           true,
		"containerd":      true,
		"containerd-shim": true,
		"dockerd":         true,
		"sh":              true,
		"unattended-upgr": true,
		"polkitd":         true,
		"gopls":           true,
		"node":            true,
		"go":              true,
	}
	return &ProcScanner{
		ignoreList: ignoreList,
	}
}

// Scan reads /proc to discover correlated processes and ports.
// systemState is used to filter out already-managed services and ports.
func (p *ProcScanner) Scan(systemState *state.SystemState) (*HostState, error) {
	// Build a set of managed ports to skip
	managedPortMap := make(map[int]bool)
	if systemState != nil {
		for _, port := range systemState.ManagedPorts {
			managedPortMap[port] = true
		}
	}

	// Step 1: Build a map of socket_inode -> port from /proc/net/tcp(6)
	inodeToPort := p.buildInodePortMap()
	if len(inodeToPort) == 0 {
		return &HostState{Services: []Service{}}, nil
	}

	// Step 2: Iterate through PIDs and correlate to ports via /proc/[pid]/fd
	var services []Service
	seen := make(map[string]bool) // For deduplication: "processName:port"

	procDir := "/proc"
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pidStr := entry.Name()
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// Read process name from /proc/[pid]/comm
		commPath := filepath.Join(procDir, pidStr, "comm")
		commData, err := os.ReadFile(commPath)
		if err != nil {
			continue
		}

		procName := strings.TrimSpace(string(commData))
		if procName == "" || p.ignoreList[procName] {
			continue
		}

		// Iterate through /proc/[pid]/fd/ to find socket inodes
		fdDir := filepath.Join(procDir, pidStr, "fd")
		fdEntries, err := os.ReadDir(fdDir)
		if err != nil {
			// Permission denied, skip this process
			continue
		}

		for _, fdEntry := range fdEntries {
			fdPath := filepath.Join(fdDir, fdEntry.Name())
			// Read the symlink target (e.g., "socket:[12345]")
			target, err := os.Readlink(fdPath)
			if err != nil {
				continue
			}

			// Check if this is a socket and extract the inode
			if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
				inodeStr := target[8 : len(target)-1] // Extract "12345" from "socket:[12345]"
				if port, exists := inodeToPort[inodeStr]; exists {
					// Skip if this port is managed (already deployed by wizard)
					if managedPortMap[port] {
						continue
					}

					// Deduplicate: check if we already have this process:port combination
					dedupeKey := fmt.Sprintf("%s:%d", procName, port)
					if seen[dedupeKey] {
						continue
					}
					seen[dedupeKey] = true

					// Found a correlated service!
					services = append(services, Service{
						ProcessName: procName,
						Port:        port,
						PID:         pid,
					})
					// Don't break - a process might be listening on multiple ports
				}
			}
		}
	}

	return &HostState{
		Services: services,
	}, nil
}

// buildInodePortMap creates a map of socket_inode -> port by parsing /proc/net/tcp and /proc/net/tcp6
func (p *ProcScanner) buildInodePortMap() map[string]int {
	inodeToPort := make(map[string]int)
	netFiles := []string{"/proc/net/tcp", "/proc/net/tcp6"}

	for _, netFile := range netFiles {
		file, err := os.Open(netFile)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			if lineNum == 1 {
				// Skip header
				continue
			}

			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 10 {
				continue
			}

			// Field 3 is state (0A = LISTEN)
			state := fields[3]
			if state != "0A" {
				continue
			}

			// Field 1 is local address (host:port in hex)
			localAddr := fields[1]
			parts := strings.Split(localAddr, ":")
			if len(parts) != 2 {
				continue
			}

			portHex := parts[1]
			portInt, err := strconv.ParseInt(portHex, 16, 32)
			if err != nil {
				continue
			}

			// Field 9 is the inode number
			inode := fields[9]
			inodeToPort[inode] = int(portInt)
		}
	}

	return inodeToPort
}
