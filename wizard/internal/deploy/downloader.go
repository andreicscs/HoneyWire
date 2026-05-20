package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/honeywire/wizard/internal/cli"
)

// DownloadResult holds the path and metadata of a downloaded file.
type DownloadResult struct {
	Path     string
	Size     int64
	Checksum string
}

// Fetch downloads a file to a temporary directory using Go's net/http,
// verifies its SHA-256 checksum, and returns the path.
func Fetch(url, expectedSHA256 string) (*DownloadResult, error) {
	tmpDir, err := os.MkdirTemp("", "honeywire-download-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	filename := filepath.Base(url)
	if filename == "" || filename == "." {
		filename = "download"
	}
	destPath := filepath.Join(tmpDir, filename)

	fmt.Printf("      ↳ Downloading %s...%s\n", cli.Dim, cli.Reset)
	if err := downloadFile(url, destPath); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("      ↳ Verifying SHA-256 checksum...%s%s\n", cli.Dim, cli.Reset)
	actual, err := sha256File(destPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("checksum calculation failed: %w", err)
	}

	if expectedSHA256 != "" {
		expected := strings.TrimSpace(expectedSHA256)
		if !strings.EqualFold(actual, expected) {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("SHA-256 mismatch: expected %s, got %s", expected, actual)
		}
		fmt.Printf("      ↳ Checksum verified.%s%s\n", cli.Green, cli.Reset)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to stat downloaded file: %w", err)
	}

	return &DownloadResult{
		Path:     destPath,
		Size:     info.Size(),
		Checksum: actual,
	}, nil
}

func FetchWithRemoteChecksum(binaryURL string) (*DownloadResult, error) {
	checksumURL := binaryURL + ".sha256"
	checksumResult, err := Fetch(checksumURL, "")
	if err != nil {
		return nil, fmt.Errorf("failed to download checksum file: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(checksumResult.Path))

	checksumData, err := os.ReadFile(checksumResult.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read checksum file: %w", err)
	}

	expectedHash := strings.Fields(string(checksumData))[0]
	return Fetch(binaryURL, expectedHash)
}

func (r *DownloadResult) Cleanup() error {
	if r.Path != "" {
		return os.RemoveAll(filepath.Dir(r.Path))
	}
	return nil
}

func downloadFile(url, destPath string) error {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if written == 0 {
		return fmt.Errorf("downloaded file is empty")
	}

	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
