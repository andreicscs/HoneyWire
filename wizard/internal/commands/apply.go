package commands

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/honeywire/wizard/internal/cli"
	"github.com/honeywire/wizard/internal/deploy"
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

	if err := deploy.Apply(ctx, composeData); err != nil {
		return fmt.Errorf("reconciliation failed: %w", err)
	}

	fmt.Printf("    %s✅ Node reconciled. Run 'docker compose -f %s -p %s ps' to view sensors.%s\n\n", cli.Green, filepath.Join(deploy.DeployDir, deploy.ComposeFile), deploy.ProjectName, cli.Reset)
	return nil
}
