package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/honeywire/wizard/internal/cli"
)

const (
	DeployDir      = "/opt/honeywire/sensors"
	ComposeFile    = "honeywire-compose.yml"
	ProjectName    = "honeywire"
	ComposeMinVer  = "5.0.0"
	commandTimeout = 10 * time.Second
)

type composeInfo struct {
	Version string `json:"version"`
}

func GetDockerCommand() ([]string, error) {
	// ValidateDockerState now performs the full check:
	// 1. Docker binary check
	// 2. Daemon responsiveness (now handled by checkDaemon)
	// 3. Compose version verification
	cmd, err := ValidateDockerState()
	if err != nil {
		// Because ValidateDockerState returns an error already formatted
		// by generateRemediationError, we can return it directly.
		return nil, err
	}

	return cmd, nil
}

func ValidateDockerState() ([]string, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, generateRemediationError("Docker Engine is not installed.", "", err)
	}

	if err := checkDaemon(); err != nil {
		return nil, err
	}

	cmd, err := validateComposeVersion()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func checkDaemon() error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	var stdout, stderr bytes.Buffer

	// codeql[go/command-injection] Hardcoded/trusted CLI arguments.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{json .}}")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		mainReason := "Docker daemon is unresponsive. Ensure it is running."
		details := strings.TrimSpace(stderr.String())
		// We pass the actual system error to generateRemediationError now
		return generateRemediationError(mainReason, details, err)
	}

	var info struct {
		SecurityOptions []string `json:"SecurityOptions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &info); err == nil {
		for _, opt := range info.SecurityOptions {
			if strings.Contains(strings.ToLower(opt), "rootless") {
				return generateRemediationError(
					"Docker is running in Rootless mode.",
					"HoneyWire sensors require raw network access and iptables manipulation, which cannot be done in Rootless mode. Please install a standard Rootful Docker daemon.",
					nil,
				)
			}
		}
	}

	return nil
}

func validateComposeVersion() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	var outb, errb bytes.Buffer

	// codeql[go/command-injection] Hardcoded/trusted CLI arguments.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, "docker", "compose", "version", "--format", "json")
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	if err := cmd.Run(); err != nil {
		mainReason := "Docker Compose plugin is missing or malfunctioning."
		details := strings.TrimSpace(errb.String())
		return nil, generateRemediationError(mainReason, details, err)
	}

	var info composeInfo
	if err := json.Unmarshal(outb.Bytes(), &info); err != nil {
		return nil, fmt.Errorf("failed to parse docker compose version output: %w", err)
	}

	cleanVer := strings.TrimPrefix(info.Version, "v")

	if !isVersionSufficient(cleanVer, ComposeMinVer) {
		mainReason := fmt.Sprintf("Docker Compose is outdated (v%s). v%s+ is strictly required.", cleanVer, ComposeMinVer)
		return nil, generateRemediationError(mainReason, "", nil)
	}

	return []string{"docker", "compose"}, nil
}

// generateRemediationError formats an actionable error message.
func generateRemediationError(mainReason string, details string, originalErr error) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n    %s❌ Deployment Aborted: %s%s\n", cli.Red, mainReason, cli.Reset))

	if details != "" {
		sb.WriteString(fmt.Sprintf("    %sDetails: %s%s\n", cli.Dim, details, cli.Reset))
	}

	// Inject the real system error (e.g., "permission denied") so you don't lose debugging context
	if originalErr != nil {
		sb.WriteString(fmt.Sprintf("    %sSystem Error: %v%s\n", cli.Dim, originalErr, cli.Reset))
	}

	sb.WriteString(fmt.Sprintf("    %sHoneyWire requires Docker Compose v%s+ to securely orchestrate honeypot lifecycles.%s\n\n", cli.Red, ComposeMinVer, cli.Reset))

	sb.WriteString(fmt.Sprintf("    %sPlease run the official upgrade command for your OS:%s\n", cli.Cyan, cli.Reset))
	sb.WriteString("      Ubuntu/Debian:  sudo apt-get update && sudo apt-get install docker-compose-plugin\n")
	sb.WriteString("      Fedora:         sudo dnf upgrade docker-compose-plugin\n")
	sb.WriteString("      Arch:           sudo pacman -Syu docker-compose\n\n")

	sb.WriteString(fmt.Sprintf("    %sIf you installed Docker manually, update the binary:%s\n", cli.Cyan, cli.Reset))
	sb.WriteString("      mkdir -p ~/.docker/cli-plugins && \\ \n")
	sb.WriteString(fmt.Sprintf("      curl -SL https://github.com/docker/compose/releases/download/%s/docker-compose-linux-$(uname -m) \\ -o ~/.docker/cli-plugins/docker-compose && \\ \n", ComposeMinVer))
	sb.WriteString("      chmod +x ~/.docker/cli-plugins/docker-compose\n")

	return errors.New(sb.String())
}

// -----------------------------------------------------------------------------
// UTILITIES
// -----------------------------------------------------------------------------

func isVersionSufficient(current, minimum string) bool {
	currParts := parseVersionString(current)
	minParts := parseVersionString(minimum)

	for i := 0; i < 3; i++ {
		if currParts[i] > minParts[i] {
			return true
		}
		if currParts[i] < minParts[i] {
			return false
		}
	}
	return true
}

func parseVersionString(v string) [3]int {
	var parts [3]int
	segments := strings.Split(strings.TrimPrefix(v, "v"), ".")

	for i := 0; i < len(segments) && i < 3; i++ {
		cleanSeg := strings.Split(segments[i], "-")[0]
		if val, err := strconv.Atoi(cleanSeg); err == nil {
			parts[i] = val
		}
	}
	return parts
}
