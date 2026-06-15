package cli

import (
	"fmt"
	"strings"

	"github.com/honeywire/wizard/core/api"
	"github.com/honeywire/wizard/core/discovery"
	"github.com/honeywire/wizard/core/scanner"
)

var hubInjectedVars = map[string]bool{
	"HW_HUB_ENDPOINT": true,
	"HW_HUB_KEY":      true,
	"HW_SENSOR_ID":    true,
	"HW_CONFIG_REV":   true,
	"HW_TEST_MODE":    true,
}

func PrintSectionHeader(title, color string) {
	fmt.Printf("\n%s%s=== %s ===%s\n\n", Bold, color, title, Reset)
}

func PrintDeploymentPlan(recommendations []*discovery.Recommendation) {
	fmt.Printf("    %sSuggested Sensors:%s\n\n", Gray, Reset)

	for _, rec := range recommendations {
		fmt.Printf("    %s+%s %s%s%s %s(%s)%s\n", Green, Reset, Bold, rec.SensorName, Reset, Gray, rec.SensorID, Reset)
		fmt.Printf("        %sReason:%s %s\n", Cyan, Reset, rec.Reason)
		imageStr := rec.DeploymentTemplate.ImageRepository
		if rec.DeploymentTemplate.ImageTag != "" {
			imageStr += ":" + rec.DeploymentTemplate.ImageTag
		} else {
			imageStr += ":latest"
		}
		if rec.DeploymentTemplate.ImageDigest != "" {
			imageStr += "@" + rec.DeploymentTemplate.ImageDigest
		}

		fmt.Printf("        %sImage:%s  %s\n", Cyan, Reset, imageStr)

		if len(rec.DeploymentTemplate.VolumeMounts) > 0 {
			fmt.Printf("        %sVolume Mounts:%s\n", Cyan, Reset)
			for _, vol := range rec.DeploymentTemplate.VolumeMounts {
				readonly := ""
				if vol.ReadOnly {
					readonly = ":ro"
				}
				fmt.Printf("          ~ %s:%s%s\n", vol.Source, vol.Target, readonly)
			}
		}

		if len(rec.DeploymentTemplate.EnvVars) > 0 {
			var hasVisibleEnv bool
			for _, env := range rec.DeploymentTemplate.EnvVars {
				if !env.Hidden && strings.TrimSpace(env.Default) != "" {
					hasVisibleEnv = true
					break
				}
			}

			if hasVisibleEnv {
				fmt.Printf("        %sConfig Variables:%s\n", Cyan, Reset)
				for _, env := range rec.DeploymentTemplate.EnvVars {
					if env.Hidden {
						continue
					}

					val := strings.TrimSpace(env.Default)
					if val != "" {
						suffix := ""
						if hubInjectedVars[env.Name] {
							suffix = " (injected by Hub)"
						}
						fmt.Printf("          ~ %s = %s%s\n", env.Name, val, suffix)
					}
				}
			}
		}
		fmt.Println()
	}
}

func PrintExpectedImpact(recs []*discovery.Recommendation) {
	fmt.Printf("      %sExpected Impact:%s\n", Bold, Reset)
	fmt.Printf("       - Memory: ~%dMB (low overhead)\n", len(recs)*15)
	fmt.Printf("       - CPU:    <1%% (event-driven logic)\n")
	fmt.Printf("       - Net:    Minimal (heartbeat only)\n\n")
}

func PrintCorrelatedServices(hostState *scanner.HostState, dockerMap map[int]string) {
	fmt.Printf("    %sDiscovered Attack Surface:%s\n", Gray, Reset)
	for _, svc := range hostState.Services {
		if svc.ProcessName == "docker-proxy" && dockerMap != nil {
			if imageName, exists := dockerMap[svc.Port]; exists {
				fmt.Printf("      %s[Container]%s %-6d ➔  %s%s%s (PID: %d)\n", Magenta, Reset, svc.Port, Cyan, imageName, Reset, svc.PID)
				continue
			}
		}
		fmt.Printf("      %s[Native]   %s %-6d ➔  %s%s%s (PID: %d)\n", Green, Reset, svc.Port, Yellow, svc.ProcessName, Reset, svc.PID)
	}
}

func PrintNodeStatus(nodeInfo *api.NodeInfo, hubURL string) {
	fmt.Printf("    %sNode:%s        %s\n", Cyan, Reset, nodeInfo.Alias)
	fmt.Printf("    %sNode ID:%s     %s\n", Cyan, Reset, nodeInfo.NodeID)
	fmt.Printf("    %sStatus:%s      %s\n", Cyan, Reset, nodeInfo.Status)
	fmt.Printf("    %sHub:%s         %s\n", Cyan, Reset, hubURL)

	fmt.Printf("    %sPending:%s     %v\n", Cyan, Reset, nodeInfo.PendingConfig)
	fmt.Printf("    %sActive Rev:%s  %s\n", Cyan, Reset, nodeInfo.ActiveRevision)
	fmt.Printf("    %sDesired Rev:%s %s\n", Cyan, Reset, nodeInfo.DesiredRevision)

	if nodeInfo.HasUpdateAvailable {
		fmt.Printf("    %sUpdates:%s     %s[AVAILABLE]%s\n", Cyan, Reset, Cyan, Reset)
	}

	syncState := "Synced"
	syncColor := Green
	if nodeInfo.PendingConfig {
		syncState = "Pending sync"
		syncColor = Yellow
	}
	fmt.Printf("    %sSync State:%s  %s%s%s\n", Cyan, Reset, syncColor, syncState, Reset)

	if len(nodeInfo.InstalledSensors) > 0 {
		fmt.Printf("\n    %sInstalled Sensors (%d):%s\n", Cyan, len(nodeInfo.InstalledSensors), Reset)
		for _, s := range nodeInfo.InstalledSensors {
			fmt.Printf("      • %s", s.SensorID)
			if s.CustomName != "" {
				fmt.Printf(" (%s)", s.CustomName)
			}
			if s.IsSilenced {
				fmt.Printf(" %s[silenced]%s", Yellow, Reset)
			}
			if s.UpdateAvailable {
				fmt.Printf(" %s[update available]%s", Cyan, Reset)
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("\n    %sNo sensors installed.%s\n", Dim, Reset)
	}

	fmt.Println()
}
