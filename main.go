package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tanq16/nottif/internal/config"
	"github.com/tanq16/nottif/internal/server"
)

//go:embed all:frontend
var frontendFS embed.FS

func main() {
	// Ensure the data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Load application configuration from the data directory
	configPath := filepath.Join("data", "config.json")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create a new server instance
	srv, err := server.New(cfg, frontendFS)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start the cron scheduler in the background
	go srv.Scheduler.Start()
	log.Println("Cron scheduler started")

	// Start the HTTP server
	log.Println("Starting nottif server on :8080")
	if err := http.ListenAndServe(":8080", srv.Router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
