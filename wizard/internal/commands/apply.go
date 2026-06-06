package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/honeywire/wizard/internal/cli"
	"github.com/honeywire/wizard/internal/deploy"
	"github.com/honeywire/wizard/internal/system"
	"gopkg.in/yaml.v3"
)

func HandleApply() error {
	applied, err := ApplyDesiredState()
	if err == nil && applied {
		fmt.Printf("    %sRun 'honeywire firedrill' to test deployed sensors.%s\n\n", cli.Dim, cli.Reset)
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

	// Fetch the compose payload from the Hub. We intentionally do NOT rely on
	// the Hub's global Pending/Active revision tracker here — instead we will
	// perform a deep file diff against the local compose file and apply when
	// the files differ. This prevents short-circuiting when partial deploys
	// left the local state inconsistent with the Hub's wholesale payload.
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
		fmt.Printf("    %sUse 'honeywire discover' or the Hub Dashboard to add sensors first.%s\n\n", cli.Dim, cli.Reset)
		return false, nil
	}

	// Validate the Hub-provided compose before making changes.
	if err := system.ValidateComposeConfig(composeData); err != nil {
		return false, fmt.Errorf("compose validation failed: %w", err)
	}

	// Load local compose file (if present) and perform a deep YAML comparison
	// against the Hub payload. If they are identical, skip apply; otherwise
	// proceed to write/apply the Hub-provided compose file.
	composePath := filepath.Join(deploy.DeployDir, deploy.ComposeFile)
	var localData []byte
	if b, err := os.ReadFile(composePath); err == nil {
		localData = b
	} else if !os.IsNotExist(err) {
		// If we failed to read for a reason other than missing file, surface it.
		return false, fmt.Errorf("failed to read local compose file: %w", err)
	}

	var desiredObj map[string]interface{}
	var localObj map[string]interface{}
	if err := yaml.Unmarshal(composeData, &desiredObj); err != nil {
		return false, fmt.Errorf("failed to parse Hub compose payload: %w", err)
	}

	identical := false
	if len(localData) > 0 {
		if err := yaml.Unmarshal(localData, &localObj); err != nil {
			fmt.Printf("    %s[!] Warning: failed to parse local compose file; will apply fetched payload.%s\n", cli.Yellow, cli.Reset)
		} else {
			if reflect.DeepEqual(localObj, desiredObj) {
				identical = true
			}
		}
	}

	if identical {
		fmt.Printf("    %s✅ Node is already up to date with Hub's desired state.%s\n\n", cli.Green, cli.Reset)
		return false, nil
	}

	if err := deploy.Apply(ctx, composeData); err != nil {
		return false, fmt.Errorf("reconciliation failed: %w", err)
	}

	fmt.Printf("    %s✅ Node reconciled. Run 'docker compose -f %s -p %s ps' to view sensors.%s\n\n", cli.Green, filepath.Join(deploy.DeployDir, deploy.ComposeFile), deploy.ProjectName, cli.Reset)
	return true, nil
}
