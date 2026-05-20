package deploy

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/honeywire/wizard/internal/cli"
)

const commandTimeout = 30 * time.Second

func GetDockerCommand() ([]string, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return handleMissingDocker()
	}

	if err := checkDaemon(); err != nil {
		return nil, err
	}

	if cmd, err := findCompose(); err == nil {
		return cmd, nil
	}

	return installCompose()
}

func checkDaemon() error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker is installed, but the daemon is not running.\nPlease start it (e.g., 'sudo systemctl start docker') and try again")
	}
	return nil
}

func findCompose() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if cmd := exec.CommandContext(ctx, "docker", "compose", "version"); cmd.Run() == nil {
		return []string{"docker", "compose"}, nil
	}
	if cmd := exec.CommandContext(ctx, "docker-compose", "version"); cmd.Run() == nil {
		return []string{"docker-compose"}, nil
	}
	return nil, fmt.Errorf("compose not found")
}

func installCompose() ([]string, error) {
	fmt.Printf("\n    %s⚠️ Docker is running, but the Compose plugin is missing.%s\n\n", cli.Yellow, cli.Reset)

	fmt.Printf("    Recommended install methods:\n")
	fmt.Printf("      Ubuntu/Debian:  sudo apt-get install docker-compose-plugin\n")
	fmt.Printf("      Fedora:         sudo dnf install docker-compose-plugin\n")
	fmt.Printf("      Arch:           sudo pacman -S docker-compose\n")
	fmt.Printf("      Manual:         https://docs.docker.com/compose/install/\n\n")

	arch := runtime.GOARCH
	var dockerArch string
	switch arch {
	case "amd64":
		dockerArch = "x86_64"
	case "arm64":
		dockerArch = "aarch64"
	default:
		return nil, fmt.Errorf("automatic Compose installation is not available for architecture '%s'.\nPlease install docker-compose manually: https://docs.docker.com/compose/install/", arch)
	}

	fmt.Printf("    Download and install the official Compose plugin binary (%s) after SHA-256 verification? [y/N]: ", dockerArch)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	if strings.TrimSpace(strings.ToLower(input)) != "y" {
		return nil, fmt.Errorf("deployment aborted: Docker Compose plugin is required")
	}

	fmt.Printf("\n    %s⚙️ Downloading and verifying official Compose plugin...%s\n", cli.Cyan, cli.Reset)

	binaryName := fmt.Sprintf("docker-compose-linux-%s", dockerArch)
	binaryURL := fmt.Sprintf("https://github.com/docker/compose/releases/download/%s/%s", ComposeVersion, binaryName)

	result, err := FetchWithRemoteChecksum(binaryURL)
	if err != nil {
		return nil, fmt.Errorf("download or verification failed: %w", err)
	}
	defer result.Cleanup()

	pluginDir := "/usr/local/lib/docker/cli-plugins"
	destPath := filepath.Join(pluginDir, "docker-compose")

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}

	if err := copyFile(result.Path, destPath); err != nil {
		return nil, fmt.Errorf("failed to install compose binary: %w", err)
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to set compose binary permissions: %w", err)
	}

	fmt.Printf("    %s✅ Compose plugin installed.%s\n", cli.Green, cli.Reset)
	return []string{"docker", "compose"}, nil
}

func handleMissingDocker() ([]string, error) {
	fmt.Printf("\n    %s⚠️ Docker is not installed on this system.%s\n", cli.Yellow, cli.Reset)
	fmt.Printf("    %sHoneyWire requires Docker to safely isolate the honeypot sensors.%s\n\n", cli.Dim, cli.Reset)

	fmt.Printf("    Recommended install methods:\n")
	fmt.Printf("      Ubuntu/Debian:  sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin\n")
	fmt.Printf("      Fedora:         sudo dnf install docker-ce docker-ce-cli containerd.io docker-compose-plugin\n")
	fmt.Printf("      Arch:           sudo pacman -S docker docker-compose\n")
	fmt.Printf("      Manual:         https://docs.docker.com/engine/install/\n\n")

	fmt.Printf("    Download and execute Docker's official installation script after SHA-256 verification? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	if strings.TrimSpace(strings.ToLower(input)) != "y" {
		return nil, fmt.Errorf("deployment aborted: Docker is required to isolate sensors")
	}

	fmt.Printf("\n    %s⚙️ Downloading and verifying Docker install script...%s\n", cli.Cyan, cli.Reset)

	result, err := FetchWithRemoteChecksum(DockerScriptURL)
	if err != nil {
		return nil, fmt.Errorf("download or verification failed: %w", err)
	}
	defer result.Cleanup()

	fmt.Printf("      ↳ Checksum verified. Executing install script...%s%s\n", cli.Dim, cli.Reset)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", result.Path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Docker installation failed: %w", err)
	}

	fmt.Printf("    %s✅ Docker installed.%s\n", cli.Green, cli.Reset)
	return []string{"docker", "compose"}, nil
}
