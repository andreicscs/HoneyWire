package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func AtomicWriteFile(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".honeywire-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if err := os.Chmod(tmpPath, 0600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set temp file permissions: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write config data: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to fsync config data: %w", err)
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to atomically rename config: %w", err)
	}

	return nil
}
