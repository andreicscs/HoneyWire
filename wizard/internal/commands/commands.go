package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/honeywire/wizard/internal/app"
	"github.com/honeywire/wizard/internal/cli"
)

const (
	defaultTimeout = 15 * time.Second
	composeTimeout = 30 * time.Second
)

func loadApp() (*app.App, error) {
	return app.LoadApp()
}

func warnIfHTTP(hubURL string, force bool) error {
	if strings.HasPrefix(hubURL, "http://") {
		fmt.Printf("\n%s⚠️  WARNING: Using HTTP (not HTTPS) for Hub communication!%s\n"+
			"This exposes you to MITM and DNS poisoning attacks.\n"+
			"Attackers could:\n"+
			"  1. Intercept DNS requests to redirect to fake Hub\n"+
			"  2. Inject malicious sensor images\n"+
			"  3. Gain root access to your system\n\n"+
			"Use HTTPS in production. HTTP is only for local testing.\n\n",
			cli.Yellow, cli.Reset,
		)
		if !cli.ConfirmAction("Continue with HTTP anyway", force) {
			return fmt.Errorf("aborted: HTTPS required for security")
		}
		fmt.Println()
	}
	return nil
}
