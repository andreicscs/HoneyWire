package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/honeywire/wizard/internal/cli"
	"github.com/honeywire/wizard/internal/commands"
	"github.com/honeywire/wizard/internal/system"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s✖ Fatal Error: %v%s\n", cli.Red, err, cli.Reset)
		os.Exit(1)
	}
}

func run() error {
	cli.SetupUsage()

	uninstallPtr := flag.Bool("uninstall", false, "Tear down and remove all managed sensors from this node")
	forcePtr := flag.Bool("force", false, "Bypass confirmation prompts (useful for automation)")
	linkURL := flag.String("link", "", "Hub URL to link to (e.g., https://hub.honeywire.local)")
	apiKeyPtr := flag.String("api-key", "", "Node API key (for linking to an existing node)")
	aliasPtr := flag.String("alias", "", "Custom alias for this node (defaults to OS hostname)")
	tagsPtr := flag.String("tags", "", "Comma-separated tags for this node")

	flag.Parse()

	if warning, _ := system.CheckRoot(); warning != "" {
		return fmt.Errorf("Wizard must be run as root (sudo) for deep system access")
	}

	if *uninstallPtr {
		return commands.HandleTeardown(*forcePtr)
	}

	if *linkURL != "" {
		return commands.HandleLink(*linkURL, *apiKeyPtr, *aliasPtr, *tagsPtr, *forcePtr)
	}

	args := flag.Args()
	if len(args) == 0 {
		return commands.HandleInteractiveMenu(*forcePtr)
	}

	switch args[0] {
	case "apply":
		return commands.HandleApply()
	case "discover":
		return commands.HandleDiscover(*forcePtr)
	case "status":
		return commands.HandleStatus()
	case "relink":
		return commands.HandleRelink(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}
