package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// --- Mock Store ---
type mockAuthStore struct {
	configMap map[string]string
	nodeMap   map[string]string
	nodeCalls int
}

func (m *mockAuthStore) GetConfigValue(key string) (string, error) {
	if val, ok := m.configMap[key]; ok {
		return val, nil
	}
	return "", errors.New("not found")
}

func (m *mockAuthStore) GetNodeByKey(token string) (string, error) {
	m.nodeCalls++
	if val, ok := m.nodeMap[token]; ok {
		return val, nil
	}
	return "", errors.New("not found")
}

// --- Tests ---

func TestSessionManagement(t *testing.T) {
	svc := NewService(&mockAuthStore{}, "dummy")

	// 1. Create Session
	token, err := svc.CreateSession()
	require.NoError(t, err)
	assert.Len(t, token, 64, "Token should be 64 hex chars (32 bytes)")

	// 2. IsValid works
	assert.True(t, svc.IsValid(token), "Session should be valid")
	assert.False(t, svc.IsValid("unknown_token"), "Unknown token should be invalid")

	// 3. Expired Session
	svc.sessionMu.Lock()
	svc.sessions[token] = time.Now().Add(-1 * time.Hour) // Force expire
	svc.sessionMu.Unlock()

	assert.False(t, svc.IsValid(token), "Expired session should be invalid and deleted")

	// 4. Delete Session
	token2, _ := svc.CreateSession()
	svc.DeleteSession(token2)
	assert.False(t, svc.IsValid(token2), "Deleted session should be invalid")
}

func TestBruteForceProtection(t *testing.T) {
	svc := NewService(&mockAuthStore{}, "correct_pass")
	ip := "192.168.1.100"

	// 1. Fail 9 times
	for i := 0; i < 9; i++ {
		_, err := svc.Login("wrong_pass", ip)
		assert.EqualError(t, err, "invalid_password")
	}

	// 2. 10th failure triggers lockout
	_, err := svc.Login("wrong_pass", ip)
	assert.EqualError(t, err, "invalid_password") // The 10th attempt still returns invalid_password

	// 3. 11th attempt is blocked entirely
	_, err = svc.Login("correct_pass", ip) // Even with correct pass, they are locked out
	assert.EqualError(t, err, "too_many_requests")

	// 4. Correct login clears tracker (test with different IP)
	ip2 := "10.0.0.5"
	svc.Login("wrong_pass", ip2) // 1 strike
	_, err = svc.Login("correct_pass", ip2)
	require.NoError(t, err)

	svc.authMutex.Lock()
	_, exists := svc.authTracker[ip2]
	svc.authMutex.Unlock()
	assert.False(t, exists, "Successful login should clear brute-force tracker")
}

func TestLogin_Bcrypt(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("db_secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	store := &mockAuthStore{
		configMap: map[string]string{"admin_hash": string(hash)},
	}
	// No hardcoded password, forces bcrypt branch
	svc := NewService(store, "")

	// Failure
	_, err = svc.Login("wrong", "1.1.1.1")
	assert.Error(t, err)

	// Success
	token, err := svc.Login("db_secret", "1.1.1.1")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthenticateNodeRequest_Caching(t *testing.T) {
	store := &mockAuthStore{
		nodeMap: map[string]string{"valid-api-key": "node-123"},
	}
	svc := NewService(store, "")

	// 1. Empty token
	_, err := svc.AuthenticateNodeRequest("")
	assert.Error(t, err)

	// 2. Invalid token
	_, err = svc.AuthenticateNodeRequest("bad-key")
	assert.Error(t, err)
	assert.Equal(t, 1, store.nodeCalls) // hit db

	// 3. Valid token (First call)
	nodeID, err := svc.AuthenticateNodeRequest("valid-api-key")
	require.NoError(t, err)
	assert.Equal(t, "node-123", nodeID)
	assert.Equal(t, 2, store.nodeCalls) // hit db

	// 4. Valid token (Second call) -> MUST use cache
	nodeID2, err := svc.AuthenticateNodeRequest("valid-api-key")
	require.NoError(t, err)
	assert.Equal(t, "node-123", nodeID2)

	// Assert database was NOT called again
	assert.Equal(t, 2, store.nodeCalls, "Node API key should have been cached")
}
