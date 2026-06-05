package commands

import (
	"context"
	"fmt"

	"github.com/honeywire/wizard/internal/cli"
)

func HandleLink(hubURL, apiKey, alias, tags string, force bool) error {
	if err := warnIfHTTP(hubURL, force); err != nil {
		return err
	}

	if apiKey != "" {
		if err := linkExistingNode(hubURL, apiKey, force); err != nil {
			return err
		}
	} else {
		if appInstance, err := loadApp(); err == nil {
			fmt.Printf("\n    %s[*] Dashboard auth required to provision a new node.%s\n", cli.Cyan, cli.Reset)
			if authErr := appInstance.RequireDashboardAuth(); authErr != nil {
				return fmt.Errorf("dashboard authentication required: %w", authErr)
			}
		}
		if err := provisionNewNode(hubURL, alias, tags, force); err != nil {
			return err
		}
	}

	return nil
}

func HandleInteractiveMenu(force bool) error {
	app, err := loadApp()
	if err != nil {
		return cli.ShowOnboarding()
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	nodeInfo, err := app.Hub.GetCurrentNode(ctx, app.Config.APIKey)
	cancel() // Release context immediately after network call

	if err != nil {
		fmt.Printf("%s[!] Warning: Could not reach Hub: %v%s\n\n", cli.Yellow, err, cli.Reset)
	}

	fmt.Printf("\n%s%s=== HoneyWire Wizard ===%s\n\n", cli.Bold, cli.Cyan, cli.Reset)

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
	fmt.Printf("      %s[1]%s Apply Hub's state\n", cli.Cyan, cli.Reset)
	fmt.Printf("      %s[2]%s Run discovery & recommendations\n", cli.Cyan, cli.Reset)
	fmt.Printf("      %s[3]%s Show node status\n", cli.Cyan, cli.Reset)
	fmt.Printf("      %s[4]%s Trigger firedrill (live test)\n", cli.Cyan, cli.Reset)
	fmt.Printf("      %s[5]%s Re-link node\n", cli.Cyan, cli.Reset)
	fmt.Printf("      %s[6]%s Uninstall node\n", cli.Red, cli.Reset)
	fmt.Printf("      %s[7]%s Exit\n\n", cli.Dim, cli.Reset)

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
		return HandleFiredrill()
	case "5":
		return HandleRelink(nil, force)
	case "6":
		err := HandleTeardown(force)
		if err == nil {
			fmt.Printf("\n  Some HoneyWire sensors may have created host-side artifacts or decoy files.\n")
			fmt.Printf("    Review your deployment configuration and remove any remaining files manually if desired.\n\n")
		}
		return err
	case "7":
		fmt.Printf("\n")
		return nil
	default:
		fmt.Printf("\n    %sInvalid choice.%s\n\n", cli.Red, cli.Reset)
		return nil
	}
}
