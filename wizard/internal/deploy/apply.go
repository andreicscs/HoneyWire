package deploy

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"strings"

	"github.com/honeywire/wizard/internal/cli"
)

func performRollback(reason string, hasBackup bool, backupPath, composePath string, cmdBase []string) error {
	fmt.Printf("\n%s[!] %s. Attempting rollback...%s\n", cli.Yellow, reason, cli.Reset)

	if hasBackup {
		if rollbackErr := copyFile(backupPath, composePath); rollbackErr != nil {
			fmt.Printf("%s[!] Rollback failed to restore previous compose: %v%s\n", cli.Red, rollbackErr, cli.Reset)
			fmt.Printf("    Manual recovery may be required at: %s\n", composePath)
		} else {
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer rollbackCancel()

			rollbackArgs := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "up", "-d", "--remove-orphans")
			rollbackCmd := exec.CommandContext(rollbackCtx, rollbackArgs[0], rollbackArgs[1:]...)
			rollbackCmd.Dir = DeployDir
			rollbackCmd.Stdout = os.Stdout
			rollbackCmd.Stderr = os.Stderr

			if rollbackErr := rollbackCmd.Run(); rollbackErr != nil {
				fmt.Printf("%s[!] Rollback compose up failed: %v%s\n", cli.Red, rollbackErr, cli.Reset)
				fmt.Printf("    Manual recovery may be required at: %s\n", composePath)
			} else {
				fmt.Printf("%s[*] Rolled back to previous compose configuration.%s\n", cli.Yellow, cli.Reset)
				os.Remove(backupPath)
			}
		}
	} else {
		fmt.Printf("%s[!] No backup available for rollback.%s\n", cli.Red, cli.Reset)
		if removeErr := os.Remove(composePath); removeErr != nil {
			fmt.Printf("%s[!] Warning: failed to remove failed compose file: %v%s\n", cli.Yellow, removeErr, cli.Reset)
		}
	}

	return fmt.Errorf("deployment failed: %s", reason)
}

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

	backupPath := composePath + ".backup.yml"
	hasBackup := false
	if _, err := os.Stat(composePath); err == nil {
		if err := copyFile(composePath, backupPath); err != nil {
			fmt.Printf("%s[!] Warning: failed to snapshot current compose: %v%s\n", cli.Yellow, err, cli.Reset)
		} else {
			hasBackup = true
		}
	}

	if err := os.Rename(newComposePath, composePath); err != nil {
		os.Remove(newComposePath)
		return fmt.Errorf("failed to swap compose file: %w", err)
	}

	fmt.Printf("%s[*] Pulling latest container images...%s\n", cli.Cyan, cli.Reset)

	pullCtx, pullCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer pullCancel()

	pullArgs := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "pull")
	pullCmd := exec.CommandContext(pullCtx, pullArgs[0], pullArgs[1:]...)
	pullCmd.Dir = DeployDir
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr

	if err := pullCmd.Run(); err != nil {
		fmt.Printf("%s[!] Warning: Failed to pull latest images. Continuing with local cache. Error: %v%s\n", cli.Yellow, err, cli.Reset)
	}

	upCtx, upCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer upCancel()

	upArgs := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "up", "-d", "--pull", "always", "--remove-orphans")
	upCmd := exec.CommandContext(upCtx, upArgs[0], upArgs[1:]...)
	upCmd.Dir = DeployDir
	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr

	if err := upCmd.Run(); err != nil {
		return performRollback(fmt.Sprintf("Compose up failed: %v", err), hasBackup, backupPath, composePath, cmdBase)
	}

	fmt.Printf("%s[*] Verifying deployment success...%s\n", cli.Cyan, cli.Reset)
	time.Sleep(3 * time.Second)

	checkCtx, checkCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer checkCancel()

	checkArgs := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "ps", "--services", "--filter", "status=exited", "--filter", "status=restarting")
	checkCmd := exec.CommandContext(checkCtx, checkArgs[0], checkArgs[1:]...)
	checkCmd.Dir = DeployDir

	output, _ := checkCmd.Output()
	failedServices := strings.TrimSpace(string(output))

	if failedServices != "" {
		failedList := strings.ReplaceAll(failedServices, "\n", ", ")
		return performRollback(fmt.Sprintf("Sensors crashed after startup: %s", failedList), hasBackup, backupPath, composePath, cmdBase)
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
