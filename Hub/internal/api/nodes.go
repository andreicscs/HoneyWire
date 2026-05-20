package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// CreateNode handles UI-first node creation
// Route: POST /api/v1/nodes
func (h *Handler) CreateNode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Alias string   `json:"alias"`
		Tags  []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Alias == "" {
		RespondError(w, "Alias is required", http.StatusBadRequest)
		return
	}

	tagsJSON, _ := json.Marshal(req.Tags)
	if string(tagsJSON) == "null" {
		tagsJSON = []byte("[]")
	}

	nodeID, apiKey, err := h.Store.CreateNode(req.Alias, string(tagsJSON))
	if err != nil {
		RespondError(w, "Failed to create node in database", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusCreated, map[string]interface{}{
		"node_id": nodeID,
		"apiKey":  apiKey,
		"alias":   req.Alias,
	})
}

// UpdateNode handles UI requests to edit a Node's metadata
// Route: PATCH /api/v1/nodes/{id}
func (h *Handler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")

	var req struct {
		Alias     string   `json:"alias"`
		Tags      []string `json:"tags"`
		PublicIP  *string  `json:"publicIp"`
		PrivateIP *string  `json:"privateIp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.Alias == "" {
		RespondError(w, "Alias cannot be empty", http.StatusBadRequest)
		return
	}

	tagsJSON, _ := json.Marshal(req.Tags)
	if string(tagsJSON) == "null" {
		tagsJSON = []byte("[]")
	}

	if err := h.Store.UpdateNodeMeta(nodeID, req.Alias, string(tagsJSON), req.PublicIP, req.PrivateIP); err != nil {
		RespondError(w, "Failed to update node", http.StatusInternalServerError)
		return
	}

	// Optional: Broadcast WS to UI to instantly update the Fleet grid
	h.broadcastWS("UPDATE_NODE", map[string]string{"node_id": nodeID})

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// GetNodes handles GET /api/v1/nodes
func (h *Handler) GetNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.Store.GetNodes()
	if err != nil {
		log.Printf("[ERROR] GetNodes failed: %v\n", err)
		RespondError(w, "Failed to fetch fleet nodes", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, nodes)
}

// GetNodeDetails handles GET /api/v1/nodes/{id}
func (h *Handler) GetNodeDetails(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")

	node, err := h.Store.GetNodeDetails(nodeID)
	if err != nil {
		RespondError(w, "Node not found", http.StatusNotFound)
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"id":               node.ID,
		"alias":            node.Alias,
		"apiKey":           node.APIKey,
		"activeRevision":   node.ActiveRevision,
		"desiredRevision":  node.DesiredRevision,
		"publicIp":         node.PublicIP,
		"privateIp":        node.PrivateIP,
		"tags":             node.Tags,
		"hasPendingConfig": node.HasPendingConfig,
		"lastHeartbeat":    node.LastHeartbeat,
		"status":           node.Status,
		"installedSensors": node.InstalledSensors,
	})
}

// AddNodeSensor handles POST /api/v1/nodes/{id}/sensors
func (h *Handler) AddNodeSensor(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")

	var req struct {
		SensorID     string                 `json:"sensor_id"`
		CustomName   string                 `json:"custom_name"`
		ConfigValues map[string]interface{} `json:"config_values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.SensorID == "" || req.CustomName == "" {
		RespondError(w, "sensor_id and custom_name are required", http.StatusBadRequest)
		return
	}

	err := h.Store.AddSensorToNode(nodeID, req.SensorID, req.CustomName, req.ConfigValues)
	if err != nil {
		RespondError(w, "Failed to add sensor. It may already be installed on this node.", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Sensor added, node pending sync"})
}

// EditNodeSensor handles PUT /api/v1/nodes/{id}/sensors/{sensor_id}
func (h *Handler) EditNodeSensor(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")
	sensorID := chi.URLParam(r, "sensor_id")

	var req struct {
		CustomName   string                 `json:"custom_name"`
		ConfigValues map[string]interface{} `json:"config_values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if err := h.Store.UpdateNodeSensor(nodeID, sensorID, req.CustomName, req.ConfigValues); err != nil {
		RespondError(w, "Failed to update sensor", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// DeleteNodeSensor handles DELETE /api/v1/nodes/{id}/sensors/{sensor_id}
func (h *Handler) DeleteNodeSensor(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")
	sensorID := chi.URLParam(r, "sensor_id")

	if err := h.Store.RemoveNodeSensor(nodeID, sensorID); err != nil {
		RespondError(w, "Failed to remove sensor", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// generateRevisionHash creates a random 8-char string for tracking config state
func generateRevisionHash() string {
	b := make([]byte, 4)
	rand.Read(b)
	return "rev_" + hex.EncodeToString(b)
}

// DeleteNode handles DELETE /api/v1/nodes/{id}
func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")

	if err := h.Store.DeleteNode(nodeID); err != nil {
		RespondError(w, "Failed to delete node", http.StatusInternalServerError)
		return
	}

	// Broadcast WS to UI to instantly remove the row from the Fleet grid
	h.broadcastWS("DELETE_NODE", map[string]string{"node_id": nodeID})

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}


// GetCurrentNode handles GET /api/v1/nodes/me
// Used by wizard agents authenticated via Bearer token
func (h *Handler) GetCurrentNode(w http.ResponseWriter, r *http.Request) {

	// Authenticate via Bearer token
	nodeID, err := h.authenticateNodeRequest(r)

	if err != nil {
		RespondError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Fetch full node details
	node, err := h.Store.GetNodeDetails(nodeID)

	if err != nil {
		RespondError(w, "Node not found", http.StatusNotFound)
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"id":               node.ID,
		"alias":            node.Alias,
		"activeRevision":   node.ActiveRevision,
		"desiredRevision":  node.DesiredRevision,
		"publicIp":         node.PublicIP,
		"privateIp":        node.PrivateIP,
		"tags":             node.Tags,
		"hasPendingConfig": node.HasPendingConfig,
		"lastHeartbeat":    node.LastHeartbeat,
		"status":           node.Status,
		"installedSensors": node.InstalledSensors,
	})
}