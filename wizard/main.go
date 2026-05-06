package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"math/rand"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/honeywire/wizard/pkg/api"
	"github.com/honeywire/wizard/pkg/autodiscovery"
	"github.com/honeywire/wizard/pkg/generator"
	"github.com/honeywire/wizard/pkg/scanner"
	"github.com/honeywire/wizard/pkg/schema"
	"github.com/honeywire/wizard/pkg/state"
)

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Cyan    = "\033[36m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Red     = "\033[31m"
	Magenta = "\033[35m"
	Gray    = "\033[90m"
)

type NodeConfig struct {
	HubURL  string `json:"hub_url"`
	NodeID  string `json:"node_id"`
	NodeKey string `json:"node_key"`
	Alias   string `json:"alias"`
}

const configPath = "/etc/honeywire/config.json"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s✖ Fatal Error: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
}

func run() error {
	uninstallPtr := flag.Bool("uninstall", false, "Tear down and remove all managed sensors from this node")
	registryPtr := flag.String("registry", api.DefaultRegistryURL, "URL or local path to manifests.json")
	linkURL := flag.String("link", "", "Hub URL to link to (e.g., http://hub:8080)")
	flag.Parse()

	if warning, _ := state.CheckRoot(); warning != "" {
		return fmt.Errorf("Wizard must be run as root (sudo). Deep system access required")
	}

	// Handle teardown
	if *uninstallPtr {
		fmt.Printf("\n%s%s=== HoneyWire Sensor Teardown ===%s\n\n", Bold, Cyan, Reset)
		if err := generator.Uninstall(); err != nil {
			return fmt.Errorf("Teardown failed: %w", err)
		}
		fmt.Printf("%s✅ All HoneyWire sensors have been successfully removed.%s\n\n", Green, Reset)
		return nil
	}

	// Handle explicit linking
	if *linkURL != "" {
		token := flag.Arg(0)
		if token == "" {
			return fmt.Errorf("Missing token. Usage: ./wizard --link <url> <token>")
		}

		fmt.Printf("\n%s%s=== HoneyWire Provisioning ===%s\n\n", Bold, Cyan, Reset)
		fmt.Printf("%s[*] Negotiating with Hub at %s...%s\n", Bold, *linkURL, Reset)

		if err := performLink(*linkURL, token); err != nil {
			return fmt.Errorf("Linking failed: %w", err)
		}

		fmt.Printf("%s✅ Successfully linked to Hub! Run ./wizard again to deploy sensors.%s\n\n", Green, Reset)
		return nil
	}

	fmt.Printf("\n%s%s=== HoneyWire Infrastructure Auditor v1.0 ===%s\n\n", Bold, Cyan, Reset)

	nodeConfig, err := loadNodeConfig()
	if err != nil {
		fmt.Printf("%s✖ Node is not provisioned!%s\n", Red, Reset)
		fmt.Printf("%s↳ Please click 'Add Node' in the HoneyWire Dashboard and run the provided --link command.%s\n\n", Dim, Reset)
		os.Exit(1)
	}
	fmt.Printf("%s[*] Authenticated as Node: %s%s%s\n\n", Dim, Cyan, nodeConfig.Alias, Reset)

	// Resource checks
	checks := []func() (string, error){state.CheckMemory, state.CheckLoad, state.CheckDiskSpace}
	for _, check := range checks {
		if warning, err := check(); err == nil && warning != "" {
			fmt.Printf("%s%s%s\n", Yellow, warning, Reset)
		}
	}

	systemState, err := state.LoadCurrentState()
	if err != nil {
		return fmt.Errorf("failed to load system state: %w", err)
	}

	fmt.Printf("%s[*] Step 1/3: Analyzing Host OS & Sockets...%s\n", Bold, Reset)
	hostScanner := scanner.NewProcScanner()
	hostState, err := hostScanner.Scan(systemState)
	if err != nil {
		return fmt.Errorf("failed to scan host: %w", err)
	}

	fmt.Printf("%s[*] Step 2/3: Interrogating Docker Daemon...%s\n", Bold, Reset)
	dockerMap, dockerErr := buildDockerPortMap()
	if dockerErr != nil {
		fmt.Printf("    %s↳ Warning: Cannot access Docker API (%v). Proceeding with native scanning.%s\n", Yellow, dockerErr, Reset)
	} else {
		fmt.Printf("    %s↳ Successfully mapped %d containerized ports.%s\n", Green, len(dockerMap), Reset)
	}

	fmt.Println()
	printCorrelatedServices(hostState, dockerMap)

	fmt.Printf("\n%s[*] Step 3/3: Formulating Deception Strategy...%s\n", Bold, Reset)

	manifests, apiErr := api.FetchManifests(*registryPtr)
	if apiErr != nil {
		return fmt.Errorf("Registry error: %w", apiErr)
	}

	fmt.Printf("    %s↳ Synced %d active sensors from: %s%s\n", Green, len(manifests), *registryPtr, Reset)

	engine := autodiscovery.NewEngine(manifests)
	recommendations := engine.GetRecommendations(hostState, systemState)

	renderManifestTemplates(recommendations, dockerMap, hostState)

	for _, rec := range recommendations {
		updateEnvVar(rec, "HW_HUB_ENDPOINT", nodeConfig.HubURL)
		updateEnvVar(rec, "HW_HUB_KEY", nodeConfig.NodeKey)
		updateEnvVar(rec, "HW_NODE_ID", nodeConfig.NodeID)
		updateEnvVar(rec, "HW_SENSOR_ID", rec.SensorID)
	}

	fmt.Println()
	if len(recommendations) == 0 {
		fmt.Printf("    %sNo recommendations at this time.%s\n", Dim, Reset)
		return nil
	}

	printDeploymentPlan(recommendations)

	if len(systemState.DeployedImages) > 0 {
		fmt.Printf("    %s↳ Note: %d sensors are already actively managed. (Skipped)%s\n\n", Green, len(systemState.DeployedImages), Reset)
	}

	printExpectedImpact(recommendations)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("    🚀 Deploy these %d sensors now? [Y/n/edit]: ", len(recommendations))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "y" || input == "yes" || input == "" {
			fmt.Printf("\n    %s⚙️ Executing Infrastructure-as-Code...%s\n", Cyan, Reset)
			
			// Trigger the deployment
			if err := generator.Apply(recommendations); err != nil {
				return fmt.Errorf("Deployment failed: %w", err)
			}
			
			fmt.Printf("    %s✅ Deployment complete! Run 'docker compose -f %s -p honeywire ps' to view your sensors.%s\n\n", Green, filepath.Join(generator.DeployDir, generator.ComposeFile), Reset)
			break
		}

		if input == "n" || input == "no" {
			fmt.Printf("\n    %sDeployment aborted.%s\n\n", Dim, Reset)
			break
		}

		if input == "edit" || input == "e" {
			recommendations = editRecommendations(recommendations, reader, dockerMap)
			fmt.Println()
			printDeploymentPlan(recommendations)
			printExpectedImpact(recommendations)
			continue
		}

		fmt.Printf("    %sInvalid input. Please type Y, n, or edit.%s\n", Red, Reset)
	}

	return nil
}

// --- PROVISIONING HELPERS ---

func performLink(hubURL, token string) error {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown-node"
	}

	payload := map[string]string{
		"token": token,
		"alias": hostname,
	}
	body, _ := json.Marshal(payload)

	endpoint := strings.TrimRight(hubURL, "/") + "/api/v1/wizard/link"
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("network error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hub rejected request (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result NodeConfig
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse hub response: %v", err)
	}
	result.HubURL = hubURL

	return saveNodeConfig(result)
}

func saveNodeConfig(cfg NodeConfig) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	// Restrict permissions to root-only read/write to protect the API key
	return os.WriteFile(configPath, data, 0600)
}

func loadNodeConfig() (*NodeConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg NodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}


// --- EXISTING FUNCTIONS (WITH MINOR TWEAKS) ---

func renderManifestTemplates(recs []*autodiscovery.Recommendation, dockerMap map[int]string, hostState *scanner.HostState) {
	// Build the IgnorePorts string and a map of used ports for the resolver
	var portStrs []string
	usedPorts := make(map[int]bool)
	for _, svc := range hostState.Services {
		portStrs = append(portStrs, fmt.Sprintf("%d", svc.Port))
		usedPorts[svc.Port] = true
	}
	ignorePorts := strings.Join(portStrs, ",")

	usedFiles := make(map[string]bool)

	safeFilePick := func(files []string, basePath string) string {
		rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
		for _, f := range files {
			if !usedFiles[f] {
				if _, err := os.Stat(filepath.Join(basePath, f)); os.IsNotExist(err) {
					usedFiles[f] = true
					return f
				}
			}
		}
		fallback := fmt.Sprintf("backup_hw_%d.bak", rand.Intn(99999))
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
			MatchedServices []autodiscovery.MatchedService
		}{
			HasWeb:          hasWeb,
			HasDB:           hasDB,
			IgnorePorts:     ignorePorts,
			TrapPath:        trapPath,
			MatchedServices: rec.MatchedServices,
		}

		for i, vol := range rec.DeploymentTemplate.VolumeMounts {
			if t, err := template.New("vol").Funcs(funcMap).Parse(vol.Source); err == nil {
				var buf bytes.Buffer
				t.Execute(&buf, ctx)
				rec.DeploymentTemplate.VolumeMounts[i].Source = buf.String()
			}
		}

		for i, env := range rec.DeploymentTemplate.EnvVars {
			if t, err := template.New("env").Funcs(funcMap).Parse(env.Default); err == nil {
				var buf bytes.Buffer
				t.Execute(&buf, ctx)
				rec.DeploymentTemplate.EnvVars[i].Default = buf.String()
			}
		}
	}
}

func editRecommendations(recs []*autodiscovery.Recommendation, reader *bufio.Reader, dockerMap map[int]string) []*autodiscovery.Recommendation {
	fmt.Printf("\n    %s[🛠️  Customizing Deployment Plan]%s\n", Bold, Reset)

	activeRecs := recs
	for {
		if len(activeRecs) == 0 {
			break
		}

		fmt.Printf("\n      %sActive Sensors:%s\n", Dim, Reset)
		for i, r := range activeRecs {
			fmt.Printf("        [%d] %-30s\n", i+1, r.SensorName)
		}

		fmt.Print("\n      Action ('1,3' to remove, 'i 2' to inspect, Enter to finish): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" {
			break
		}

		if strings.HasPrefix(input, "i ") || strings.HasPrefix(input, "inspect ") {
			parts := strings.Fields(input)
			if len(parts) >= 2 {
				var choice int
				fmt.Sscanf(parts[1], "%d", &choice)
				if choice > 0 && choice <= len(activeRecs) {
					showSensorDeepDive(activeRecs[choice-1], dockerMap)
				}
			}
			continue
		}

		removeMap := make(map[int]bool)
		for _, valStr := range strings.Split(input, ",") {
			var idx int
			if _, err := fmt.Sscanf(strings.TrimSpace(valStr), "%d", &idx); err == nil {
				removeMap[idx-1] = true
			}
		}

		var updatedRecs []*autodiscovery.Recommendation
		fmt.Println()
		for i, rec := range activeRecs {
			if removeMap[i] {
				fmt.Printf("        %s- Removed: %s%s\n", Red, rec.SensorName, Reset)
			} else {
				updatedRecs = append(updatedRecs, rec)
			}
		}
		activeRecs = updatedRecs
	}
	return activeRecs
}

func showSensorDeepDive(rec *autodiscovery.Recommendation, dockerMap map[int]string) {
	fmt.Printf("\n      %s--- Deep Dive: %s ---%s\n", Cyan, rec.SensorName, Reset)
	fmt.Printf("      %sSummary:%s     %s\n", Bold, Reset, rec.Manifest.Description)
	
	if len(rec.Manifest.Documentation.Sections) > 0 {
		for _, section := range rec.Manifest.Documentation.Sections {
			fmt.Printf("      %s%s:%s\n", Bold, section.Title, Reset)
			for _, item := range section.Content {
				fmt.Printf("        • %s\n", item)
			}
		}
	}

	fmt.Printf("\n      %sForensic Evidence (Why this was chosen):%s\n", Bold, Reset)
	if len(rec.MatchedServices) == 0 {
		fmt.Printf("        ↳ General environment defense requirement.\n")
	} else {
		for _, svc := range rec.MatchedServices {
			name := svc.ProcessName
			origin := "Native"
			if name == "docker-proxy" && dockerMap != nil {
				if img, ok := dockerMap[svc.Port]; ok {
					name = img
					origin = "Container"
				}
			}
			fmt.Printf("        ↳ [%s] %s (PID: %d) actively listening on port %d\n", origin, name, svc.PID, svc.Port)
		}
	}
	fmt.Println()
}

func buildDockerPortMap() (map[int]string, error) {
	portMap := make(map[int]string)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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

func printCorrelatedServices(hostState *scanner.HostState, dockerMap map[int]string) {
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

func printDeploymentPlan(recommendations []*autodiscovery.Recommendation) {
	fmt.Printf("    %sHoneyWire Infrastructure-as-Code (Dry Run):%s\n\n", Gray, Reset)

	for _, rec := range recommendations {
		fmt.Printf("    %s+%s %s%s%s %s(%s)%s\n", Green, Reset, Bold, rec.SensorName, Reset, Gray, rec.SensorID, Reset)
		fmt.Printf("        %sReason:%s %s\n", Cyan, Reset, rec.Reason)
		fmt.Printf("        %sImage:%s  %s\n", Cyan, Reset, rec.DeploymentTemplate.Image)

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
						fmt.Printf("          ~ %s = %s\n", env.Name, val)
					}
				}
			}
		}
		fmt.Println()
	}
}

func printExpectedImpact(recs []*autodiscovery.Recommendation) {
	fmt.Printf("    📊 %sExpected Impact:%s\n", Bold, Reset)
	fmt.Printf("       - Memory: ~%dMB (low overhead)\n", len(recs)*15)
	fmt.Printf("       - CPU:    <1%% (event-driven logic)\n")
	fmt.Printf("       - Net:    Minimal (heartbeat only)\n\n")
}

func updateEnvVar(rec *autodiscovery.Recommendation, key, value string) {
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