package cli

import (
	"flag"
	"fmt"
)

func SetupUsage() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()

		// Helper function centralizes the security suppressions to a single place
		printUsage := func(format string, a ...any) {
			// nosemgrep: go.lang.security.audit.xss.no-fprintf-to-responsewriter.no-fprintf-to-responsewriter
			// codeql[go/xss] Printing to CLI stdout/stderr, not HTTP.
			fmt.Fprintf(out, format, a...)
		}

		printUsage("\n%sHoneyWire Wizard — Node Management Client%s\n\n", Bold, Reset)
		printUsage("Commands:\n")
		printUsage("  %sapply%s      Reconcile local node against Hub's desired state\n", Cyan, Reset)
		printUsage("  %sdiscover%s   Run host audit and produce sensor recommendations\n", Cyan, Reset)
		printUsage("  %sstatus%s     Show linked node, sync state, and installed sensors\n", Cyan, Reset)
		printUsage("  %srelink%s     Replace local node identity with a new node\n", Cyan, Reset)
		
		printUsage("\nBootstrap:\n")
		printUsage("  %s--link <hub>%s                  Create new node and link to Hub\n", Cyan, Reset)
		printUsage("  %s--link <hub> --api-key <key>%s  Link to existing node\n", Cyan, Reset)
		printUsage("  %s--alias <name>%s                Custom alias for this node (defaults to OS hostname)\n", Cyan, Reset)
		printUsage("  %s--tags <t1,t2>%s                Comma-separated tags for this node\n", Cyan, Reset)
		
		printUsage("\nOther:\n")
		printUsage("  %s--uninstall%s  Tear down sensors and remove node identity\n", Red, Reset)
		printUsage("  %s--force%s      Bypass confirmation prompts (useful for automation)\n", Cyan, Reset)
		
		printUsage("\nEnvironment:\n")
		printUsage("  HW_DASHBOARD_PASSWORD  Dashboard password (non-interactive use)\n\n")
	}
}