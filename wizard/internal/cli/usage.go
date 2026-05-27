package cli

import (
	"flag"
	"fmt"
)

func SetupUsage() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "\n%sHoneyWire Wizard — Node Management Client%s\n\n", Bold, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "Commands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %sapply%s      Reconcile local node against Hub's desired state\n", Cyan, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "  %sdiscover%s   Run host audit and produce sensor recommendations\n", Cyan, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "  %sstatus%s     Show linked node, sync state, and installed sensors\n", Cyan, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "  %srelink%s     Replace local node identity with a new node\n", Cyan, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "\nBootstrap:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s--link <hub>%s                  Create new node and link to Hub\n", Cyan, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s--link <hub> --api-key <key>%s  Link to existing node\n", Cyan, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s--alias <name>%s                Custom alias for this node (defaults to OS hostname)\n", Cyan, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s--tags <t1,t2>%s                Comma-separated tags for this node\n", Cyan, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "\nOther:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s--uninstall%s  Tear down sensors and remove node identity\n", Red, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s--force%s      Bypass confirmation prompts (useful for automation)\n", Cyan, Reset)
		fmt.Fprintf(flag.CommandLine.Output(), "\nEnvironment:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  HW_DASHBOARD_PASSWORD  Dashboard password (non-interactive use)\n\n")
	}
}
