package app

import (
	"fmt"
	"strings"
)

const ConfigPath = "/etc/honeywire/config.json"

// NodeConfig stores only the permanent machine identity.
// Dashboard auth is never persisted.
type NodeConfig struct {
	HubURL string `json:"hub_url"`
	NodeID string `json:"node_id"`
	APIKey string `json:"api_key"`
}

func (c *NodeConfig) Validate() error {
	if c.HubURL == "" {
		return fmt.Errorf("hub_url is empty")
	}
	if !strings.HasPrefix(c.HubURL, "http://") && !strings.HasPrefix(c.HubURL, "https://") {
		return fmt.Errorf("hub_url must start with http:// or https:// (got: %s)", c.HubURL)
	}
	if c.NodeID == "" {
		return fmt.Errorf("node_id is empty")
	}
	if c.APIKey == "" {
		return fmt.Errorf("api_key is empty")
	}
	return nil
}