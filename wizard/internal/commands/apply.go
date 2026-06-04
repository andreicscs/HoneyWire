package commands

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/honeywire/wizard/internal/cli"
	"github.com/honeywire/wizard/internal/deploy"
	"gopkg.in/yaml.v3"
)

func HandleApply() error {
	applied, err := ApplyDesiredState()
	if err == nil && applied {
		fmt.Printf("    %sRun 'wizard firedrill' to test deployed sensors.%s\n\n", cli.Dim, cli.Reset)
	}
	return err
}

// ApplyDesiredState reconciles the node against the Hub's desired state
// and returns true if sensors were actually deployed.
func ApplyDesiredState() (bool, error) {
	app, err := loadApp()
	if err != nil {
		return false, err
	}

	cli.PrintSectionHeader("HoneyWire Apply", cli.Cyan)
	fmt.Printf("%s[*] Reconciling local node against Hub's desired state...%s\n", cli.Dim, cli.Reset)

	ctx, cancel := context.WithTimeout(context.Background(), composeTimeout)
	defer cancel()

	nodeInfo, err := app.Hub.GetCurrentNode(ctx, app.Config.APIKey)
	if err != nil {
		return false, fmt.Errorf("failed to check node status: %w", err)
	}

	if !nodeInfo.PendingConfig {
		fmt.Printf("    %s✅ Node is already up to date with Hub's desired state.%s\n\n", cli.Green, cli.Reset)
		return false, nil
	}

	composeData, err := app.Hub.FetchCompose(ctx, app.Config.APIKey)
	if err != nil {
		return false, fmt.Errorf("failed to fetch deployment bundle: %w", err)
	}

	var compose struct {
		Services map[string]interface{} `yaml:"services"`
	}
	// If the compose file is empty or has no services, there's nothing to do.
	if err := yaml.Unmarshal(composeData, &compose); err != nil || len(compose.Services) == 0 {
		fmt.Printf("\n    %sNothing to reconcile. No sensors are configured for this node.%s\n", cli.Yellow, cli.Reset)
		fmt.Printf("    %sUse 'wizard discover' or the Hub Dashboard to add sensors first.%s\n\n", cli.Dim, cli.Reset)
		return false, nil
	}

	if err := deploy.Apply(ctx, composeData); err != nil {
		return false, fmt.Errorf("reconciliation failed: %w", err)
	}

	fmt.Printf("    %s✅ Node reconciled. Run 'docker compose -f %s -p %s ps' to view sensors.%s\n\n", cli.Green, filepath.Join(deploy.DeployDir, deploy.ComposeFile), deploy.ProjectName, cli.Reset)
	return true, nil
}
