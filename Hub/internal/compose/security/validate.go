package security

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/honeywire/hub/internal/models"
)

var allowedCaps = map[string]bool{
	"NET_RAW":          true,
	"NET_BIND_SERVICE": true,
	"NET_ADMIN":        true,
	"DAC_OVERRIDE":     true,
}

var forbiddenMountPrefixes = []string{
	"/var/run/docker.sock",
	"/proc",
	"/sys",
	"/etc",
	"/root",
}

func containsInterpolation(s string) bool {
	return strings.Contains(s, "$") || strings.Contains(s, "{{")
}

func ValidateManifest(m models.SensorManifest) error {
	// Privilege Check
	for _, cap := range m.Deployment.CapAdd {
		if !allowedCaps[cap] {
			return fmt.Errorf("SECURITY REJECT: capability not allowed in v2: %s", cap)
		}
	}
	for _, ic := range m.Deployment.InitContainers {
		for _, cap := range ic.CapAdd {
			if !allowedCaps[cap] {
				return fmt.Errorf("SECURITY REJECT: capability not allowed in init container in v2: %s", cap)
			}
		}
	}

	// Host Escape Checks
	if err := checkVolumeMounts(m.Deployment.VolumeMounts); err != nil {
		return err
	}

	for _, initContainer := range m.Deployment.InitContainers {
		if err := checkVolumeMounts(initContainer.VolumeMounts); err != nil {
			return err
		}
	}

	// Interpolation Checks
	if containsInterpolation(m.Deployment.ImageRepository) || containsInterpolation(m.Deployment.ImageTag) || containsInterpolation(m.Deployment.ImageDigest) {
		return fmt.Errorf("SECURITY REJECT: interpolation not allowed in image fields")
	}
	if containsInterpolation(m.Deployment.NetworkMode) {
		return fmt.Errorf("SECURITY REJECT: interpolation not allowed in network_mode")
	}
	for _, vol := range m.Deployment.VolumeMounts {
		if containsInterpolation(vol.Source) || containsInterpolation(vol.Target) {
			return fmt.Errorf("SECURITY REJECT: interpolation not allowed in volume mounts")
		}
	}
	for _, ic := range m.Deployment.InitContainers {
		if containsInterpolation(ic.ImageRepository) || containsInterpolation(ic.ImageTag) || containsInterpolation(ic.ImageDigest) {
			return fmt.Errorf("SECURITY REJECT: interpolation not allowed in init container image fields")
		}
		if containsInterpolation(ic.Command) {
			return fmt.Errorf("SECURITY REJECT: interpolation not allowed in init container command")
		}
		for _, vol := range ic.VolumeMounts {
			if containsInterpolation(vol.Source) || containsInterpolation(vol.Target) {
				return fmt.Errorf("SECURITY REJECT: interpolation not allowed in init container volume mounts")
			}
		}
	}

	return nil
}

func ValidateMountPath(path string) error {
	cleanSource := filepath.Clean(path)
	for _, prefix := range forbiddenMountPrefixes {
		if cleanSource == prefix || strings.HasPrefix(cleanSource, prefix+"/") {
			return fmt.Errorf("SECURITY REJECT: mount path is forbidden: %s", path)
		}
	}
	return nil
}

func checkVolumeMounts(mounts []models.VolumeMount) error {
	for _, vol := range mounts {
		if err := ValidateMountPath(vol.Source); err != nil {
			return err
		}
	}
	return nil
}
