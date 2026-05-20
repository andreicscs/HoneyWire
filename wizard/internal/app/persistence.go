package app

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/honeywire/wizard/internal/util"
)

func SaveConfig(path string, cfg *NodeConfig) error {
	return util.AtomicWriteFile(path, cfg)
}

func LoadConfig(path string) (*NodeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg NodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.APIKey != "" {
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("config validation failed: %w", err)
		}
		return &cfg, nil
	}

	var legacy LegacyNodeConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("config file is neither new nor legacy format: %w", err)
	}

	apiKey := legacy.APIKey
	if apiKey == "" {
		apiKey = legacy.NodeKey
	}
	if apiKey == "" {
		return nil, fmt.Errorf("config file contains no api_key or node_key")
	}

	cfg = NodeConfig{
		HubURL: legacy.HubURL,
		NodeID: legacy.NodeID,
		APIKey: apiKey,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}
