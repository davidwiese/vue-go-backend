// main.go is the entry point for the application backend.
// It initializes configuration, database, OneStepGPS client, WebSocket hub,
// and sets up HTTP routes for serving the frontend.

package main

import (
	"fmt"
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
	// Load environment variables from .env file for local development
	// In production, these variables are set in AWS Elastic Beanstalk
	if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using environment variables")
    }

	// Load application configuration from environment variables
	// See config/config.go for all available configuration options
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Initialize MySQL database connection
	// Frontend uses this database to store user preferences for vehicle display
	db, err := database.NewDB(cfg.DBConfig.DSN)
	if err != nil {
		log.Fatal("Error initializing database:", err)
	}
	defer db.Close() // Ensure database connection is closed when application exits

	// Create necessary database tables if they don't exist
	// Creates tables: vehicles (for cache), user_preferences (for frontend settings)
	if err := db.CreateTableIfNotExists(); err != nil {
		log.Fatal("Error creating tables:", err)
	}

	// Initialize OneStepGPS API client
	// This client is used to fetch real-time vehicle data
	// Used by WebSocket hub to broadcast updates to connected clients
	gpsClient := onestepgps.NewClient(cfg.APIConfig.GPSApiKey)

	// Initialize WebSocket hub for real-time updates
	// Frontend connects to this in HomeView.vue via initWebSocket()
	// Broadcasts vehicle updates every 5 seconds to all connected clients
	hub := websocket.NewHub(gpsClient, 5*time.Second)
	go hub.Run() // Start the hub in a separate goroutine

	// Create main API handler with all dependencies
	// This handler manages all HTTP endpoints used by the frontend
	handler := api.NewHandler(
		db,
		hub.Broadcast,
		gpsClient,
		api.HandlerConfig{
			OneStepGPSAPIKey: cfg.APIConfig.GPSApiKey,
			BaseURL:          "https://track.onestepgps.com/v3/api/public",
		},
	)

	// Setup API routes
	// These routes handle:
	// - Vehicle data (/vehicles) used in VehicleList.vue
	// - User preferences (/preferences) used in VehiclePreferences.vue
	// - Report generation (/report/generate) used in ReportDialog.vue
	fmt.Println("main.go: Setting up routes...")
	handler.SetupRoutes()
	fmt.Println("main.go: Routes setup completed")

	// Setup WebSocket endpoint
	// Frontend connects to this in HomeView.vue for real-time vehicle updates
	http.HandleFunc("/ws", hub.HandleWebSocket)

	// Start HTTP server
	// Serves both REST API endpoints and WebSocket connections
	log.Printf("Server started on port %s", cfg.APIConfig.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.APIConfig.Port, nil))
}