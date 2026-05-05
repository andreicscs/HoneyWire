package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/honeywire/wizard/pkg/schema"
)

const DefaultRegistryURL = "https://raw.githubusercontent.com/andreicscs/HoneyWire/main/Sensors/official/manifests.json"

func FetchManifests(source string) ([]*schema.SensorManifest, error) {
	// 1. Testing Logic: If source is a local file path
	if !strings.HasPrefix(source, "http") {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("local manifest error: %w", err)
		}
		var manifests []*schema.SensorManifest
		err = json.Unmarshal(data, &manifests)
		return manifests, err
	}

	// 2. Production Logic: Remote Fetch
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(source)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var manifests []*schema.SensorManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifests); err != nil {
		return nil, err
	}

	return manifests, nil
}