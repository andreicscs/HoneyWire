package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"bufio"


	"github.com/honeywire/wizard/pkg/autodiscovery"
	"gopkg.in/yaml.v3"
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

// DockerCompose represents the structure of a v3 compose file
type DockerCompose struct {
	Version  string             `yaml:"version"`
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

// getDockerCommand determines the correct compose command and gracefully handles missing dependencies
func getDockerCommand() ([]string, error) {
	// 1. Check if the base 'docker' command exists on the host
	_, err := exec.LookPath("docker")
	if err == nil {
		if err := exec.Command("docker", "info").Run(); err != nil {
			return nil, fmt.Errorf("Docker is installed, but the daemon is not running.\nPlease start it (e.g., 'sudo systemctl start docker') and try again")
		}

		if err := exec.Command("docker", "compose", "version").Run(); err == nil {
			return []string{"docker", "compose"}, nil
		}
		if err := exec.Command("docker-compose", "version").Run(); err == nil {
			return []string{"docker-compose"}, nil
		}

		// --- NEW: Cryptographically Secure Plugin Installation ---
		fmt.Printf("\n    %s⚠️ Docker is running, but the Compose plugin is missing.%s\n", Yellow, Reset)
		fmt.Printf("    Would you like HoneyWire to securely install the official Docker Compose plugin? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "y" || input == "yes" {
			fmt.Printf("\n    %s⚙️ Verifying checksums and installing official plugin...%s\n", Cyan, Reset)
			
			// Secure bash script: Downloads original name, verifies, then renames and installs.
			secureInstallCmd := `
				set -e
				cd /tmp
				echo "↳ Downloading binary..."
				curl -sSL "https://github.com/docker/compose/releases/download/v2.26.1/docker-compose-linux-x86_64" -o docker-compose-linux-x86_64
				echo "↳ Downloading official signature..."
				curl -sSL "https://github.com/docker/compose/releases/download/v2.26.1/docker-compose-linux-x86_64.sha256" -o docker-compose-linux-x86_64.sha256
				echo "↳ Verifying cryptographic checksum..."
				sha256sum -c docker-compose-linux-x86_64.sha256
				echo "↳ Installing to Docker CLI plugins..."
				mkdir -p /usr/local/lib/docker/cli-plugins
				mv docker-compose-linux-x86_64 /usr/local/lib/docker/cli-plugins/docker-compose
				chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
				rm docker-compose-linux-x86_64.sha256
			`
			cmd := exec.Command("sh", "-c", secureInstallCmd)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("security verification or installation failed: %v", err)
			}
			
			fmt.Printf("    %s✅ Secure installation successful!%s\n", Green, Reset)
			return []string{"docker", "compose"}, nil
		}

		return nil, fmt.Errorf("deployment aborted: Docker Compose plugin is required")
	}

	// 2. Base Docker Installation (Uses Official Convenience Script which validates GPG keys internally)
	fmt.Printf("\n    %s⚠️ Docker is not installed on this system.%s\n", Yellow, Reset)
	fmt.Printf("    %sHoneyWire requires Docker to safely isolate the honeypot sensors.%s\n", Dim, Reset)
	fmt.Printf("    Would you like to automatically install Docker using the official convenience script? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		fmt.Printf("\n    %s⚙️ Downloading and running official Docker install script...%s\n", Cyan, Reset)
		
		cmd := exec.Command("sh", "-c", "curl -fsSL https://get.docker.com | sh")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to install Docker: %v", err)
		}
		
		fmt.Printf("    %s✅ Docker installed successfully!%s\n", Green, Reset)
		return []string{"docker", "compose"}, nil
	}

	return nil, fmt.Errorf("deployment aborted: Docker is required to isolate sensors")
}

// Apply writes the compartmentalized compose file and brings the containers up.
func Apply(recs []*autodiscovery.Recommendation) error {
	cmdBase, err := getDockerCommand()
	if err != nil {
		return err
	}

	compose := DockerCompose{
		Version:  "3.8",
		Services: make(map[string]Service),
	}

	for _, rec := range recs {
		svc := Service{
			Image:         rec.DeploymentTemplate.Image,
			ContainerName: rec.SensorID,
			Restart:       "unless-stopped",
			NetworkMode:   rec.DeploymentTemplate.NetworkMode,
			CapAdd:        rec.DeploymentTemplate.CapAdd,
		}

		for _, env := range rec.DeploymentTemplate.EnvVars {
			val := strings.TrimSpace(env.Default)
			if val != "" {
				svc.Environment = append(svc.Environment, fmt.Sprintf("%s=%s", env.Name, val))
			}
		}

		for _, vol := range rec.DeploymentTemplate.VolumeMounts {
			mount := fmt.Sprintf("%s:%s", vol.Source, vol.Target)
			if vol.ReadOnly {
				mount += ":ro"
			}
			svc.Volumes = append(svc.Volumes, mount)
		}

		if svc.NetworkMode != "host" {
			for _, p := range rec.DeploymentTemplate.PortAssignments {
				svc.Ports = append(svc.Ports, fmt.Sprintf("%d:%d", p.Default, p.Default))
			}
		}

		compose.Services[rec.SensorID] = svc
	}

	yamlData, err := yaml.Marshal(&compose)
	if err != nil {
		return fmt.Errorf("failed to marshal compose data: %w", err)
	}

	if err := os.MkdirAll(DeployDir, 0750); err != nil {
		return fmt.Errorf("failed to create deployment directory: %w", err)
	}

	composePath := filepath.Join(DeployDir, ComposeFile)
	if err := os.WriteFile(composePath, yamlData, 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", ComposeFile, err)
	}

	// Execute: docker compose -f honeywire-compose.yml -p honeywire up -d --remove-orphans
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
	cmdBase, err := getDockerCommand()
	if err != nil {
		return err 
	}

	composePath := filepath.Join(DeployDir, ComposeFile)
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("no active deployment found at %s", composePath)
	}

	// Execute: docker compose -f honeywire-compose.yml -p honeywire down -v --remove-orphans
	args := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "down", "-v", "--remove-orphans")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = DeployDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to tear down sensors: %s\nOutput: %s", err, string(output))
	}

	return os.Remove(composePath)
}