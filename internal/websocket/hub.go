package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/models"
	"github.com/davidwiese/fleet-tracker-backend/internal/onestepgps"
	"github.com/gorilla/websocket"
)

type Hub struct {
    // Registered clients
    clients map[*websocket.Conn]bool

    // Change channel type to accept array of vehicles
    Broadcast chan []models.Vehicle

    // Upgrader for WebSocket connections
    upgrader websocket.Upgrader

    // GPS client for updates
    gpsClient *onestepgps.Client

    // Update interval
    updateInterval time.Duration
}

func NewHub(gpsClient *onestepgps.Client, updateInterval time.Duration) *Hub {
    return &Hub{
        clients:   make(map[*websocket.Conn]bool),
        Broadcast: make(chan []models.Vehicle),
        upgrader: websocket.Upgrader{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
            CheckOrigin: func(r *http.Request) bool {
                return true
            },
        },
        gpsClient:      gpsClient,
        updateInterval: updateInterval,
    }
}

// Run starts the hub and begins polling for updates
func (h *Hub) Run() {
    // Start polling for updates
    go h.pollUpdates()

    // Handle broadcasting messages to clients
    for vehicles := range h.Broadcast {
        for client := range h.clients {
            err := client.WriteJSON(vehicles)
            if err != nil {
                log.Printf("WebSocket Write Error: %v", err)
                client.Close()
                delete(h.clients, client)
            }
        }
    }
}

// pollUpdates periodically fetches updates from the GPS API
func (h *Hub) pollUpdates() {
    ticker := time.NewTicker(h.updateInterval)
    defer ticker.Stop()

    for range ticker.C {
        vehicles, err := h.gpsClient.GetDevices()
        if err != nil {
            log.Printf("Error fetching vehicle updates: %v", err)
            continue
        }
        h.Broadcast <- vehicles
    }
}

// HandleWebSocket handles incoming WebSocket connections
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := h.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket Upgrade Error:", err)
        return
    }

    h.clients[conn] = true
    log.Println("Client connected")

    // Send initial vehicle data
    vehicles, err := h.gpsClient.GetDevices()
    if err != nil {
        log.Printf("Error fetching initial vehicle data: %v", err)
    } else {
        err = conn.WriteJSON(vehicles)
        if err != nil {
            log.Printf("Error sending initial data: %v", err)
        }
    }

    defer func() {
        conn.Close()
        delete(h.clients, conn)
        log.Println("Client disconnected")
    }()

    // Keep connection alive
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            log.Printf("WebSocket Read Error: %v", err)
            break
        }
    }
}