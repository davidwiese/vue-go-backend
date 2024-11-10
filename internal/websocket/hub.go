// hub.go manages real-time vehicle data broadcasting through WebSocket
// connections between the backend and frontend clients.

package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/models"
	"github.com/davidwiese/fleet-tracker-backend/internal/onestepgps"
	"github.com/gorilla/websocket"
)

// Hub coordinates WebSocket connections and vehicle data broadcasting.
// It maintains connected clients and handles real-time updates from OneStepGPS.
type Hub struct {
    clients map[*websocket.Conn]bool    // Active WebSocket connections
    Broadcast chan []models.Vehicle     // Channel for sending vehicle updates to all clients
    upgrader websocket.Upgrader         // WebSocket connection upgrader
    gpsClient *onestepgps.Client        // Client for fetching OneStepGPS data
    updateInterval time.Duration        // How often to poll for vehicle updates
}

// NewHub creates a new WebSocket hub with specified update frequency.
// Called in main.go during server initialization.
func NewHub(gpsClient *onestepgps.Client, updateInterval time.Duration) *Hub {
    return &Hub{
        clients:   make(map[*websocket.Conn]bool),
        Broadcast: make(chan []models.Vehicle),
        upgrader: websocket.Upgrader{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
            CheckOrigin: func(r *http.Request) bool {
                return true // Allow all origins for development
            },
        },
        gpsClient:      gpsClient,
        updateInterval: updateInterval,
    }
}

// Run starts the hub's main operations:
// 1. Polling OneStepGPS for vehicle updates
// 2. Broadcasting updates to all connected clients
// Started as a goroutine in main.go
func (h *Hub) Run() {
    // Start polling in separate goroutine
    go h.pollUpdates()

    // Main broadcast loop
    for vehicles := range h.Broadcast {
        // Send updates to all connected clients
        for client := range h.clients {
            err := client.WriteJSON(vehicles)
            if err != nil {
                log.Printf("WebSocket Write Error: %v", err)
                client.Close()
                delete(h.clients, client) // Remove disconnected client
            }
        }
    }
}

// pollUpdates periodically fetches vehicle data from OneStepGPS.
// Runs in background, pushing updates to the Broadcast channel.
func (h *Hub) pollUpdates() {
    ticker := time.NewTicker(h.updateInterval)
    defer ticker.Stop()

    for range ticker.C {
        vehicles, err := h.gpsClient.GetDevices()
        if err != nil {
            log.Printf("Error fetching vehicle updates: %v", err)
            continue // Skip this update on error
        }
        h.Broadcast <- vehicles // Send update to broadcast channel
    }
}

// HandleWebSocket manages individual WebSocket connections.
// Called when frontend (HomeView.vue) initiates WebSocket connection.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Upgrade HTTP connection to WebSocket
    conn, err := h.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket Upgrade Error:", err)
        return
    }

    // Register new client
    h.clients[conn] = true
    log.Println("Client connected")

    // Send initial vehicle data to new client
    vehicles, err := h.gpsClient.GetDevices()
    if err != nil {
        log.Printf("Error fetching initial vehicle data: %v", err)
    } else {
        err = conn.WriteJSON(vehicles)
        if err != nil {
            log.Printf("Error sending initial data: %v", err)
        }
    }

    // Cleanup on disconnect
    defer func() {
        conn.Close()
        delete(h.clients, conn)
        log.Println("Client disconnected")
    }()

    // Keep connection alive until error occurs
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            log.Printf("WebSocket Read Error: %v", err)
            break
        }
    }
}