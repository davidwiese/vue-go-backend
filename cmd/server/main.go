package main

import (
	"log"
	"net/http"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/api"
	"github.com/davidwiese/fleet-tracker-backend/internal/config"
	"github.com/davidwiese/fleet-tracker-backend/internal/database"
	"github.com/davidwiese/fleet-tracker-backend/internal/onestepgps"
	"github.com/davidwiese/fleet-tracker-backend/internal/websocket"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using environment variables")
    }

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Initialize database
	db, err := database.NewDB(cfg.DBConfig.DSN)
	if err != nil {
		log.Fatal("Error initializing database:", err)
	}
	defer db.Close()

	// Create tables
  if err := db.CreateTableIfNotExists(); err != nil {
		log.Fatal("Error creating tables:", err)
	}

	// Initialize OneStepGPS client
  gpsClient := onestepgps.NewClient(cfg.APIConfig.GPSApiKey)

	// Initialize WebSocket hub with GPS client and update interval
  hub := websocket.NewHub(gpsClient, 5*time.Second)
  go hub.Run()

  // Create API handler
  handler := api.NewHandler(db, hub.Broadcast, gpsClient)

  // Setup routes
  handler.SetupRoutes()

  // Setup WebSocket endpoint
  http.HandleFunc("/ws", hub.HandleWebSocket)

  // Start server
  log.Printf("Server started on port %s", cfg.APIConfig.Port)
  log.Fatal(http.ListenAndServe(":"+cfg.APIConfig.Port, nil))
}