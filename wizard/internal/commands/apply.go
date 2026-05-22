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
	app, err := loadApp()
	if err != nil {
		return err
	}

	cli.PrintSectionHeader("HoneyWire Apply", cli.Cyan)
	fmt.Printf("%s[*] Reconciling local node against Hub's desired state...%s\n", cli.Dim, cli.Reset)

	ctx, cancel := context.WithTimeout(context.Background(), composeTimeout)
	defer cancel()

	composeData, err := app.Hub.FetchCompose(ctx, app.Config.APIKey)
	if err != nil {
		return fmt.Errorf("failed to fetch deployment bundle: %w", err)
	}

	var compose struct {
		Services map[string]interface{} `yaml:"services"`
	}
	// If the compose file is empty or has no services, there's nothing to do.
	if err := yaml.Unmarshal(composeData, &compose); err != nil || len(compose.Services) == 0 {
		fmt.Printf("\n    %sNothing to reconcile. No sensors are configured for this node.%s\n", cli.Yellow, cli.Reset)
		fmt.Printf("    %sUse 'wizard discover' or the Hub Dashboard to add sensors first.%s\n\n", cli.Dim, cli.Reset)
		return nil
	}

	if err := deploy.Apply(ctx, composeData); err != nil {
		return fmt.Errorf("reconciliation failed: %w", err)
	}

	fmt.Printf("    %s✅ Node reconciled. Run 'docker compose -f %s -p %s ps' to view sensors.%s\n\n", cli.Green, filepath.Join(deploy.DeployDir, deploy.ComposeFile), deploy.ProjectName, cli.Reset)
	return nil
}
