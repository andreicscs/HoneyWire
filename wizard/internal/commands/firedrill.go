package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/honeywire/wizard/internal/cli"
	"github.com/honeywire/wizard/internal/deploy"
)

func HandleFiredrill() error {
	appInstance, err := loadApp()
	if err != nil {
		return err
	}

	cli.PrintSectionHeader("HoneyWire Firedrill", cli.Magenta)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	installed, err := appInstance.Hub.FetchInstalledSensors(ctx, appInstance.Config.NodeID, appInstance.Config.APIKey)
	cancel() // Release context immediately after network call

	// Adapt the snippet logic: For a firedrill, we must know what's installed, so we fail gracefully.
	if err != nil {
		return fmt.Errorf("could not fetch installed sensors from Hub: %w", err)
	}

	if len(installed) == 0 {
		fmt.Printf("    %sNo sensors installed on this node. Nothing to test.%s\n\n", cli.Yellow, cli.Reset)
		return nil
	}

	fmt.Printf("%s[*] Initiating live test sequence on installed sensors...%s\n", cli.Dim, cli.Reset)

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to initialize Docker client: %w", err)
	}
	defer dockerClient.Close()

	// Add strict filters to ensure we only query HoneyWire deployments (-p honeywire)
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("com.docker.compose.project=%s", deploy.ProjectName))

	listCtx, listCancel := context.WithTimeout(context.Background(), 10*time.Second)
	containers, err := dockerClient.ContainerList(listCtx, container.ListOptions{
		Filters: f,
	})
	listCancel()
	if err != nil {
		return fmt.Errorf("failed to list docker containers: %w", err)
	}

	foundSensors := 0
	for _, c := range containers {
		project := c.Labels["com.docker.compose.project"]
		service := c.Labels["com.docker.compose.service"]
		configFile := c.Labels["com.docker.compose.project.config_files"]

		isHoneyWireProject := project == deploy.ProjectName
		isHoneyWireConfig := strings.Contains(configFile, deploy.ComposeFile)

		// Strictly verify the project matches (-p honeywire), the compose file matches honeywire-compose.yml,
		// and the service is actively registered in the Hub.
		if isHoneyWireProject && isHoneyWireConfig && installed[service] {
			killCtx, killCancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := dockerClient.ContainerKill(killCtx, c.ID, "SIGUSR1")
			killCancel()

			if err != nil {
				fmt.Printf("    %s[!] Failed to trigger %s: %v%s\n", cli.Red, service, err, cli.Reset)
			} else {
				fmt.Printf("    %s↳ Triggered test for: %s%s\n", cli.Green, service, cli.Reset)
				foundSensors++
			}
		}
	}

	if foundSensors == 0 {
		fmt.Printf("\n    %s[!] No running containers found for the installed sensors.%s\n", cli.Yellow, cli.Reset)
		fmt.Printf("    %sRun 'honeywire apply' to ensure deployments are up to date.%s\n\n", cli.Dim, cli.Reset)
	} else {
		fmt.Printf("\n    %s✅ Firedrill complete. Check your Hub dashboard for the test events!%s\n\n", cli.Green, cli.Reset)
	}

	return nil
}
