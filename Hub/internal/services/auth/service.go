package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// --- Brute-Force Protection State ---
type loginState struct {
	attempts    int
	lockedUntil time.Time
}

type Store interface {
	GetConfigValue(key string) (string, error)
	GetNodeByKey(token string) (string, error)
}

type Service struct {
	store             Store
	dashboardPassword string

	// Session management
	sessionMu sync.RWMutex
	sessions  map[string]time.Time

	// Brute-force protection
	authTracker map[string]*loginState
	authMutex   sync.Mutex

	// Node auth cache
	nodeAuthCache sync.Map
}

func NewService(store Store, dashboardPassword string) *Service {
	return &Service{
		store:             store,
		dashboardPassword: dashboardPassword,
		sessions:          make(map[string]time.Time),
		authTracker:       make(map[string]*loginState),
	}
}

// StartWorkers starts background goroutines for cleaning up sessions and brute-force trackers.
func (s *Service) StartWorkers(ctx context.Context) {
	log.Println("[Auth] Worker started.")
	go s.cleanupSessions(ctx)
	go s.cleanupAuthTracker(ctx)
}

// --- Session Management ---

func (s *Service) cleanupSessions(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("[Auth] Worker stopped.")
			return
		case <-ticker.C:
			s.sessionMu.Lock()
			now := time.Now()
			for token, exp := range s.sessions {
				if now.After(exp) {
					delete(s.sessions, token)
				}
			}
			s.sessionMu.Unlock()
		}
	}
}

func (s *Service) CreateSession() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)

	s.sessionMu.Lock()
	s.sessions[token] = time.Now().Add(30 * 24 * time.Hour)
	s.sessionMu.Unlock()

	return token, nil
}

func (s *Service) IsValid(token string) bool {
	s.sessionMu.RLock()
	expiration, exists := s.sessions[token]
	s.sessionMu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiration) {
		s.sessionMu.Lock()
		delete(s.sessions, token)
		s.sessionMu.Unlock()
		return false
	}

	return true
}

func (s *Service) DeleteSession(token string) {
	s.sessionMu.Lock()
	delete(s.sessions, token)
	s.sessionMu.Unlock()
}

func (s *Service) ClearAllSessions() {
	s.sessionMu.Lock()
	s.sessions = make(map[string]time.Time)
	s.sessionMu.Unlock()
}

// --- UI Authentication ---

func (s *Service) cleanupAuthTracker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.authMutex.Lock()
			now := time.Now()
			for ip, state := range s.authTracker {
				if now.After(state.lockedUntil) {
					delete(s.authTracker, ip)
				}
			}
			s.authMutex.Unlock()
		}
	}
}

func (s *Service) Login(password, ip string) (string, error) {
	// Rate Limiter Pre-Check
	s.authMutex.Lock()
	if state, exists := s.authTracker[ip]; exists {
		if state.attempts >= 10 {
			if time.Now().Before(state.lockedUntil) {
				s.authMutex.Unlock()
				return "", errors.New("too_many_requests")
			}
			delete(s.authTracker, ip)
		}
	}
	s.authMutex.Unlock()

	// Evaluate Authorization
	var isAuthorized bool
	if s.dashboardPassword != "" {
		isAuthorized = subtle.ConstantTimeCompare([]byte(password), []byte(s.dashboardPassword)) == 1
	} else {
		dbHash, err := s.store.GetConfigValue("admin_hash")
		if err == nil {
			err = bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(password))
			isAuthorized = (err == nil)
		}
	}

	if isAuthorized {
		s.authMutex.Lock()
		delete(s.authTracker, ip)
		s.authMutex.Unlock()
		return s.CreateSession()
	}

	// Handle Failure & Increment Rate Limiter
	s.authMutex.Lock()
	if _, exists := s.authTracker[ip]; !exists {
		s.authTracker[ip] = &loginState{}
	}
	s.authTracker[ip].attempts++
	if s.authTracker[ip].attempts >= 10 {
		s.authTracker[ip].lockedUntil = time.Now().Add(15 * time.Minute)
		log.Printf("[!] AUDIT: IP %s locked out of dashboard for 15 minutes due to brute-force", ip)
	}
	s.authMutex.Unlock()

	return "", errors.New("invalid_password")
}

// --- Node Authentication ---

func (s *Service) AuthenticateNodeRequest(token string) (string, error) {
	if token == "" {
		return "", errors.New("missing token")
	}

	if cachedNodeID, ok := s.nodeAuthCache.Load(token); ok {
		return cachedNodeID.(string), nil
	}

	nodeID, err := s.store.GetNodeByKey(token)
	if err != nil || nodeID == "" {
		return "", errors.New("invalid node api key")
	}

	s.nodeAuthCache.Store(token, nodeID)
	return nodeID, nil
}
