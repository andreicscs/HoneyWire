package api

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/honeywire/hub/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

// --- Brute-Force Protection State ---
type loginState struct {
	attempts    int
	lockedUntil time.Time
}

// Background routine to prevent memory leaks from abandoned IPs
func (h *Handler) cleanupAuthTracker() {
	for {
		time.Sleep(5 * time.Minute)
		h.authMutex.Lock()
		now := time.Now()
		for ip, state := range h.authTracker {
			if now.After(state.lockedUntil) {
				delete(h.authTracker, ip)
			}
		}
		h.authMutex.Unlock()
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ip := h.getRealIP(r)

	// Rate Limiter Pre-Check
	h.authMutex.Lock()
	if state, exists := h.authTracker[ip]; exists {
		if state.attempts >= 10 {
			if time.Now().Before(state.lockedUntil) {
				h.authMutex.Unlock()
				RespondError(w, "Too many failed attempts. Try again later.", http.StatusTooManyRequests)
				return
			}
			// Lockout expired, wipe the slate clean
			delete(h.authTracker, ip)
		}
	}
	h.authMutex.Unlock()

	// Parse Request
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Evaluate Authorization
	var isAuthorized bool

	if h.Cfg.DashboardPassword != "" {
		// Layer A: Infrastructure Override (.env file)
		isAuthorized = subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.Cfg.DashboardPassword)) == 1
	} else {
		// Layer B: Runtime Database Hash (Setup UI)
		dbHash, err := h.Store.GetConfigValue("admin_hash")
		if err == nil {
			err = bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(req.Password))
			isAuthorized = (err == nil)
		}
	}

	if isAuthorized {
		// Clear failed attempts for this IP on successful login
		h.authMutex.Lock()
		delete(h.authTracker, ip)
		h.authMutex.Unlock()

		token, err := h.SessionStore.Create()
		if err != nil {
			RespondError(w, "Session creation failed", http.StatusInternalServerError)
			return
		}

		isProd := h.Cfg.Env == "production"
		http.SetCookie(w, &http.Cookie{
			Name:     auth.CookieName,
			Value:    token,
			MaxAge:   2592000,
			HttpOnly: true,
			Secure:   isProd,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
		})

		SendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	// Handle Failure & Increment Rate Limiter
	h.authMutex.Lock()
	if _, exists := h.authTracker[ip]; !exists {
		h.authTracker[ip] = &loginState{}
	}
	h.authTracker[ip].attempts++

	if h.authTracker[ip].attempts >= 10 {
		h.authTracker[ip].lockedUntil = time.Now().Add(15 * time.Minute)
		log.Printf("[!] AUDIT: IP %s locked out of dashboard for 15 minutes due to brute-force", ip)
	}
	h.authMutex.Unlock()

	RespondError(w, "Invalid Password", http.StatusUnauthorized)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		h.SessionStore.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// validateNodeAuth checks if a request is authenticated for a given node
// Extracts Bearer token from Authorization header and maps it to the expected node_id
func (h *Handler) validateNodeAuth(r *http.Request, expectedNodeID string) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false
	}
	token := parts[1] // This is the api_key (hw_key_...)

	// 1. Check cache first (Key: Token, Value: NodeID)
	if cachedNodeID, ok := h.nodeAuthCache.Load(token); ok {
		return subtle.ConstantTimeCompare([]byte(cachedNodeID.(string)), []byte(expectedNodeID)) == 1
	}

	// 2. Cache miss - query database to find the Node ID that owns this API key
	actualNodeID, err := h.Store.GetNodeByKey(token)
	if err != nil || actualNodeID == "" {
		return false
	}

	// 3. Cache the valid token-to-node mapping for future requests
	h.nodeAuthCache.Store(token, actualNodeID)

	// 4. Validate that the token's owner matches the Node ID claiming the request
	return subtle.ConstantTimeCompare([]byte(actualNodeID), []byte(expectedNodeID)) == 1
}