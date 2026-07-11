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

func HandleRelink(args []string, force bool) error {
	cli.PrintSectionHeader("HoneyWire Relink", cli.Yellow)

	var backupPath string
	hasBackup := false
	var defaultHubURL string

	if existing, err := app.LoadConfig(app.ConfigPath); err == nil {
		defaultHubURL = existing.HubURL
		fmt.Printf("    %sExisting node detected:%s\n", cli.Cyan, cli.Reset)
		fmt.Printf("      Node ID: %s\n", existing.NodeID)
		fmt.Printf("      Hub:     %s\n\n", existing.HubURL)
		fmt.Printf("    %sWarning: This will replace the existing node identity.%s\n", cli.Yellow, cli.Reset)
		fmt.Printf("    The old node entry remains in the Hub. Delete it from the Dashboard if needed.\n\n")

		if !cli.ConfirmAction("Continue?", force) {
			fmt.Printf("\n    %sRelink aborted.%s\n\n", cli.Dim, cli.Reset)
			return nil
		}

		backupPath = app.ConfigPath + ".bak"
		if data, err := os.ReadFile(app.ConfigPath); err == nil {
			if err := os.WriteFile(backupPath, data, 0600); err != nil {
				fmt.Printf("    %s[!] Warning: failed to backup current config: %v%s\n", cli.Yellow, err, cli.Reset)
			} else {
				hasBackup = true
			}
		}
	}

	err := executeRelink(args, defaultHubURL, force)

	if err != nil {
		if hasBackup {
			fmt.Printf("\n%s[!] Relink failed. Attempting rollback...%s\n", cli.Yellow, cli.Reset)
			if data, readErr := os.ReadFile(backupPath); readErr == nil {
				if writeErr := os.WriteFile(app.ConfigPath, data, 0600); writeErr != nil {
					fmt.Printf("%s[!] Rollback also failed: %v%s\n", cli.Red, writeErr, cli.Reset)
					fmt.Printf("    Manual recovery may be required at: %s\n", app.ConfigPath)
				} else {
					fmt.Printf("%s[*] Rolled back to previous configuration.%s\n", cli.Yellow, cli.Reset)
				}
			} else {
				fmt.Printf("%s[!] Rollback failed to read backup: %v%s\n", cli.Red, readErr, cli.Reset)
			}
		}
		return err
	}

	if hasBackup {
		os.Remove(backupPath)
	}

	return nil
}

func executeRelink(args []string, defaultHubURL string, force bool) error {
	var hubURL, apiKey string
	if len(args) > 0 {
		hubURL = args[0]
	}
	if len(args) > 1 {
		apiKey = args[1]
	}

	if !cli.IsTerminal() {
		if hubURL == "" || apiKey == "" {
			return fmt.Errorf("Hub URL and API key are required in non-interactive mode. Provide as arguments: honeywire relink <hub_url> <api_key>")
		}
	}

	if hubURL == "" {
		if defaultHubURL != "" {
			hubURL, _ = cli.PromptInput(fmt.Sprintf("    Hub URL [%s]: ", defaultHubURL))
			if strings.TrimSpace(hubURL) == "" {
				hubURL = defaultHubURL
			}
		} else {
			hubURL, _ = cli.PromptInput("    Hub URL: ")
		}
	}
	if hubURL == "" {
		return fmt.Errorf("Hub URL is required")
	}

	if err := warnIfHTTP(hubURL, force); err != nil {
		return err
	}

	if apiKey != "" {
		return linkExistingNode(hubURL, apiKey, force)
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
		alias, _ := cli.PromptInput("    Custom Alias (leave blank for hostname): ")
		tags, _ := cli.PromptInput("    Tags (comma-separated, leave blank for none): ")
		return provisionNewNode(hubURL, alias, tags, force)
	case "2":
		apiKey, err = cli.ReadPasswordMasked("    API Key: ")
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		if apiKey == "" {
			return fmt.Errorf("API key is required")
		}
		return linkExistingNode(hubURL, apiKey, force)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

func linkExistingNode(hubURL, apiKey string, force bool) error {
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

	if cli.IsTerminal() {
		if nodeInfo.PendingConfig {
			if cli.ConfirmAction("Apply Hub's desired state now", force) {
				applied, err := ApplyDesiredState(force)
				if err == nil && applied {
					if cli.ConfirmAction("Trigger a firedrill to test deployed sensors", force) {
						return HandleFiredrill()
					}
					fmt.Printf("\n    %sRun 'honeywire firedrill' when ready.%s\n\n", cli.Dim, cli.Reset)
				} else if err != nil {
					fmt.Printf("\n    %s[!] Apply failed, but the node remains linked. Run 'honeywire apply' to try again after fixing the Hub config.%s\n\n", cli.Yellow, cli.Reset)
				}
				return nil
			}
			fmt.Printf("    %sRun 'honeywire apply' to deploy this node's sensors.%s\n\n", cli.Dim, cli.Reset)
			return nil
		}
		if cli.ConfirmAction("Run host discovery now", force) {
			return HandleDiscover(force)
		}
	} else if nodeInfo.PendingConfig {
		fmt.Printf("    %sRun 'honeywire apply' to deploy this node's sensors.%s\n\n", cli.Dim, cli.Reset)
		return nil
	}

	fmt.Printf("    %sRun 'honeywire discover' to audit the host and add new sensors.%s\n\n", cli.Dim, cli.Reset)
	return nil
}

func provisionNewNode(hubURL, customAlias, tagsStr string, force bool) error {
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

	if cli.IsTerminal() {
		if cli.ConfirmAction("Run host discovery now", force) {
			return HandleDiscover(force)
		}
	}
	fmt.Printf("    %sRun 'honeywire discover' to audit the host and add new sensors.%s\n\n", cli.Dim, cli.Reset)
	return nil
}
