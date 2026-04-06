package main

import (
	"log"
	"net/http"

	"github.com/honeywire/hub/internal/api"
	"github.com/honeywire/hub/internal/auth"
	"github.com/honeywire/hub/internal/config"
	"github.com/honeywire/hub/internal/store"
)

// We will eventually load this from internal/config
const Version = "1.0.0"

func main() {
	log.Println("Starting HoneyWire Go Hub initialization...")
	
	cfg := config.Load()

	dbStore, err := store.NewStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbStore.DB.Close()

	sessionStore := auth.NewSessionStore()

	r := api.SetupRouter(cfg, dbStore, sessionStore)

	// Start the Server
	port := ":" + cfg.Port
	log.Printf("Server listening on port %s\n", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}