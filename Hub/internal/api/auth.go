package api

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/honeywire/hub/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

// --- Brute-Force Protection State ---
type loginState struct {
	attempts    int
	lockedUntil time.Time
}

var (
	authTracker = make(map[string]*loginState)
	authMutex   sync.Mutex
)

// Background routine to prevent memory leaks from abandoned IPs
func (h *Handler) cleanupAuthTracker() {
	for {
		time.Sleep(5 * time.Minute)
		authMutex.Lock()
		now := time.Now()
		for ip, state := range authTracker {
			if now.After(state.lockedUntil) {
				delete(authTracker, ip)
			}
		}
		authMutex.Unlock()
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ip := h.getRealIP(r)

	// Rate Limiter Pre-Check
	authMutex.Lock()
	if state, exists := authTracker[ip]; exists {
		if state.attempts >= 10 {
			if time.Now().Before(state.lockedUntil) {
				authMutex.Unlock()
				http.Error(w, "Too many failed attempts. Try again later.", http.StatusTooManyRequests)
				return
			}
			// Lockout expired, wipe the slate clean
			delete(authTracker, ip)
		}
	}
	authMutex.Unlock()

	// Parse Request
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Evaluate Authorization
	var isAuthorized bool

	if h.Cfg.DashboardPassword != "" {
		// Layer A: Infrastructure Override (.env file)
		isAuthorized = subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.Cfg.DashboardPassword)) == 1
	} else {
		// Layer B: Runtime Database Hash (Setup UI)
		var dbHash string
		err := h.Store.DB.QueryRow("SELECT value FROM config WHERE key='admin_hash'").Scan(&dbHash)
		if err == nil {
			err = bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(req.Password))
			isAuthorized = (err == nil)
		}
	}

	if isAuthorized {
		// Clear failed attempts for this IP on successful login
		authMutex.Lock()
		delete(authTracker, ip)
		authMutex.Unlock()

		token, err := h.SessionStore.Create()
		if err != nil {
			http.Error(w, "Session creation failed", http.StatusInternalServerError)
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
	authMutex.Lock()
	if _, exists := authTracker[ip]; !exists {
		authTracker[ip] = &loginState{}
	}
	authTracker[ip].attempts++
	
	if authTracker[ip].attempts >= 10 {
		authTracker[ip].lockedUntil = time.Now().Add(15 * time.Minute)
		log.Printf("[!] AUDIT: IP %s locked out of dashboard for 15 minutes due to brute-force", ip)
	}
	authMutex.Unlock()

	http.Error(w, "Invalid Password", http.StatusUnauthorized)
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