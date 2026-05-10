package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)


// GenerateToken creates a one-time pairing token for the wizard
// Protected by UI authentication (dashboard session cookie)
func (h *Handler) GenerateToken(w http.ResponseWriter, r *http.Request) {
	// Generate a random 32-character hex token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		RespondError(w, "Token generation failed", http.StatusInternalServerError)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Token expires in 15 minutes
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	if err := h.Store.InsertPairingToken(token, expiresAt, now); err != nil {
		RespondError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"expires_in": 900, // 15 minutes in seconds
	})
}

// WizardLink handles node provisioning from the wizard
// Public endpoint - creates a new node with one-time token exchange
func (h *Handler) WizardLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
		Alias string `json:"alias"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		RespondError(w, "Token is required", http.StatusBadRequest)
		return
	}

	// Validate token exists and is not expired
	isValid, err := h.Store.ValidatePairingToken(req.Token)
	if err != nil || !isValid {
		RespondError(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	// Generate node credentials
	nodeID := uuid.New().String()
	nodeKeyBytes := make([]byte, 32)
	if _, err := rand.Read(nodeKeyBytes); err != nil {
		RespondError(w, "Node key generation failed", http.StatusInternalServerError)
		return
	}
	nodeKey := hex.EncodeToString(nodeKeyBytes)

	if req.Alias == "" {
		req.Alias = "node-" + nodeID[:8] // Default alias if none provided
	}

	// Get client IP address
	clientIP := h.getRealIP(r)

	// Insert new node
	now := time.Now().Format(time.RFC3339)
	if err := h.Store.CreateNode(nodeID, req.Alias, nodeKey, clientIP, now); err != nil {
		RespondError(w, "Failed to create node", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{
		"node_id":   nodeID,
		"node_key":  nodeKey,
		"alias":     req.Alias,
		"timestamp": now,
	})
}
