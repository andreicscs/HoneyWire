package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcScanner_DirtyNetTCP(t *testing.T) {
	// This tests silent wrong discovery from bad text parsing.
	// We inject bad hex, wrong states, short lines, and valid lines.
	mockTCP := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 c1 100 0 0 10 -1
   1: 00000000:XXXX 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12346 1 c1 100 0 0 10 -1
   2: 00000000:22B8 00000000:0000 01 00000000:00000000 00:00000000 00000000     0        0 12347 1 c1 100 0 0 10 -1
   3: short line string here
`
	// Valid: 1F90 -> Port 8080 (State 0A -> LISTEN, inode 12345)
	// Invalid: XXXX -> Bad Hex
	// Invalid: 22B8 -> Port 8888, but State 01 (ESTABLISHED, not LISTEN)

	tmpFile := filepath.Join(t.TempDir(), "tcp")
	os.WriteFile(tmpFile, []byte(mockTCP), 0644)

	p := NewProcScanner()
	p.netFiles = []string{tmpFile}

	portMap := p.buildInodePortMap()

	require.Len(t, portMap, 1, "Should only parse the single valid listening port")
	assert.Equal(t, 8080, portMap["12345"])
}

func TestProcScanner_PartialProcFailure(t *testing.T) {
	// This ensures real-world runtime permissions and transiently dying processes
	// don't panic or halt the entire discovery sequence.
	tmpDir := t.TempDir()

	// 1. Valid Process (PID 100)
	pid100 := filepath.Join(tmpDir, "100")
	os.MkdirAll(filepath.Join(pid100, "fd"), 0755)
	os.WriteFile(filepath.Join(pid100, "comm"), []byte("nginx\n"), 0644)
	os.Symlink("socket:[99999]", filepath.Join(pid100, "fd", "3")) // matches our mock inode

	// 2. Dead Process (PID 101) - Missing comm file
	pid101 := filepath.Join(tmpDir, "101")
	os.MkdirAll(pid101, 0755)
	// no comm file

	// 3. Permission Denied Process (PID 102) - Unreadable fd directory
	pid102 := filepath.Join(tmpDir, "102")
	os.MkdirAll(pid102, 0755)
	os.WriteFile(filepath.Join(pid102, "comm"), []byte("root_proc\n"), 0644)
	// Instead of a directory, write a file so ReadDir fails gracefully
	os.WriteFile(filepath.Join(pid102, "fd"), []byte("not_a_dir"), 0644)

	// 4. Ignored Process (PID 103) - Should be skipped
	pid103 := filepath.Join(tmpDir, "103")
	os.MkdirAll(filepath.Join(pid103, "fd"), 0755)
	os.WriteFile(filepath.Join(pid103, "comm"), []byte("honeywire-hub\n"), 0644)
	os.Symlink("socket:[88888]", filepath.Join(pid103, "fd", "3"))

	// Mock TCP net file
	mockTCP := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 99999 1 c1 100 0 0 10 -1
   1: 00000000:0051 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 88888 1 c1 100 0 0 10 -1
`
	tcpFile := filepath.Join(tmpDir, "tcp")
	os.WriteFile(tcpFile, []byte(mockTCP), 0644)

	// Initialize scanner
	p := NewProcScanner()
	p.procPath = tmpDir
	p.netFiles = []string{tcpFile}

	// Scan
	state, err := p.Scan(nil)

	require.NoError(t, err)
	require.NotNil(t, state)

	// We expect EXACTLY 1 service.
	// - PID 101 died (skipped)
	// - PID 102 permission denied (skipped gracefully)
	// - PID 103 is on the ignore list (skipped)
	require.Len(t, state.Services, 1)
	assert.Equal(t, "nginx", state.Services[0].ProcessName)
	assert.Equal(t, 80, state.Services[0].Port)
	assert.Equal(t, 100, state.Services[0].PID)
}
