package api

import (
	"encoding/json"
	"net/http"
	"strings"

	composesvc "github.com/honeywire/hub/internal/services/compose"
)

type ComposeHandler struct {
	service *composesvc.Service
}

func NewComposeHandler(svc *composesvc.Service) *ComposeHandler {
	return &ComposeHandler{service: svc}
}

// GetNodeCompose generates the official docker-compose.yml for a specific agent
// Authentication: Bearer <API_KEY>
func (h *ComposeHandler) GetNodeCompose(w http.ResponseWriter, r *http.Request) {

	token := r.Header.Get("X-Api-Key")
	if token == "" {
		token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}

	hostFallback := "https://" + r.Host
	yamlData, err := h.service.GetNodeCompose(token, hostFallback)
	if err != nil {
		if err.Error() == "unauthorized" {
			RespondError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		RespondError(w, "Failed to generate compose file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	// nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
	// codeql[go/xss] Writing safe JSON/YAML API response.
	w.Write(yamlData)
}

func (h *ComposeHandler) GenerateCompose(w http.ResponseWriter, r *http.Request) {
	var req composesvc.PreviewRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "Invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	yamlData, err := h.service.GeneratePreviewCompose(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	// nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
	// codeql[go/xss] Writing safe JSON/YAML API response.
	w.Write(yamlData)
}
