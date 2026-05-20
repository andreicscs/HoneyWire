package commands

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/honeywire/wizard/core/api"
	"github.com/honeywire/wizard/core/discovery"
	"github.com/honeywire/wizard/core/scanner"
	"github.com/honeywire/wizard/core/schema"
	"github.com/honeywire/wizard/internal/app"
	"github.com/honeywire/wizard/internal/cli"
	"github.com/honeywire/wizard/internal/deploy"
	"github.com/honeywire/wizard/internal/system"
)

var hubInjectedVars = map[string]bool{
	"HW_HUB_ENDPOINT": true,
	"HW_HUB_KEY":      true,
	"HW_SENSOR_ID":    true,
	"HW_CONFIG_REV":   true,
	"HW_TEST_MODE":    true,
}

func HandleDiscover(registry string, force bool) error {
	app, err := loadApp()
	if err != nil {
		return err
	}

	cli.PrintSectionHeader("HoneyWire Discover", cli.Cyan)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	installed, err := app.Hub.FetchInstalledSensors(ctx, app.Config.NodeID, app.Config.APIKey)
	if err != nil {
		fmt.Printf("%s[!] Warning: Could not fetch installed sensors from Hub: %v%s\n", cli.Yellow, err, cli.Reset)
		fmt.Printf("%s    Continuing without duplicate detection.%s\n\n", cli.Dim, cli.Reset)
		installed = make(map[string]bool)
	}

	if err := runPreFlightChecks(force); err != nil {
		return err
	}

	hostState, dockerMap, systemState, err := auditEnvironment()
	if err != nil {
		return err
	}

	recommendations, err := buildStrategy(hostState, systemState, dockerMap, registry, app.Config, app.Random())
	if err != nil {
		return err
	}

	recommendations = filterInstalledSensors(recommendations, installed)

	if len(recommendations) == 0 {
		fmt.Printf("    %sNo new recommendations. All applicable sensors are already deployed.%s\n\n", cli.Dim, cli.Reset)
		return nil
	}

	cli.PrintDeploymentPlan(recommendations)
	cli.PrintExpectedImpact(recommendations)

	if !cli.ConfirmAction(fmt.Sprintf("Apply these %d sensor suggestions", len(recommendations))) {
		fmt.Printf("\n    %sSuggestions not applied. Run 'wizard apply' when ready.%s\n\n", cli.Dim, cli.Reset)
		return nil
	}

	return applySuggestions(app, recommendations, dockerMap)
}

func applySuggestions(app *app.App, recs []*discovery.Recommendation, dockerMap map[int]string) error {
	fmt.Printf("\n    %s[*] Dashboard auth required to register sensors.%s\n", cli.Cyan, cli.Reset)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	if err := app.RequireDashboardAuth(ctx); err != nil {
		return fmt.Errorf("dashboard authentication required: %w", err)
	}

	fmt.Printf("    %s[*] Registering sensors with Hub...%s\n", cli.Cyan, cli.Reset)
	for _, rec := range recs {
		configValues := make(map[string]string)
		for _, env := range rec.DeploymentTemplate.EnvVars {
			if strings.HasPrefix(env.Name, "HW_") && !hubInjectedVars[env.Name] {
				configValues[env.Name] = env.Default
			}
		}

		if err := app.Hub.AddSensor(ctx, app.Config.NodeID, app.DashboardCookie(), rec.SensorID, rec.SensorName, configValues); err != nil {
			return fmt.Errorf("failed to register sensors with Hub: %w", err)
		}
		fmt.Printf("    %s↳ Registered: %s%s\n", cli.Green, rec.SensorID, cli.Reset)
	}

	fmt.Printf("    %s[*] Reconciling node against Hub's desired state...%s\n", cli.Cyan, cli.Reset)

	reconcileCtx, reconcileCancel := context.WithTimeout(context.Background(), composeTimeout)
	defer reconcileCancel()

	composeData, err := app.Hub.FetchCompose(reconcileCtx, app.Config.APIKey)
	if err != nil {
		return fmt.Errorf("failed to fetch deployment bundle: %w", err)
	}

	if err := deploy.Apply(reconcileCtx, composeData); err != nil {
		return fmt.Errorf("reconciliation failed: %w", err)
	}

	fmt.Printf("    %s✅ Node reconciled. Run 'docker compose -f %s -p %s ps' to view sensors.%s\n\n", cli.Green, filepath.Join(deploy.DeployDir, deploy.ComposeFile), deploy.ProjectName, cli.Reset)
	return nil
}

func runPreFlightChecks(force bool) error {
	var hasWarnings bool
	checks := []func() (string, error){system.CheckMemory, system.CheckLoad, system.CheckDiskSpace}
	for _, check := range checks {
		if warning, err := check(); err == nil && warning != "" {
			fmt.Printf("%s%s%s\n", cli.Yellow, warning, cli.Reset)
			hasWarnings = true
		}
	}

	if hasWarnings && !force {
		fmt.Printf("\n%s⚠️  Host environment is severely degraded. Proceeding may cause instability.%s\n", cli.Red, cli.Reset)
		if !cli.ConfirmAction("Continue anyway") {
			return fmt.Errorf("aborted due to failing health checks")
		}
		fmt.Println()
	}
	return nil
}

func auditEnvironment() (*scanner.HostState, map[int]string, *system.SystemState, error) {
	systemState, err := system.LoadCurrentState()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load system state: %w", err)
	}

	fmt.Printf("%s[*] Step 1/3: Analyzing Host OS & Sockets...%s\n", cli.Bold, cli.Reset)
	hostScanner := scanner.NewProcScanner()
	hostState, err := hostScanner.Scan(systemState)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to scan host: %w", err)
	}

	fmt.Printf("%s[*] Step 2/3: Interrogating Docker Daemon...%s\n", cli.Bold, cli.Reset)
	dockerMap, dockerErr := buildDockerPortMap()
	if dockerErr != nil {
		fmt.Printf("    %s↳ Warning: Cannot access Docker API (%v). Proceeding with native scanning.%s\n", cli.Yellow, dockerErr, cli.Reset)
	} else {
		fmt.Printf("    %s↳ Successfully mapped %d containerized ports.%s\n", cli.Green, len(dockerMap), cli.Reset)
	}

	fmt.Println()
	cli.PrintCorrelatedServices(hostState, dockerMap)
	return hostState, dockerMap, systemState, nil
}

func buildStrategy(hostState *scanner.HostState, systemState *system.SystemState, dockerMap map[int]string, registry string, nodeConfig *app.NodeConfig, rng *rand.Rand) ([]*discovery.Recommendation, error) {
	fmt.Printf("\n%s[*] Step 3/3: Formulating Deception Strategy...%s\n", cli.Bold, cli.Reset)

	manifests, apiErr := api.FetchManifests(registry)
	if apiErr != nil {
		return nil, fmt.Errorf("Registry error: %w", apiErr)
	}

	fmt.Printf("    %s↳ Synced %d active sensors from registry%s\n", cli.Green, len(manifests), cli.Reset)

	engine := discovery.NewEngine(manifests)
	recommendations := engine.GetRecommendations(hostState, systemState)

	renderManifestTemplates(recommendations, dockerMap, hostState, rng)

	for _, rec := range recommendations {
		updateEnvVar(rec, "HW_HUB_ENDPOINT", nodeConfig.HubURL)
		updateEnvVar(rec, "HW_SENSOR_ID", rec.SensorID)
	}

	return recommendations, nil
}

func filterInstalledSensors(recs []*discovery.Recommendation, installed map[string]bool) []*discovery.Recommendation {
	var filtered []*discovery.Recommendation
	for _, rec := range recs {
		if installed[rec.SensorID] {
			fmt.Printf("    %s⊘ Skipped: %s (already installed)%s\n", cli.Dim, rec.SensorID, cli.Reset)
			continue
		}
		filtered = append(filtered, rec)
	}
	return filtered
}

func renderManifestTemplates(recs []*discovery.Recommendation, dockerMap map[int]string, hostState *scanner.HostState, rng *rand.Rand) {
	var portStrs []string
	usedPorts := make(map[int]bool)
	for _, svc := range hostState.Services {
		portStrs = append(portStrs, fmt.Sprintf("%d", svc.Port))
		usedPorts[svc.Port] = true
	}
	ignorePorts := strings.Join(portStrs, ",")

	usedFiles := make(map[string]bool)

	safeFilePick := func(files []string, basePath string) string {
		rng.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
		for _, f := range files {
			if !usedFiles[f] {
				if _, err := os.Stat(filepath.Join(basePath, f)); os.IsNotExist(err) {
					usedFiles[f] = true
					return f
				}
			}
		}
		fallback := fmt.Sprintf("backup_hw_%d.bak", rng.Intn(99999))
		usedFiles[fallback] = true
		return fallback
	}

	funcMap := template.FuncMap{
		"randWebFile": func(basePath string) string {
			return safeFilePick([]string{"wp-config.old.php", ".env.staging", "config.bak.php", "aws_s3_keys.txt"}, basePath)
		},
		"randDBFile": func(basePath string) string {
			return safeFilePick([]string{"dump_2023.sql", "db_backup_prod.sql", ".pgpass.old", "migration_rollback.sql"}, basePath)
		},
		"availablePort": func(base int) string {
			p := base
			for {
				if !usedPorts[p] {
					addr := fmt.Sprintf(":%d", p)
					ln, err := net.Listen("tcp", addr)
					if err == nil {
						ln.Close()
						usedPorts[p] = true
						return fmt.Sprintf("%d", p)
					}
				}
				p++
			}
		},
	}

	for _, rec := range recs {
		var trapPath string
		hasWeb := false
		hasDB := false

		for _, svc := range rec.MatchedServices {
			name := strings.ToLower(svc.ProcessName)
			if name == "docker-proxy" && dockerMap != nil {
				if img, ok := dockerMap[svc.Port]; ok {
					name = strings.ToLower(img)
				}
			}
			if strings.Contains(name, "postgres") || strings.Contains(name, "mysql") || strings.Contains(name, "redis") {
				hasDB = true
			}
			if strings.Contains(name, "nginx") || strings.Contains(name, "apache") || strings.Contains(name, "httpd") {
				hasWeb = true
			}
		}

		if hasWeb {
			trapPath = "/var/www/html/.backups"
		} else if hasDB {
			trapPath = "/var/lib/db_backups"
		} else {
			trapPath = "/opt/app_data/backups"
		}

		ctx := struct {
			HasWeb          bool
			HasDB           bool
			IgnorePorts     string
			TrapPath        string
			MatchedServices []discovery.MatchedService
		}{
			HasWeb:          hasWeb,
			HasDB:           hasDB,
			IgnorePorts:     ignorePorts,
			TrapPath:        trapPath,
			MatchedServices: rec.MatchedServices,
		}

		for i, vol := range rec.DeploymentTemplate.VolumeMounts {
			t, err := template.New("vol").Funcs(funcMap).Parse(vol.Source)
			if err != nil {
				continue
			}
			var buf strings.Builder
			if err := t.Execute(&buf, ctx); err == nil {
				rec.DeploymentTemplate.VolumeMounts[i].Source = buf.String()
			}
		}

		for i, env := range rec.DeploymentTemplate.EnvVars {
			t, err := template.New("env").Funcs(funcMap).Parse(env.Default)
			if err != nil {
				continue
			}
			var buf strings.Builder
			if err := t.Execute(&buf, ctx); err == nil {
				rec.DeploymentTemplate.EnvVars[i].Default = buf.String()
			}
		}
	}
}

func updateEnvVar(rec *discovery.Recommendation, key, value string) {
	for i, env := range rec.DeploymentTemplate.EnvVars {
		if env.Name == key {
			rec.DeploymentTemplate.EnvVars[i].Default = value
			return
		}
	}

	rec.DeploymentTemplate.EnvVars = append(rec.DeploymentTemplate.EnvVars, schema.ConfigVar{
		Name:    key,
		Default: value,
		Hidden:  true,
	})
}

func buildDockerPortMap() (map[int]string, error) {
	portMap := make(map[int]string)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	for _, c := range containers {
		imageName := strings.Split(c.Image, "@")[0]
		for _, port := range c.Ports {
			if port.PublicPort > 0 {
				portMap[int(port.PublicPort)] = imageName
			}
		}
	}
	return portMap, nil
}
