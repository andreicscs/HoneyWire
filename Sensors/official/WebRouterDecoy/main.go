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
	port        string
	routerBrand string
	hw          *sdk.Sensor
)

// The HTML template with %[1]s for the brand and %[2]s for the error block display
const loginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%[1]s Router Login</title>

<style>
* {
    box-sizing: border-box;
}

body {
    margin: 0;
    font-family: Arial, Helvetica, sans-serif;
    background: #dfe3e8;
    color: #333;
}

.header {
    height: 64px;
    background: linear-gradient(to bottom, #3b6ea5, #2c5d90);
    border-bottom: 1px solid #21486f;
    display: flex;
    align-items: center;
    padding: 0 24px;
    color: white;
}

.brand {
    font-size: 24px;
    font-weight: bold;
    letter-spacing: 0.5px;
}

.container {
    width: 100%%;
    max-width: 520px;
    margin: 80px auto;
}

.panel {
    background: white;
    border: 1px solid #b7bec8;
}

.panel-header {
    background: #f2f4f7;
    border-bottom: 1px solid #d3d8de;
    padding: 12px 16px;
    font-weight: bold;
    color: #444;
}

.panel-body {
    padding: 24px;
}

.info {
    margin-bottom: 20px;
    font-size: 13px;
    color: #666;
    line-height: 1.5;
}

.error {
    display: %[2]s;
    background: #fff2f2;
    border: 1px solid #d77;
    color: #a00000;
    padding: 10px;
    margin-bottom: 18px;
    font-size: 13px;
}

.form-row {
    margin-bottom: 16px;
}

label {
    display: block;
    margin-bottom: 6px;
    font-size: 13px;
    font-weight: bold;
}

input {
    width: 100%%;
    height: 34px;
    padding: 0 10px;
    border: 1px solid #aeb6bf;
    font-size: 13px;
}

input:focus {
    outline: none;
    border-color: #3b6ea5;
}

.actions {
    margin-top: 20px;
}

button {
    min-width: 100px;
    height: 34px;
    border: 1px solid #1f4d79;
    background: linear-gradient(to bottom, #4a7ab0, #2f5f95);
    color: white;
    font-weight: bold;
    cursor: pointer;
}

button:hover {
    background: linear-gradient(to bottom, #5385bc, #35689f);
}

.footer {
    margin-top: 14px;
    padding: 12px;
    background: #f7f7f7;
    border: 1px solid #cfd4da;
    font-size: 12px;
    color: #666;
    line-height: 1.6;
}

.footer-row {
    display: flex;
    justify-content: space-between;
}

@media (max-width: 600px) {
    .container {
        margin: 20px;
    }
}
</style>
</head>

<body>

<div class="header">
    <div class="brand">%[1]s</div>
</div>

<div class="container">

    <div class="panel">

        <div class="panel-header">
            Router Login
        </div>

        <div class="panel-body">

            <div class="info">
                Enter administrator credentials to access the device management interface.
            </div>

            <div class="error">
                Login failed. Incorrect username or password.
            </div>

            <form method="post" action="/login">

                <div class="form-row">
                    <label for="username">Username</label>
                    <input
                        id="username"
                        name="username"
                        type="text"
                        autocomplete="username"
                        required
                    >
                </div>

                <div class="form-row">
                    <label for="password">Password</label>
                    <input
                        id="password"
                        name="password"
                        type="password"
                        autocomplete="current-password"
                        required
                    >
                </div>

                <div class="actions">
                    <button type="submit">Login</button>
                </div>

            </form>

        </div>
    </div>

    <div class="footer">
        <div class="footer-row">
            <span>Model</span>
            <span>RV340</span>
        </div>

        <div class="footer-row">
            <span>Firmware Version</span>
            <span>1.0.03.20</span>
        </div>

        <div class="footer-row">
            <span>Hardware Version</span>
            <span>1.0</span>
        </div>
    </div>

</div>

</body>
</html>`

func main() {
	var err error
	hw, err = sdk.NewSensor()
	if err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}

	hw.SetTestPayload(
		"web_login_attempt",
		"Wizard Firedrill",
		"Web Interface",
		sdk.EventDetails{
			{"test_message", "Wizard triggered a synthetic event firedrill."},
			{"user_agent", "HoneyWire-Firedrill/1.0"},
			{"attempted_username", "admin"},
			{"attempted_password", "password123"},
			{"action_taken", "logged"},
		},
	)

	// Load environment variables AFTER the SDK loads the .env file
	port = getEnv("HW_BIND_PORT", "8080")
	routerBrand = getEnv("HW_ROUTER_BRAND", "Enterprise")

	if hw.TestMode {
		if hw.RunTestMode() {
			os.Exit(0)
		}
		os.Exit(1)
	}

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
    log.Printf("[*] HoneyWire Web Router Decoy | Brand: %s | Port: %s", routerBrand, port)
    
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()


    if err := hw.Start(); err != nil {
		log.Fatalf("[!] FATAL: %v", err)
	}
	defer hw.Stop()

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
	// %[1]s = Brand, %[2]s = 'none' (hides the error banner initially)

	// codeql[go/xss] Honeypot decoy rendering intentional raw HTML.
	// nosemgrep: go.lang.security.audit.xss.no-fprintf-to-responsewriter.no-fprintf-to-responsewriter
	fmt.Fprintf(w, loginHTML, routerBrand, "none")
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
	if userAgent == "" {
		userAgent = "Unknown"
	}

	srcIP := r.RemoteAddr
	if ip, _, err := net.SplitHostPort(srcIP); err == nil {
		srcIP = ip
	}

	log.Printf("[!] Login attempt from %s (User: %s)", srcIP, username)

	hw.ReportEvent(
		"web_login_attempt",
		srcIP,
		"Web Interface",
		sdk.EventDetails{
			{"user_agent", userAgent},
			{"attempted_username", username},
			{"attempted_password", password},
			{"action_taken", "logged"},
		},
	)

	// Tarpit delay: Add an artificial backend processing delay to slow down automated scripts
	time.Sleep(1500 * time.Millisecond)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	// Re-render the exact same login page, but cleanly display the red error block
	// codeql[go/xss] Honeypot decoy rendering intentional raw HTML.
	// nosemgrep: go.lang.security.audit.xss.no-fprintf-to-responsewriter.no-fprintf-to-responsewriter
	fmt.Fprintf(w, loginHTML, routerBrand, "block")
}

func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return fallback
}
