package websocket

import (
	"log"
	"net/http"

	"github.com/davidwiese/fleet-tracker-backend/internal/models"
	"github.com/gorilla/websocket"
)

type Hub struct {
    // Registered clients
    clients map[*websocket.Conn]bool

    // Broadcaster channel - now exported
    Broadcast chan models.Vehicle

    // Upgrader for WebSocket connections
    upgrader websocket.Upgrader
}

func NewHub() *Hub {
    return &Hub{
        clients:   make(map[*websocket.Conn]bool),
        Broadcast: make(chan models.Vehicle), // exported field
        upgrader: websocket.Upgrader{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
            CheckOrigin: func(r *http.Request) bool {
                return true
            },
        },
    }
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := h.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket Upgrade Error:", err)
        return
    }

    h.clients[conn] = true
    log.Println("Client connected")

    defer func() {
        conn.Close()
        delete(h.clients, conn)
        log.Println("Client disconnected")
    }()

    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            log.Printf("WebSocket Read Error: %v", err)
            break
        }
    }
}

func (h *Hub) Run() {
    for {
        select {
        case vehicle := <-h.Broadcast:
            for client := range h.clients {
                err := client.WriteJSON(vehicle)
                if err != nil {
                    log.Printf("WebSocket Write Error: %v", err)
                    client.Close()
                    delete(h.clients, client)
                }
            }
        }
    }
}