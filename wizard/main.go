package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"flag"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/honeywire/wizard/pkg/autodiscovery"
	"github.com/honeywire/wizard/pkg/scanner"
	"github.com/honeywire/wizard/pkg/state"
	"github.com/honeywire/wizard/pkg/api"
)

// ANSI Color Codes for Enterprise UX
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

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s✖ Fatal Error: %v%s\n", Red, err, Reset)
		os.Exit(1)
	}
}

func run() error {
	registryPtr := flag.String("registry", api.DefaultRegistryURL, "URL or local path to manifests.json")
    flag.Parse()

	rand.Seed(time.Now().UnixNano()) //TODO deprecatged...
	fmt.Printf("\n%s%s=== HoneyWire Infrastructure Auditor v1.0 ===%s\n\n", Bold, Cyan, Reset)

	if warning, err := state.CheckRoot(); err == nil && warning != "" {
		fmt.Printf("%s%s%s\n", Yellow, warning, Reset)
	}
	if warning, err := state.CheckMemory(); err == nil && warning != "" {
		fmt.Printf("%s%s%s\n", Yellow, warning, Reset)
	}
	if warning, err := state.CheckLoad(); err == nil && warning != "" {
		fmt.Printf("%s%s%s\n", Yellow, warning, Reset)
	}
	if warning, err := state.CheckDiskSpace(); err == nil && warning != "" {
		fmt.Printf("%s%s%s\n", Yellow, warning, Reset)
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
		fmt.Printf("    %s↳ Warning: Cannot access Docker API (%v). Falling back to purely native OS parsing.%s\n", Yellow, dockerErr, Reset)
		dockerMap = nil
	} else {
		fmt.Printf("    %s↳ Successfully mapped %d containerized ports.%s\n", Green, len(dockerMap), Reset)
	}

	fmt.Println()
	printCorrelatedServices(hostState, dockerMap)

	fmt.Printf("\n%s[*] Step 3/3: Formulating Deception Strategy...%s\n", Bold, Reset)
	
	// Pass the flag value to the fetcher
    manifests, apiErr := api.FetchManifests(*registryPtr)
    if apiErr != nil {
        fmt.Printf("    %s✖ Fatal: Registry error: %v%s\n", Red, apiErr, Reset)
        return apiErr
    }
    
    fmt.Printf("    %s↳ Synced %d active sensors from: %s%s\n", Green, len(manifests), *registryPtr, Reset)

	engine := autodiscovery.NewEngine(manifests)
	recommendations := engine.GetRecommendations(hostState, systemState)

	// --- THE AGNOSTIC TEMPLATE ENGINE ---
	// Evaluates JSON templates dynamically, handling Port Conflicts and Ignore Lists automatically!
	renderManifestTemplates(recommendations, dockerMap, hostState)
	// ------------------------------------

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
			// TODO: Call generator.Apply() here!
			fmt.Printf("    %s✅ Deployment complete! Run 'docker-compose ps' to view your sensors.%s\n\n", Green, Reset)
			break
		}

		if input == "n" || input == "no" {
			fmt.Printf("\n    %sDeployment aborted. Safe travels!%s\n\n", Dim, Reset)
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

// THE AGNOSTIC TEMPLATE ENGINE
func renderManifestTemplates(recs []*autodiscovery.Recommendation, dockerMap map[int]string, hostState *scanner.HostState) {
	// Build the IgnorePorts string and a map of used ports for the resolver
	var portStrs []string
	usedPorts := make(map[int]bool)
	for _, svc := range hostState.Services {
		portStrs = append(portStrs, fmt.Sprintf("%d", svc.Port))
		usedPorts[svc.Port] = true
	}
	ignorePorts := strings.Join(portStrs, ",")

	// State trackers to prevent duplicates during template rendering
	usedFiles := make(map[string]bool)

	// Safe File Picker Logic
	safeFilePick := func(files []string, basePath string) string {
		rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
		for _, f := range files {
			if !usedFiles[f] {
				// Prevent overwriting real host files
				if _, err := os.Stat(filepath.Join(basePath, f)); os.IsNotExist(err) {
					usedFiles[f] = true
					return f
				}
			}
		}
		// Failsafe if all names are taken
		fallback := fmt.Sprintf("backup_hw_%d.bak", rand.Intn(99999))
		usedFiles[fallback] = true
		return fallback
	}

	// Template helper functions available to all Manifests
	funcMap := template.FuncMap{
		"randWebFile": func(basePath string) string {
			return safeFilePick([]string{"wp-config.old.php", ".env.staging", "config.bak.php", "aws_s3_keys.txt"}, basePath)
		},
		"randDBFile": func(basePath string) string {
			return safeFilePick([]string{"dump_2023.sql", "db_backup_prod.sql", ".pgpass.old", "migration_rollback.sql"}, basePath)
		},
		// Dynamic Port Conflict Resolver!
		"availablePort": func(base int) string {
			p := base
			for {
				if !usedPorts[p] {
					// Attempt to bind to the port to guarantee availability
					addr := fmt.Sprintf(":%d", p)
					ln, err := net.Listen("tcp", addr)
					if err == nil {
						ln.Close()
						usedPorts[p] = true // Mark as used so multiple sensors don't grab the same port
						return fmt.Sprintf("%d", p)
					}
				}
				p++
			}
		},
	}

	for _, rec := range recs {
		// Pre-evaluate TrapPath for the context
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

		// Context exposed to the JSON templates
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

		// Compile VolumeMounts
		for i, vol := range rec.DeploymentTemplate.VolumeMounts {
			if t, err := template.New("vol").Funcs(funcMap).Parse(vol.Source); err == nil {
				var buf bytes.Buffer
				t.Execute(&buf, ctx)
				rec.DeploymentTemplate.VolumeMounts[i].Source = buf.String()
			}
		}

		// Compile EnvVars
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

		// INSPECT LOGIC (e.g., "i 2")
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

		// REMOVE LOGIC (e.g., "1,3")
		removeMap := make(map[int]bool)
		for _, valStr := range strings.Split(input, ",") {
			var idx int
			if _, err := fmt.Sscanf(strings.TrimSpace(valStr), "%d", &idx); err == nil {
				removeMap[idx-1] = true // 0-indexed
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
	
	// Show documentation sections instead of old DeepDive/RiskLevel
	if len(rec.Manifest.Documentation.Sections) > 0 {
		for _, section := range rec.Manifest.Documentation.Sections {
			fmt.Printf("      %s%s:%s\n", Bold, section.Title, Reset)
			for _, item := range section.Content {
				fmt.Printf("        • %s\n", item)
			}
		}
	}

	// Concrete Forensic Evidence Printout
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
			// First, check if there are any VISIBLE environment variables
			var hasVisibleEnv bool
			for _, env := range rec.DeploymentTemplate.EnvVars {
				if !env.Hidden && strings.TrimSpace(env.Default) != "" {
					hasVisibleEnv = true
					break
				}
			}
			
			// Only print the Config Variables section if there is at least one visible var
			if hasVisibleEnv {
				fmt.Printf("        %sConfig Variables:%s\n", Cyan, Reset)
				for _, env := range rec.DeploymentTemplate.EnvVars {
					// Skip any variables explicitly marked as hidden in the JSON manifest
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
}