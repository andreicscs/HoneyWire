package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const CookieName = "hw_auth"

// Mutex for thread safety
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]time.Time),
	}
}

// Create generates a secure 32-byte hex token and stores it for 30 days
func (s *SessionStore) Create() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)

	s.mu.Lock()
	s.sessions[token] = time.Now().Add(30 * 24 * time.Hour)
	s.mu.Unlock()

	return token, nil
}

func (s *SessionStore) IsValid(token string) bool {
	s.mu.RLock()
	expiration, exists := s.sessions[token]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiration) {
		s.mu.Lock()
		delete(s.sessions, token) // Clean up expired session
		s.mu.Unlock()
		return false
	}

	return true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}