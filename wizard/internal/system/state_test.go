package system

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckLoad(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "loadavg")
	procLoadAvgPath = tmpFile // Override
	defer func() { procLoadAvgPath = "/proc/loadavg" }()

	// Test 1: OK Load
	os.WriteFile(tmpFile, []byte("1.20 0.80 0.50 1/100 1234\n"), 0644)
	warn, err := CheckLoad()
	require.NoError(t, err)
	assert.Empty(t, warn)

	// Test 2: High Load
	os.WriteFile(tmpFile, []byte("5.10 4.80 4.50 2/200 5678\n"), 0644)
	warn, err = CheckLoad()
	require.NoError(t, err)
	assert.Contains(t, warn, "High CPU load detected")
}

func TestCheckMemory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "meminfo")
	procMemInfoPath = tmpFile // Override
	defer func() { procMemInfoPath = "/proc/meminfo" }()

	// Test 1: OK Memory (1GB available)
	os.WriteFile(tmpFile, []byte("MemTotal: 2048000 kB\nMemAvailable: 1048576 kB\n"), 0644)
	warn, err := CheckMemory()
	require.NoError(t, err)
	assert.Empty(t, warn)

	// Test 2: Low Memory (400MB available)
	os.WriteFile(tmpFile, []byte("MemTotal: 2048000 kB\nMemAvailable: 409600 kB\n"), 0644)
	warn, err = CheckMemory()
	require.NoError(t, err)
	assert.Contains(t, warn, "Low memory detected")
}

func TestCheckDiskSpace(t *testing.T) {
	origStat := statfsFunc
	defer func() { statfsFunc = origStat }()

	// Mock struct block sizes (using 4KB block size for mock)
	var mockBsize uint64 = 4096

	// Test 1: OK Disk (2GB free)
	statfsFunc = func(path string, buf *syscall.Statfs_t) error {
		buf.Bsize = int64(mockBsize) // Explicit cast to int64 for cross-platform support
		buf.Bavail = uint64((2 * 1024 * 1024 * 1024) / mockBsize)
		return nil
	}
	warn, err := CheckDiskSpace()
	require.NoError(t, err)
	assert.Empty(t, warn)

	// Test 2: Low Disk (0.5GB free)
	statfsFunc = func(path string, buf *syscall.Statfs_t) error {
		buf.Bsize = int64(mockBsize) // Explicit cast to int64 for cross-platform support
		buf.Bavail = uint64((500 * 1024 * 1024) / mockBsize)
		return nil
	}
	warn, err = CheckDiskSpace()
	require.NoError(t, err)
	assert.Contains(t, warn, "Low disk space detected")
}

func TestLoadCurrentState_ParsingCorrectness(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "honeywire-compose.yml")
	composeFilePath = tmpFile // Override
	defer func() { composeFilePath = "honeywire-compose.yml" }()

	yamlData := `
services:
  sensor-web:
    image: honeywire/nginx:latest
    ports:
      - "8080:80"
      - "8443:443/tcp"
  sensor-db:
    image: honeywire/postgres:13
    ports:
      - "5432:5432"
  sensor-cache:
    image: honeywire/nginx:latest # Intentional duplicate to test deduplication
`
	os.WriteFile(tmpFile, []byte(yamlData), 0644)

	state, err := LoadCurrentState()
	require.NoError(t, err)

	// Assert ports are extracted correctly
	assert.ElementsMatch(t, []int{8080, 8443, 5432}, state.ManagedPorts)

	// Assert images are deduplicated
	assert.ElementsMatch(t, []string{"honeywire/nginx:latest", "honeywire/postgres:13"}, state.DeployedImages)
}
