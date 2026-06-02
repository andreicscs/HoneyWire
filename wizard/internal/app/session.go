package app

import (
	"context"
	
	// codeql[go/insecure-randomness] Non-cryptographic use case.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand"
	"fmt"
	"os"
	"time"

	"github.com/honeywire/wizard/core/api"
	"github.com/honeywire/wizard/internal/cli"
	"golang.org/x/term"
)

type App struct {
	Config          *NodeConfig
	Hub             *api.HubClient
	dashboardCookie string
	rng             *rand.Rand
}

func NewApp(cfg *NodeConfig) *App {
	return &App{
		Config: cfg,
		Hub:    api.NewClient(cfg.HubURL),
		
		// codeql[go/insecure-randomness] Non-cryptographic use case.
		// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func LoadApp() (*App, error) {
	cfg, err := LoadConfig(ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("node not provisioned. Run 'wizard --link <hub_url>' first")
	}
	return NewApp(cfg), nil
}

func (a *App) HasDashboardAuth() bool {
	return a.dashboardCookie != ""
}

func (a *App) DashboardCookie() string {
	return a.dashboardCookie
}

func (a *App) Random() *rand.Rand {
	return a.rng
}

func (a *App) RequireDashboardAuth() error {
	if a.HasDashboardAuth() {
		return nil
	}

	if envPW := os.Getenv("HW_DASHBOARD_PASSWORD"); envPW != "" {
		authCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cookie, err := a.Hub.AuthenticateDashboard(authCtx, envPW)
		if err != nil {
			return fmt.Errorf("authentication via HW_DASHBOARD_PASSWORD failed: %w", err)
		}
		a.dashboardCookie = cookie
		return nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("dashboard authentication required but stdin is not a terminal. Set HW_DASHBOARD_PASSWORD environment variable or run interactively")
	}

	password, err := cli.ReadPasswordMasked("Enter dashboard password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Use a fresh context
	authCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cookie, err := a.Hub.AuthenticateDashboard(authCtx, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	a.dashboardCookie = cookie
	return nil
}
