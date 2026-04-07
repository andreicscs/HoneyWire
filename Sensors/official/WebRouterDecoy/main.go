package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/honeywire/sdk-go"
)

var (
	port        = getEnv("HW_BIND_PORT", "8080")
	routerBrand = getEnv("HW_ROUTER_BRAND", "Netgear")
	hw          *sdk.Sensor
)

// The HTML template with %s placeholders for the router brand
const loginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%s Administration</title>
  <style>
    body { background: #f1f5f9; color: #111827; font-family: Inter, system-ui, sans-serif; }
    .card { max-width: 420px; margin: 6rem auto; padding: 2rem; background: white; box-shadow: 0 24px 80px rgba(15,23,42,0.08); border-radius: 0.5rem; }
    h1 { margin-bottom: 1.25rem; font-size: 1.75rem; letter-spacing: -.03em; }
    label { display: block; margin-top: 1rem; font-size: 0.9rem; color: #374151; }
    input { width: 100%%; margin-top: 0.5rem; padding: 0.9rem 1rem; border: 1px solid #d1d5db; border-radius: 0.25rem; box-sizing: border-box; }
    button { width: 100%%; margin-top: 1.75rem; padding: 0.95rem 1rem; background: #0f766e; color: white; border: none; border-radius: 0.25rem; font-weight: 700; cursor: pointer; }
    .footer { margin-top: 1.5rem; font-size: 0.85rem; color: #6b7280; }
  </style>
</head>
<body>
  <div class="card">
    <h1>%s Router</h1>
    <p>Sign in to the administration panel.</p>
    <form method="post" action="/login">
      <label>Username</label>
      <input name="username" type="text" autocomplete="username" value="admin" />
      <label>Password</label>
      <input name="password" type="password" autocomplete="current-password" />
      <button type="submit">Login</button>
    </form>
    <div class="footer">If credentials are invalid, please try again.</div>
  </div>
</body>
</html>`

func main() {
	// 1. Initialize SDK
	hw = sdk.NewSensor("web_honeypot")
	hw.Start()

	log.Printf("[*] HoneyWire Web Router Decoy | Brand: %s | Port: %s", routerBrand, port)

	// 2. Register Routes
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/login", handleLogin)

	// 3. Start the Server
	addr := fmt.Sprintf("0.0.0.0:%s", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[!] FATAL: Web server failed: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Only respond to GET requests on the root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Inject the brand name into the HTML
	fmt.Fprintf(w, loginHTML, routerBrand, routerBrand)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the submitted form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	userAgent := r.UserAgent()
	if userAgent == "" {
		userAgent = "Unknown"
	}

	// Extract the real IP (strip the ephemeral port)
	srcIP := r.RemoteAddr
	if ip, _, err := net.SplitHostPort(srcIP); err == nil {
		srcIP = ip
	}

	log.Printf("[!] Login attempt from %s (User: %s)", srcIP, username)

	// Dispatch the Event via the SDK
	hw.ReportEvent(
		"web_login_attempt",
		"critical",
		map[string]any{
			"user_agent":         userAgent,
			"attempted_username": username,
			"attempted_password": password,
		},
		"logged",
		srcIP,
		"Web Interface",
	)

	// Always return 401 Unauthorized to keep them guessing
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, "<h1>401 Unauthorized</h1><p>Invalid Username or Password.</p>")
}

func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return fallback
}