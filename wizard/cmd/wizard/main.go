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

		// codeql[go/xss] Printing to CLI stdout/stderr, not HTTP.
		// nosemgrep: go.lang.security.audit.xss.no-fprintf-to-responsewriter.no-fprintf-to-responsewriter
		fmt.Fprintf(os.Stderr, "\n%sFatal Error: %v%s\n", cli.Red, err, cli.Reset)
		os.Exit(1)
	}
}

func run() error {
	cli.SetupUsage()

	forcePtr := flag.Bool("force", false, "Bypass confirmation prompts (useful for automation)")
	linkURL := flag.String("link", "", "Hub URL to link to (e.g., https://hub.honeywire.local)")
	apiKeyPtr := flag.String("api-key", "", "Node API key (for linking to an existing node)")
	aliasPtr := flag.String("alias", "", "Custom alias for this node (defaults to OS hostname)")
	tagsPtr := flag.String("tags", "", "Comma-separated tags for this node")

	flag.Parse()

	if warning, _ := system.CheckRoot(); warning != "" {
		return fmt.Errorf("HoneyWire wizard must be run as root (sudo) for system analysis and sensor deployment.")
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
	case "firedrill":
		return commands.HandleFiredrill()
	case "status":
		return commands.HandleStatus()
	case "relink":
		return commands.HandleRelink(args[1:], *forcePtr)
	case "uninstall":
		err := commands.HandleTeardown(*forcePtr)
		if err == nil {
			fmt.Printf("\n  Some HoneyWire sensors may have created host-side artifacts or decoy files.\n")
			fmt.Printf("    Review your deployment configuration and remove any remaining files manually if desired.\n\n")
		}
		return err
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}
