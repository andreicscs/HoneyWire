package deploy

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/honeywire/wizard/internal/cli"
)

func Apply(ctx context.Context, composeData []byte) error {
	cmdBase, err := GetDockerCommand()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(DeployDir, 0750); err != nil {
		return fmt.Errorf("failed to create deployment directory: %w", err)
	}

	composePath := filepath.Join(DeployDir, ComposeFile)
	newComposePath := composePath + ".new"
	if err := os.WriteFile(newComposePath, composeData, 0600); err != nil {
		return fmt.Errorf("failed to write new compose file: %w", err)
	}

	validateCtx, validateCancel := context.WithTimeout(context.Background(), commandTimeout)
	defer validateCancel()

	validateArgs := append(cmdBase, "-f", newComposePath, "config", "--quiet")
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	// codeql[go/command-injection] Hardcoded/trusted CLI arguments.
	validateCmd := exec.CommandContext(validateCtx, validateArgs[0], validateArgs[1:]...)
	validateCmd.Dir = DeployDir

	if output, err := validateCmd.CombinedOutput(); err != nil {
		os.Remove(newComposePath)
		return fmt.Errorf("compose validation failed: %s\nOutput: %s", err, string(output))
	}

	backupPath := composePath + ".bak"
	hasBackup := false
	if _, err := os.Stat(composePath); err == nil {
		if err := copyFile(composePath, backupPath); err != nil {
			fmt.Printf("%s[!] Warning: failed to backup current compose: %v%s\n", cli.Yellow, err, cli.Reset)
		} else {
			hasBackup = true
		}
	}

	if err := os.Rename(newComposePath, composePath); err != nil {
		os.Remove(newComposePath)
		return fmt.Errorf("failed to swap compose file: %w", err)
	}

	// Explicitly pull latest images before starting
	fmt.Printf("%s[*] Pulling latest container images...%s\n", cli.Cyan, cli.Reset)
	
	pullCtx, pullCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer pullCancel()

	pullArgs := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "pull")
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	// codeql[go/command-injection] Hardcoded/trusted CLI arguments.
	pullCmd := exec.CommandContext(pullCtx, pullArgs[0], pullArgs[1:]...)
	pullCmd.Dir = DeployDir
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr

	if err := pullCmd.Run(); err != nil {
		// If pulling fails (e.g., registry down, no internet), you might want to log it
		// but continue to try 'up' with whatever local images you have as a fallback.
		// Alternatively, you can return an error here to halt deployment.
		fmt.Printf("%s[!] Warning: Failed to pull latest images. Continuing with local cache. Error: %v%s\n", cli.Yellow, err, cli.Reset)
	}

	upCtx, upCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer upCancel()

	upArgs := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "up", "-d", "--pull", "always", "--remove-orphans")
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	// codeql[go/command-injection] Hardcoded/trusted CLI arguments.
	upCmd := exec.CommandContext(upCtx, upArgs[0], upArgs[1:]...)
	upCmd.Dir = DeployDir
	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr

	if err := upCmd.Run(); err != nil {
		fmt.Printf("\n%s[!] Compose up failed. Attempting rollback...%s\n", cli.Yellow, cli.Reset)

		if hasBackup {
			if rollbackErr := copyFile(backupPath, composePath); rollbackErr != nil {
				fmt.Printf("%s[!] Rollback also failed: %v%s\n", cli.Red, rollbackErr, cli.Reset)
				fmt.Printf("    Manual recovery may be required at: %s\n", composePath)
			} else {
				fmt.Printf("%s[*] Rolled back to previous compose configuration.%s\n", cli.Yellow, cli.Reset)
			}
		} else {
			fmt.Printf("%s[!] No backup available for rollback.%s\n", cli.Red, cli.Reset)
		}

		return fmt.Errorf("docker compose up failed: %w", err)
	}

	if hasBackup {
		os.Remove(backupPath)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := os.Chmod(dst, 0600); err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
