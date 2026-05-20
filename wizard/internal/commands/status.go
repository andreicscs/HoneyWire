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

	nodeInfo, err := app.Hub.GetCurrentNode(ctx, app.Config.APIKey)
	if err != nil {
		return fmt.Errorf("failed to resolve node identity: %w", err)
	}

	cli.PrintNodeStatus(nodeInfo, app.Config.HubURL)
	return nil
}
