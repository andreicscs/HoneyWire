package commands

import (
	"context"
	"fmt"

	"github.com/honeywire/wizard/internal/cli"
)

func HandleLink(hubURL, apiKey, alias, tags string, force bool) error {
	if apiKey != "" {
		if err := linkExistingNode(hubURL, apiKey); err != nil {
			return err
		}
	} else {
		if err := provisionNewNode(hubURL, alias, tags); err != nil {
			return err
		}
	}

	if !cli.IsTerminal() {
		fmt.Printf("\n    %sRun 'wizard discover' to audit the host.%s\n\n", cli.Dim, cli.Reset)
		return nil
	}

	if cli.ConfirmAction("Run host discovery now") {
		return HandleDiscover(force)
	}

	fmt.Printf("\n    %sRun 'wizard discover' when ready.%s\n\n", cli.Dim, cli.Reset)
	return nil
}

func HandleInteractiveMenu(force bool) error {
	app, err := loadApp()
	if err != nil {
		return cli.ShowOnboarding()
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	nodeInfo, err := app.Hub.GetCurrentNode(ctx, app.Config.APIKey)
	if err != nil {
		fmt.Printf("%s[!] Warning: Could not reach Hub: %v%s\n\n", cli.Yellow, err, cli.Reset)
	}

	fmt.Printf("\n%s%s=== HoneyWire Wizard v2.0 ===%s\n\n", cli.Bold, cli.Cyan, cli.Reset)

	if nodeInfo != nil {
		fmt.Printf("    %sExisting HoneyWire node detected%s\n\n", cli.Bold, cli.Reset)
		fmt.Printf("    Node:  %s\n", nodeInfo.Alias)
		fmt.Printf("    ID:    %s\n", nodeInfo.NodeID)
		fmt.Printf("    Hub:   %s\n\n", app.Config.HubURL)
	} else {
		fmt.Printf("    %sExisting HoneyWire node detected%s\n\n", cli.Bold, cli.Reset)
		fmt.Printf("    ID:    %s\n", app.Config.NodeID)
		fmt.Printf("    Hub:   %s\n\n", app.Config.HubURL)
	}

	fmt.Printf("    Choose action:\n\n")
	fmt.Printf("      %s[1]%s Reconcile node against Hub's desired state\n", cli.Cyan, cli.Reset)
	fmt.Printf("      %s[2]%s Run discovery & recommendations\n", cli.Cyan, cli.Reset)
	fmt.Printf("      %s[3]%s Show node status\n", cli.Cyan, cli.Reset)
	fmt.Printf("      %s[4]%s Re-link node\n", cli.Cyan, cli.Reset)
	fmt.Printf("      %s[5]%s Exit\n\n", cli.Red, cli.Reset)

	choice, err := cli.PromptInput("    Choice: ")
	if err != nil {
		return fmt.Errorf("failed to read choice: %w", err)
	}

	switch choice {
	case "1":
		return HandleApply()
	case "2":
		return HandleDiscover(force)
	case "3":
		return HandleStatus()
	case "4":
		return HandleRelink(nil)
	case "5":
		fmt.Printf("\n")
		return nil
	default:
		fmt.Printf("\n    %sInvalid choice.%s\n\n", cli.Red, cli.Reset)
		return nil
	}
}
