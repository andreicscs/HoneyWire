package security

import (
	"strings"
	"testing"

	"github.com/honeywire/hub/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestValidateMountPath(t *testing.T) {
	testCases := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{"Forbidden Docker Socket", "/var/run/docker.sock", true},
		{"Forbidden Proc", "/proc", true},
		{"Forbidden Sys", "/sys", true},
		{"Forbidden Etc", "/etc", true},
		{"Forbidden Root", "/root", true},
		{"Forbidden Proc Subpath", "/proc/1/root", true},
		{"Forbidden Etc Subpath", "/etc/passwd", true},
		{"Forbidden Root Subpath", "/root/.ssh", true},
		{"Path Traversal to Docker Sock", "/opt/data/../../var/run/docker.sock", true},
		{"Path Traversal to Etc", "/home/user/../../etc/passwd", true},
		{"Allowed Path", "/opt/data", false},
		{"Allowed Home Path", "/home/user/logs", false},
		{"Allowed Relative Path", "relative/path", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMountPath(tc.path)
			if tc.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateManifest(t *testing.T) {
	baseManifest := func() models.SensorManifest {
		return models.SensorManifest{
			Deployment: models.Deployment{
				ImageRepository: "safe-image:latest",
			},
		}
	}

	testCases := []struct {
		name        string
		modifier    func(*models.SensorManifest)
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid Manifest",
			modifier:    func(m *models.SensorManifest) {},
			expectError: false,
		},
		{
			name: "Disallowed Capability in Main",
			modifier: func(m *models.SensorManifest) {
				m.Deployment.CapAdd = []string{"SYS_ADMIN"}
			},
			expectError: true,
			errorMsg:    "capability not allowed in v2: SYS_ADMIN",
		},
		{
			name: "Allowed Capability in Main",
			modifier: func(m *models.SensorManifest) {
				m.Deployment.CapAdd = []string{"NET_RAW"}
			},
			expectError: false,
		},
		{
			name: "Disallowed Capability in Init",
			modifier: func(m *models.SensorManifest) {
				m.Deployment.InitContainers = []models.InitContainer{
					{Name: "bad-init", CapAdd: []string{"SYS_ADMIN"}},
				}
			},
			expectError: true,
			errorMsg:    "capability not allowed in init container in v2: SYS_ADMIN",
		},
		{
			name: "Forbidden Mount in Main",
			modifier: func(m *models.SensorManifest) {
				m.Deployment.VolumeMounts = []models.VolumeMount{
					{Source: "/etc/passwd", Target: "/etc/passwd"},
				}
			},
			expectError: true,
			errorMsg:    "mount path is forbidden: /etc/passwd",
		},
		{
			name: "Interpolation in Main Image",
			modifier: func(m *models.SensorManifest) {
				m.Deployment.ImageRepository = "image:${TAG}"
			},
			expectError: true,
			errorMsg:    "interpolation not allowed in image",
		},
		{
			name: "Interpolation in Main NetworkMode",
			modifier: func(m *models.SensorManifest) {
				m.Deployment.NetworkMode = "host-{{mode}}"
			},
			expectError: true,
			errorMsg:    "interpolation not allowed in network_mode",
		},
		{
			name: "Interpolation in Main Volume Source",
			modifier: func(m *models.SensorManifest) {
				m.Deployment.VolumeMounts = []models.VolumeMount{
					{Source: "${HOST_PATH}", Target: "/data"},
				}
			},
			expectError: true,
			errorMsg:    "interpolation not allowed in volume mounts",
		},
		{
			name: "Interpolation in Init Command",
			modifier: func(m *models.SensorManifest) {
				m.Deployment.InitContainers = []models.InitContainer{
					{Name: "bad-init", Command: "echo ${SECRET}"},
				}
			},
			expectError: true,
			errorMsg:    "interpolation not allowed in init container command",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manifest := baseManifest()
			tc.modifier(&manifest)
			err := ValidateManifest(manifest)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDecodeManifestStrict(t *testing.T) {
	t.Run("Disallow Unknown Fields", func(t *testing.T) {
		badJson := `{ "deployment": {}, "unknown_field": "should_fail"}`
		_, err := DecodeManifestStrict(strings.NewReader(badJson))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_field")
	})

	t.Run("Valid Decode", func(t *testing.T) {
		goodJson := `{ "deployment": {"image_repository": "test"}}`
		_, err := DecodeManifestStrict(strings.NewReader(goodJson))
		assert.NoError(t, err)
	})
}
