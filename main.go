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
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// Define the websocket upgrader (upgrades HTTP requests to WebSocket connections)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Define the clients map to keep track of connected clients
// The key is a pointer to a websocket.Conn, and the value is a bool
var clients = make(map[*websocket.Conn]bool)

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

	// Create broadcast channel
	broadcastChannel := make(chan models.Vehicle)

	// Create API handler
	handler := api.NewHandler(db, broadcastChannel)

	// Setup routes
	handler.SetupRoutes()

	// Setup websocket endpoint (we'll move this to its own package next)
	http.HandleFunc("/ws", wsEndpoint)

	// Goroutines
	// Similar to js async/await, but can run thousands of threads concurrently
	// managed by Go runtime, not the OS
	// Goroutines can run in parallel on multiple CPU cores

	// Start the message handler in a separate goroutine
	go handleMessages(broadcastChannel)

	// Start the vehicle movement simulation in a separate goroutine
	go simulateVehicleMovement(db, broadcastChannel)

	// Log/start the server on port 8080
	log.Println("Server started on port", cfg.APIConfig.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.APIConfig.Port, nil))
}

// Websocket functions moved to package level for now
func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	// Allow all cross-origin requests (caution in production)
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// Upgrade the HTTP request to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket Upgrade Error:", err)
		return
	}

	// Register the client
	clients[conn] = true
	log.Println("Client connected")

	// Ensure the connection is closed and the client is removed when the function exits
	defer func() {
		conn.Close()
		delete(clients, conn)
		log.Println("Client disconnected")
	}()

	// Keep the connection open
	for {
		// ReadMessage is used here just to detect when the client disconnects
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket Read Error: %v", err)
			delete(clients, conn)
			break
		}
	}
}

// Broadcast messages to all connected clients
// Similar to Socket.io in a Node.js backend
func handleMessages(broadcastChannel chan models.Vehicle) {
	// Outer for loop runs indefinitely, similar to an event listener in Node.js
	for {
		// Waits for and receives vehicle updates from the broadcast channel
		vehicleUpdate := <-broadcastChannel

		// Send the update to every connected client
		for client := range clients {
			err := client.WriteJSON(vehicleUpdate)
			if err != nil {
				log.Printf("WebSocket Write Error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

// Simulate vehicle movement by periodically updating their positions
func simulateVehicleMovement(db *database.DB, broadcastChannel chan models.Vehicle) {
	// Create a new rand.Rand instance with its own seed
	// Local random number generator is thread safe
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		time.Sleep(5 * time.Second)
		// Fetch only active vehicles from the database
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
			// Randomly adjust latitude and longitude slightly
			v.UpdatePosition((r.Float64()-0.5)*0.01, (r.Float64()-0.5)*0.01)
			vehicles = append(vehicles, v)
		}
		rows.Close()

		// Update vehicle positions in the database and broadcast updates
		for _, v := range vehicles {
			// _ ignores the result of db.Exec (number of affected rows)
			_, err := db.Exec("UPDATE vehicles SET latitude = ?, longitude = ? WHERE id = ?", v.Latitude, v.Longitude, v.ID)
			if err != nil {
				log.Println("Error updating vehicle:", err)
				continue
			}
			// Send the updated vehicle to the broadcast channel
			broadcastChannel <- v
		}
	}
}