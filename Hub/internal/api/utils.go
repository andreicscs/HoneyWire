package api

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
)

// --- Helpers ---

func RespondError(w http.ResponseWriter, message string, code int) {
	SendJSON(w, code, map[string]string{"error": message})
}

func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[ERROR] SendJSON encode error: %v\n", err)
	}
}

func GetRealIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return strings.Split(ip, ",")[0]
		}
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			return strings.Split(ip, ",")[0]
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
