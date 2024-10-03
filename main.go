package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"

	// Blank import for side effects - allows database/sql package to use the MySQL driver
	// to connect to MySQL databases, even though I never directly call any functions from
	// the package in my code
	_ "github.com/go-sql-driver/mysql"
)

// Define global db variable pointer to sql.DB struct (thread safe for concurrent use)
var db *sql.DB

// Define the websocket upgrader (upgrades HTTP requests to WebSocket connections)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Define the broadcast channel to send updates to all connected clients
var broadcastChannel = make(chan Vehicle)

// Define the clients map to keep track of connected clients
// The key is a pointer to a websocket.Conn, and the value is a bool
var clients = make(map[*websocket.Conn]bool)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get db connection string from environment variable
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is not set")
	}

	// Open the database connection
	// The "mysql" driver is used to connect to MySQL databases
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close() // Ensure the database connection is closed when the program exits

	// Test the database connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging database:", err)
	}

	// Goroutines
	// Similar to js async/await, but can run thousands of threads concurrently
	// managed by Go runtime, not the OS
	// Goroutines can run in parallel on multiple CPU cores

	// Start the message handler in a separate goroutine
	go handleMessages()

	// Start the vehicle movement simulation in a separate goroutine
	go simulateVehicleMovement()

	// Define HTTP routes w/ CORS middleware
	http.Handle("/vehicles", withCORS(http.HandlerFunc(vehiclesHandler)))
	http.Handle("/vehicles/", withCORS(http.HandlerFunc(vehicleHandler))) // for /vehicles/{id}

	// WebSocket endpoint
	http.HandleFunc("/ws", wsEndpoint)

	// DEBUGGING ENDPOINT FOR DEV ONLY
	http.HandleFunc("/debug", debugHandler)

	// Log/start the server on port 8080
	log.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}

// Handle websocket connections
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
func handleMessages() {
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
func simulateVehicleMovement() {
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

		vehicles := []Vehicle{}
		for rows.Next() {
			var v Vehicle
			if err := rows.Scan(&v.ID, &v.Name, &v.Status, &v.Latitude, &v.Longitude); err != nil {
				log.Println("Error scanning vehicle:", err)
				continue
			}
			// Randomly adjust latitude and longitude slightly
			v.Latitude += (r.Float64() - 0.5) * 0.01
			v.Longitude += (r.Float64() - 0.5) * 0.01
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

// Middleware to add CORS headers to HTTP responses
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the necessary headers
		// Allow all origins (for development purposes). In production, set this to your frontend's origin.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight OPTIONS requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// Debugging endpoint to retrieve all vehicles (for development only)
func debugHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT * FROM vehicles")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var vehicles []Vehicle
	for rows.Next() {
		var v Vehicle
		err := rows.Scan(&v.ID, &v.Name, &v.Status, &v.Latitude, &v.Longitude)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vehicles = append(vehicles, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vehicles)
}
