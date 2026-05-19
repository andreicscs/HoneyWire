package generator

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/honeywire/wizard/pkg/autodiscovery"
)

const (
	Reset  = "\033[0m"
	Dim    = "\033[2m"
	Cyan   = "\033[36m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
)

const DeployDir = "/opt/honeywire/sensors"
const ComposeFile = "honeywire-compose.yml"
const ProjectName = "honeywire"

// Match the Hub payload
type DeployableSensor struct {
	SensorID  string                 `json:"sensor_id"`
	EnvValues map[string]string      `json:"env_values"`
	Manifest  map[string]interface{} `json:"manifest"` 
}

type ComposeReq struct {
	HubEndpoint string             `json:"hub_endpoint"`
	HubKey      string             `json:"hub_key"`
	Sensors     []DeployableSensor `json:"sensors"`
}

// DockerCompose represents the structure of a modern Compose file.
// Note: 'Version' is intentionally omitted as it is obsolete in Compose V2.
type DockerCompose struct {
	Services map[string]Service `yaml:"services"`
}

type Service struct {
	Image         string   `yaml:"image"`
	ContainerName string   `yaml:"container_name"`
	Restart       string   `yaml:"restart"`
	NetworkMode   string   `yaml:"network_mode,omitempty"`
	CapAdd        []string `yaml:"cap_add,omitempty"`
	Environment   []string `yaml:"environment,omitempty"`
	Volumes       []string `yaml:"volumes,omitempty"`
	Ports         []string `yaml:"ports,omitempty"`
}

// GetDockerCommand determines the correct compose command, handling missing dependencies securely.
func GetDockerCommand() ([]string, error) {
	// 1. Check if Docker CLI is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return handleMissingDocker()
	}

	// 2. Check if Docker Daemon is actively running
	if err := checkDaemon(); err != nil {
		return nil, err
	}

	// 3. Look for an existing Compose installation
	if cmd, err := findCompose(); err == nil {
		return cmd, nil
	}

	// 4. Compose is missing. Attempt secure installation.
	return installCompose()
}

// --- HELPER FUNCTIONS ---

func checkDaemon() error {
	if err := exec.Command("docker", "info").Run(); err != nil {
		return fmt.Errorf("Docker is installed, but the daemon is not running.\nPlease start it (e.g., 'sudo systemctl start docker') and try again")
	}
	return nil
}

func findCompose() ([]string, error) {
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		return []string{"docker", "compose"}, nil
	}
	if err := exec.Command("docker-compose", "version").Run(); err == nil {
		return []string{"docker-compose"}, nil
	}
	return nil, fmt.Errorf("compose not found")
}

func installCompose() ([]string, error) {
	fmt.Printf("\n    %s⚠️ Docker is running, but the Compose plugin is missing.%s\n", Yellow, Reset)

	// Map Go architecture to Docker release binaries
	arch := runtime.GOARCH
	var dockerArch string
	switch arch {
	case "amd64":
		dockerArch = "x86_64"
	case "arm64":
		dockerArch = "aarch64"
	default:
		// We don't crash the wizard! We just inform them we can't auto-install.
		return nil, fmt.Errorf("automatic compose installation is not supported for architecture '%s'. Please install docker-compose manually", arch)
	}

	fmt.Printf("    Would you like HoneyWire to securely install the official Docker Compose plugin for %s? [y/N]: ", dockerArch)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) != "y" {
		return nil, fmt.Errorf("deployment aborted: Docker Compose plugin is required")
	}

	fmt.Printf("\n    %s⚙️ Verifying checksums and installing official plugin...%s\n", Cyan, Reset)

	version := "v2.26.1"
	binaryName := fmt.Sprintf("docker-compose-linux-%s", dockerArch)
	baseURL := fmt.Sprintf("https://github.com/docker/compose/releases/download/%s", version)

	secureInstallCmd := fmt.Sprintf(`
		set -e
		cd /tmp
		echo "↳ Downloading binary..."
		curl -sSL "%s/%s" -o %s
		echo "↳ Downloading official signature..."
		curl -sSL "%s/%s.sha256" -o %s.sha256
		echo "↳ Verifying cryptographic checksum..."
		sha256sum -c %s.sha256
		echo "↳ Installing to Docker CLI plugins..."
		mkdir -p /usr/local/lib/docker/cli-plugins
		mv %s /usr/local/lib/docker/cli-plugins/docker-compose
		chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
		rm %s.sha256
	`, baseURL, binaryName, binaryName, baseURL, binaryName, binaryName, binaryName, binaryName, binaryName)

	cmd := exec.Command("sh", "-c", secureInstallCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("security verification or installation failed: %v", err)
	}

	fmt.Printf("    %s✅ Secure installation successful!%s\n", Green, Reset)
	return []string{"docker", "compose"}, nil
}

func handleMissingDocker() ([]string, error) {
	fmt.Printf("\n    %s⚠️ Docker is not installed on this system.%s\n", Yellow, Reset)
	fmt.Printf("    %sHoneyWire requires Docker to safely isolate the honeypot sensors.%s\n", Dim, Reset)
	fmt.Printf("    Would you like to automatically install Docker using the official Docker convenience script? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) != "y" {
		return nil, fmt.Errorf("deployment aborted: Docker is required to isolate sensors")
	}

	fmt.Printf("\n    %s⚙️ Downloading and running official Docker install script...%s\n", Cyan, Reset)

	cmd := exec.Command("sh", "-c", "curl -fsSL https://get.docker.com | sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to install Docker: %v", err)
	}

	fmt.Printf("    %s✅ Docker installed successfully!%s\n", Green, Reset)
	// Modern Docker convenience script installs docker-compose-plugin by default
	return []string{"docker", "compose"}, nil
}


// --- CORE ACTIONS ---

// Apply fetches the compartmentalized compose file, modifies it with it's environment based generated values, and spins the containers up.
func Apply(recs []*autodiscovery.Recommendation, hubURL, nodeKey string) error {
	cmdBase, err := GetDockerCommand()
	if err != nil {
		return err
	}

	payload := ComposeReq{
		HubEndpoint: hubURL,
		HubKey:      nodeKey,
		Sensors:     []DeployableSensor{},
	}

	for _, rec := range recs {
		manifestMap := make(map[string]interface{})
		manifestBytes, _ := json.Marshal(rec.Manifest)
		json.Unmarshal(manifestBytes, &manifestMap)

		envVals := make(map[string]string)
		for _, env := range rec.DeploymentTemplate.EnvVars {
			envVals[env.Name] = env.Default // Use default as starting point
		}

		payload.Sensors = append(payload.Sensors, DeployableSensor{
			SensorID:  rec.SensorID,
			EnvValues: envVals,
			Manifest:  manifestMap,
		})
	}

	bodyBytes, _ := json.Marshal(payload)
	endpoint := strings.TrimRight(hubURL, "/") + "/api/v1/compose/generate"
	
	req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+nodeKey) // Inject Node Auth

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to compile deployment script from hub (Status %d): %v", resp.StatusCode, err)
	}
	defer resp.Body.Close()

	yamlData, _ := io.ReadAll(resp.Body)

	if err := os.MkdirAll(DeployDir, 0750); err != nil {
		return fmt.Errorf("failed to create deployment directory: %w", err)
	}
	composePath := filepath.Join(DeployDir, ComposeFile)
	if err := os.WriteFile(composePath, yamlData, 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", ComposeFile, err)
	}

	args := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "up", "-d", "--remove-orphans")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = DeployDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose failed: %s\nOutput: %s", err, string(output))
	}

	return nil
}

// Uninstall safely tears down the isolated namespace and cleans up the config
func Uninstall() error {
	cmdBase, err := GetDockerCommand()
	if err != nil {
		return err
	}

	composePath := filepath.Join(DeployDir, ComposeFile)
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("no active deployment found at %s", composePath)
	}

	args := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "down", "-v", "--remove-orphans", "--rmi", "all")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = DeployDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to tear down sensors: %s\nOutput: %s", err, string(output))
	}

	// Clean up compose file
	os.Remove(composePath)

	// Clean up DeployDir if it's empty
	if entries, _ := os.ReadDir(DeployDir); len(entries) == 0 {
		os.Remove(DeployDir)
	}

	return nil
}