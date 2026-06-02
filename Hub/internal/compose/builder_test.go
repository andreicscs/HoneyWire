package compose

import (
	"testing"

	"github.com/honeywire/hub/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildService_SecurityDefaults(t *testing.T) {
	manifest := models.SensorManifest{
		Deployment: models.Deployment{
			Image: "test-image",
		},
	}
	compose, err := BuildService("test-sensor", manifest, map[string]string{})
	require.NoError(t, err)
	require.Len(t, compose.Services, 1)

	svc := compose.Services[0].Service
	assert.Equal(t, "1000:1000", svc.User, "should default to non-root user")
	assert.Equal(t, []string{"ALL"}, svc.CapDrop, "should drop all capabilities by default")
	assert.True(t, svc.ReadOnly, "should be readonly by default")
	assert.Contains(t, svc.SecurityOpt, "no-new-privileges:true", "should prevent new privileges")
}

func TestBuildService_CapabilityFiltering(t *testing.T) {
	manifest := models.SensorManifest{
		Deployment: models.Deployment{
			Image:  "test-image",
			CapAdd: []string{"NET_RAW", "SYS_ADMIN"}, // SYS_ADMIN is not allowed
		},
	}
	compose, err := BuildService("test-sensor", manifest, map[string]string{})
	require.NoError(t, err)
	require.Len(t, compose.Services, 1)

	svc := compose.Services[0].Service
	assert.Equal(t, []string{"NET_RAW"}, svc.CapAdd, "should filter out disallowed capabilities")
}

func TestBuildService_DynamicVolumeExpansion(t *testing.T) {
	t.Run("Dynamic File Bind", func(t *testing.T) {
		manifest := models.SensorManifest{
			Deployment: models.Deployment{
				Image: "test-image",
				VolumeMounts: []models.VolumeMount{
					{Type: models.DynamicFileBind, SourceEnv: "HW_FILES", TargetPrefix: "/watch/"},
				},
			},
		}
		envMap := map[string]string{"HW_FILES": "/opt/z.txt, /opt/a.txt, , /opt/m.txt"}

		compose, err := BuildService("test-sensor", manifest, envMap)
		require.NoError(t, err)
		svc := compose.Services[0].Service

		require.Len(t, svc.Volumes, 3)
		assert.Equal(t, "/watch//opt/a.txt", svc.Volumes[0].Target)
		assert.Equal(t, "/opt/a.txt", svc.Volumes[0].Source)
		assert.Equal(t, "/watch//opt/m.txt", svc.Volumes[1].Target)
		assert.Equal(t, "/watch//opt/z.txt", svc.Volumes[2].Target)
	})

	t.Run("Dynamic Dir Bind", func(t *testing.T) {
		manifest := models.SensorManifest{
			Deployment: models.Deployment{
				InitContainers: []models.InitContainer{
					{Name: "init", VolumeMounts: []models.VolumeMount{
						{Type: models.DynamicDirBind, SourceEnv: "HW_FILES", TargetPrefix: "/watch/"},
					}},
				},
			},
		}
		envMap := map[string]string{"HW_FILES": "/opt/z/file.txt, /opt/a/file.txt, /opt/a/another.txt"}

		compose, err := BuildService("test-sensor", manifest, envMap)
		require.NoError(t, err)
		initSvc := compose.Services[0].Service

		require.Len(t, initSvc.Volumes, 2)
		assert.Equal(t, "/watch//opt/a", initSvc.Volumes[0].Target)
		assert.Equal(t, "/opt/a", initSvc.Volumes[0].Source)
		assert.Equal(t, "/watch//opt/z", initSvc.Volumes[1].Target)
	})

	t.Run("Dynamic Mount Security Block", func(t *testing.T) {
		manifest := models.SensorManifest{
			Deployment: models.Deployment{
				VolumeMounts: []models.VolumeMount{
					{Type: models.DynamicFileBind, SourceEnv: "HW_FILES", TargetPrefix: "/watch/"},
				},
			},
		}
		_, err := BuildService("test-sensor", manifest, map[string]string{"HW_FILES": "/etc/passwd"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mount path is forbidden: /etc/passwd")
	})
}

func TestBuildService_InitContainers(t *testing.T) {
	manifest := models.SensorManifest{
		Deployment: models.Deployment{
			Image: "main-image",
			InitContainers: []models.InitContainer{
				{Name: "z-init", Image: "init-z"},
				{Name: "a-init", Image: "init-a"},
			},
		},
	}
	envMap := map[string]string{
		"HW_SENSOR_ID": "sensor123",
		"HW_HUB_KEY":   "key123",
		"CUSTOM_VAR":   "value123",
	}

	compose, err := BuildService("test-sensor", manifest, envMap)
	require.NoError(t, err)
	require.Len(t, compose.Services, 3) // 2 init + 1 main

	assert.Equal(t, "a-init", compose.Services[0].Name, "init containers should be sorted by name")
	assert.Equal(t, "z-init", compose.Services[1].Name)

	initSvc := compose.Services[0].Service
	assert.Equal(t, []string{"CUSTOM_VAR=value123"}, initSvc.Environment, "init containers should not get core env vars")

	mainSvc := compose.Services[2].Service
	require.Len(t, mainSvc.DependsOn, 2)
	assert.Equal(t, "service_completed_successfully", mainSvc.DependsOn["a-init"].Condition)
}

func TestSortEnvKeys(t *testing.T) {
	keys := []string{"FOO", "HW_HUB_KEY", "BAR", "HW_SENSOR_ID", "HW_SEVERITY"}
	sortEnvKeys(keys)
	expected := []string{"HW_SENSOR_ID", "HW_HUB_KEY", "HW_SEVERITY", "BAR", "FOO"}
	assert.Equal(t, expected, keys)
}