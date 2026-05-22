package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/honeywire/wizard/core/api"
	"github.com/honeywire/wizard/internal/app"
	"github.com/honeywire/wizard/internal/cli"
)

func HandleRelink(args []string) error {
	cli.PrintSectionHeader("HoneyWire Relink", cli.Yellow)

	if existing, err := app.LoadConfig(app.ConfigPath); err == nil {
		fmt.Printf("    %sExisting node detected:%s\n", cli.Cyan, cli.Reset)
		fmt.Printf("      Node ID: %s\n", existing.NodeID)
		fmt.Printf("      Hub:     %s\n\n", existing.HubURL)
		fmt.Printf("    %sWarning: This will replace the existing node identity.%s\n", cli.Yellow, cli.Reset)
		fmt.Printf("    The old node entry remains in the Hub. Delete it from the Dashboard if needed.\n\n")

		if !cli.ConfirmAction("Continue?") {
			fmt.Printf("\n    %sRelink aborted.%s\n\n", cli.Dim, cli.Reset)
			return nil
		}

		if err := os.Remove(app.ConfigPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing config: %w", err)
		}
	}

	var hubURL, apiKey string
	if len(args) > 0 {
		hubURL = args[0]
	}
	if len(args) > 1 {
		apiKey = args[1]
	}

	if hubURL == "" {
		if !cli.IsTerminal() {
			return fmt.Errorf("Hub URL required but stdin is not a terminal. Provide as argument: wizard relink <hub_url>")
		}
		hubURL, _ = cli.PromptInput("    Hub URL: ")
	}
	if hubURL == "" {
		return fmt.Errorf("Hub URL is required")
	}

	if apiKey != "" {
		return linkExistingNode(hubURL, apiKey)
	}

	if !cli.IsTerminal() {
		return fmt.Errorf("API key required but stdin is not a terminal. Provide as argument: wizard relink <hub_url> <api_key>")
	}

	fmt.Printf("\n    Link method:\n")
	fmt.Printf("      [1] Create new node (requires dashboard password)\n")
	fmt.Printf("      [2] Link to existing node (requires API key)\n\n")
	choice, err := cli.PromptInput("    Choose: ")
	if err != nil {
		return fmt.Errorf("failed to read choice: %w", err)
	}

	switch strings.TrimSpace(choice) {
	case "1":
		return provisionNewNode(hubURL, "", "")
	case "2":
		apiKey, err = cli.ReadPasswordMasked("    API Key: ")
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		if apiKey == "" {
			return fmt.Errorf("API key is required")
		}
		return linkExistingNode(hubURL, apiKey)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

func linkExistingNode(hubURL, apiKey string) error {
	cli.PrintSectionHeader("HoneyWire Node Link", cli.Cyan)
	fmt.Printf("%s[*] Verifying API key with Hub at %s...%s\n", cli.Bold, hubURL, cli.Reset)

	hub := api.NewClient(hubURL)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	nodeInfo, err := hub.GetCurrentNode(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("failed to resolve node identity: %w", err)
	}

	fmt.Printf("%s[*] Resolved node: %s (%s)%s\n", cli.Green, nodeInfo.Alias, nodeInfo.NodeID, cli.Reset)

	cfg := app.NodeConfig{
		HubURL: hubURL,
		NodeID: nodeInfo.NodeID,
		APIKey: apiKey,
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("node config validation failed: %w", err)
	}

	if err := app.SaveConfig(app.ConfigPath, &cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	fmt.Printf("\n%s✅ Linked to Hub as '%s'.%s\n", cli.Green, nodeInfo.Alias, cli.Reset)
	fmt.Printf("    %sRun 'wizard apply' to deploy this node's sensors.%s\n\n", cli.Dim, cli.Reset)
	return nil
}

func provisionNewNode(hubURL, customAlias, tagsStr string) error {
	cli.PrintSectionHeader("HoneyWire Provisioning", cli.Cyan)
	fmt.Printf("%s[*] Connecting to Hub at %s...%s\n", cli.Bold, hubURL, cli.Reset)

	hub := api.NewClient(hubURL)

	password, err := cli.ResolveDashboardPassword()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cookie, err := hub.AuthenticateDashboard(ctx, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Printf("%s[*] Dashboard authentication successful.%s\n", cli.Green, cli.Reset)

	if customAlias == "" {
		customAlias, _ = os.Hostname()
		if customAlias == "" {
			customAlias = "unknown-node"
		}
	}

	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		for i, t := range tags {
			tags[i] = strings.TrimSpace(t)
		}
	}

	apiKey, err := hub.CreateNode(ctx, customAlias, tags, cookie)
	if err != nil {
		return fmt.Errorf("node creation failed: %w", err)
	}

	// The Hub's CreateNode API response currently only returns the API Key and Alias.
	// We must use the new API Key to fetch the authoritative node identity (including its NodeID).
	nodeInfo, err := hub.GetCurrentNode(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve node ID after creation: %w", err)
	}
	nodeID := nodeInfo.NodeID

	fmt.Printf("%s[*] Node created: %s (ID: %s)%s\n", cli.Green, customAlias, nodeID, cli.Reset)

	cfg := app.NodeConfig{
		HubURL: hubURL,
		NodeID: nodeID,
		APIKey: apiKey,
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("node config validation failed after creation: %w", err)
	}

	if err := app.SaveConfig(app.ConfigPath, &cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	fmt.Printf("\n%s✅ Node provisioned and identity saved.%s\n", cli.Green, cli.Reset)
	fmt.Printf("    %sRun 'wizard apply' to deploy existing sensors, or 'wizard discover' to add new ones.%s\n\n", cli.Dim, cli.Reset)
	return nil
}
