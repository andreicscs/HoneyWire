package security

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/honeywire/hub/internal/models"
)

func DecodeManifestStrict(r io.Reader) (models.SensorManifest, error) {
	var manifest models.SensorManifest
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("strict decode failed: %w", err)
	}



	return manifest, nil
}
