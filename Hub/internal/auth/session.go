package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const CookieName = "hw_auth"

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
}

func NewSessionStore() *SessionStore {
    s := &SessionStore{sessions: make(map[string]time.Time)}
    go s.cleanup()
    return s
}

func (s *SessionStore) cleanup() {
    for {
        time.Sleep(1 * time.Hour)
        s.mu.Lock()
        now := time.Now()
        for token, exp := range s.sessions {
            if now.After(exp) {
                delete(s.sessions, token)
            }
        }
        s.mu.Unlock()
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
		delete(s.sessions, token)
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

// Wipes all active sessions, forcing everyone to log back in.
func (s *SessionStore) ClearAllSessions() {
	s.mu.Lock()
	s.sessions = make(map[string]time.Time)
	s.mu.Unlock()
}