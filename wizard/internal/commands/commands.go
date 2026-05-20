package commands

import (
	"time"

	"github.com/honeywire/wizard/internal/app"
)

const (
	defaultTimeout = 15 * time.Second
	composeTimeout = 30 * time.Second
)

func loadApp() (*app.App, error) {
	return app.LoadApp()
}
