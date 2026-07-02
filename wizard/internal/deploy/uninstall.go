package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func Uninstall() error {
	cmdBase, err := GetDockerCommand()
	if err != nil {
		return err
	}

	composePath := filepath.Join(DeployDir, ComposeFile)
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("no active deployment found at %s", composePath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := append(cmdBase, "-f", ComposeFile, "-p", ProjectName, "down", "-v", "--remove-orphans", "--rmi", "all")
	// codeql[go/command-injection] Hardcoded/trusted CLI arguments.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = DeployDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tear down sensors: %w", err)
	}

	if err := os.Remove(composePath); err != nil {
		return fmt.Errorf("failed to remove compose file: %w", err)
	}

	os.Remove(composePath + ".bak")

	if entries, err := os.ReadDir(DeployDir); err == nil && len(entries) == 0 {
		os.Remove(DeployDir)
	}

	if execPath, err := os.Executable(); err == nil {
		if err := os.Remove(execPath); err != nil {
			fmt.Printf("⚠️  Could not remove HoneyWire binary at %s. You may need to delete it manually.\n", execPath)
		}
	}

	return nil
}
