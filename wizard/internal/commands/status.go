package commands

import (
	"context"
	"fmt"

	"github.com/honeywire/wizard/internal/cli"
)

func HandleStatus() error {
	app, err := loadApp()
	if err != nil {
		return err
	}

	cli.PrintSectionHeader("HoneyWire Node Status", cli.Cyan)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Explicitly fetch manifests to force the Hub to refresh its catalog cache.
	// This ensures the status command calculates the latest 'UpdateAvailable' flags accurately.
	_, _ = app.Hub.FetchManifests(ctx, app.Config.APIKey)

	nodeInfo, err := app.Hub.GetCurrentNode(ctx, app.Config.APIKey)
	if err != nil {
		return fmt.Errorf("failed to resolve node identity: %w", err)
	}

	cli.PrintNodeStatus(nodeInfo, app.Config.HubURL)
	return nil
}
