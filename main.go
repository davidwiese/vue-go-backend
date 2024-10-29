package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/api"
	"github.com/davidwiese/fleet-tracker-backend/internal/config"
	"github.com/davidwiese/fleet-tracker-backend/internal/database"
	"github.com/davidwiese/fleet-tracker-backend/internal/models"
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

	// Initialize WebSocket hub
  hub := websocket.NewHub()
  go hub.Run()

  // Create API handler
  handler := api.NewHandler(db, hub.Broadcast)

  // Setup routes
  handler.SetupRoutes()

  // Setup WebSocket endpoint
  http.HandleFunc("/ws", hub.HandleWebSocket)

  // Start vehicle movement simulation
  go simulateVehicleMovement(db, hub.Broadcast)

  // Start server
  log.Printf("Server started on port %s", cfg.APIConfig.Port)
  log.Fatal(http.ListenAndServe(":"+cfg.APIConfig.Port, nil))
}

// Simulate vehicle movement by periodically updating their positions
func simulateVehicleMovement(db *database.DB, broadcast chan models.Vehicle) {
    r := rand.New(rand.NewSource(time.Now().UnixNano()))
    for {
        time.Sleep(5 * time.Second)
        rows, err := db.Query("SELECT id, name, status, latitude, longitude FROM vehicles WHERE status = 'Active'")
        if err != nil {
            log.Println("Error fetching vehicles:", err)
            continue
        }

        vehicles := []models.Vehicle{}
        for rows.Next() {
            var v models.Vehicle
            if err := rows.Scan(&v.ID, &v.Name, &v.Status, &v.Latitude, &v.Longitude); err != nil {
                log.Println("Error scanning vehicle:", err)
                continue
            }
            v.UpdatePosition((r.Float64()-0.5)*0.01, (r.Float64()-0.5)*0.01)
            vehicles = append(vehicles, v)
        }
        rows.Close()

        for _, v := range vehicles {
            _, err := db.Exec("UPDATE vehicles SET latitude = ?, longitude = ? WHERE id = ?", 
                v.Latitude, v.Longitude, v.ID)
            if err != nil {
                log.Println("Error updating vehicle:", err)
                continue
            }
            broadcast <- v
        }
    }
}