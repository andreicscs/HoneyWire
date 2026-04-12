package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	var err error
	hw, err = sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}

	if hw.TestMode {
		if hw.RunTestMode() { os.Exit(0) }
		os.Exit(1)
	}

	if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}
	defer hw.Stop()

	log.Printf("[*] HoneyWire Web Router Decoy | Brand: %s | Port: %s", routerBrand, port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/login", handleLogin)

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
		Handler: mux,
	}

	// Run server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[!] FATAL: Web server failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("[*] Shutting down web decoy...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[-] Shutdown error: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, loginHTML, routerBrand, routerBrand)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	userAgent := r.UserAgent()
	if userAgent == "" { userAgent = "Unknown" }

	srcIP := r.RemoteAddr
	if ip, _, err := net.SplitHostPort(srcIP); err == nil { srcIP = ip }

	log.Printf("[!] Login attempt from %s (User: %s)", srcIP, username)

	hw.ReportEvent(
		"critical",
		"web_login_attempt",
		srcIP,
		"Web Interface",
		map[string]any{
			"user_agent":         userAgent,
			"attempted_username": username,
			"attempted_password": password,
			"action_taken":       "logged", 
		},
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, "<h1>401 Unauthorized</h1><p>Invalid Username or Password.</p>")
}

func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists { return val }
	return fallback
}