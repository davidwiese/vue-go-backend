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

	_ "github.com/go-sql-driver/mysql"
)

// Define global db variable (thread safe for concurrency)
var db *sql.DB

// Define the websocket upgrader
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

// Define the broadcast channel and clients map
var broadcastChannel = make(chan Vehicle)
var clients = make(map[*websocket.Conn]bool)

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Println("No .env file found, using environment variables")
    }
    
    dsn := os.Getenv("DB_DSN")
		if dsn == "" {
    log.Fatal("DB_DSN environment variable is not set")
		}

    db, err = sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Test the database connection
    err = db.Ping()
    if err != nil {
        log.Fatal(err)
    }

    // Start the message handler
    go handleMessages()

		// Start the vehicle movement simulation
    go simulateVehicleMovement()

    // Define HTTP routes w/ CORS middleware
    http.Handle("/vehicles", withCORS(http.HandlerFunc(vehiclesHandler)))
		http.Handle("/vehicles/", withCORS(http.HandlerFunc(vehicleHandler))) // for /vehicles/{id}

		// WebSocket endpoint
    http.HandleFunc("/ws", wsEndpoint)

		// DEBUGGING ENDPOINT FOR DEV ONLY
		http.HandleFunc("/debug", debugHandler)

    // Start the server
    log.Println("Server started on port 8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
		
}

// WebSocket handler
func wsEndpoint(w http.ResponseWriter, r *http.Request) {
    upgrader.CheckOrigin = func(r *http.Request) bool { return true }

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket Upgrade Error:", err)
        return
    }

    // Register the client
    clients[conn] = true
    log.Println("Client connected")

    defer func() {
        conn.Close()
        delete(clients, conn)
        log.Println("Client disconnected")
    }()

    // Keep the connection open
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            log.Printf("WebSocket Read Error: %v", err)
            delete(clients, conn)
            break
        }
    }
}

// Broadcast messages to clients
func handleMessages() {
    for {
        // Grab the next message from the broadcast channel
        vehicleUpdate := <-broadcastChannel

        // Send it out to every client that is currently connected
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

// Simulate vehicle movement
func simulateVehicleMovement() {
    // Create a new rand.Rand instance with its own seed
		// Local random number generator is thread safe
    r := rand.New(rand.NewSource(time.Now().UnixNano()))
    for {
        time.Sleep(5 * time.Second)
        // Fetch vehicles from the database
        rows, err := db.Query("SELECT id, name, status, latitude, longitude FROM vehicles")
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
            _, err := db.Exec("UPDATE vehicles SET latitude = ?, longitude = ? WHERE id = ?", v.Latitude, v.Longitude, v.ID)
            if err != nil {
                log.Println("Error updating vehicle:", err)
                continue
            }
            // Send updated vehicle to broadcast channel
            broadcastChannel <- v
        }
    }
}

// CORS middleware function
func withCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set the necessary headers
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

// For debugging/dev only
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
