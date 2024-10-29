package simulator

import (
	"log"
	"math/rand"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/database"
	"github.com/davidwiese/fleet-tracker-backend/internal/models"
)

type Simulator struct {
    db        *database.DB
    broadcast chan models.Vehicle
    interval  time.Duration
}

// NewSimulator creates a new vehicle simulator
func NewSimulator(db *database.DB, broadcast chan models.Vehicle) *Simulator {
    return &Simulator{
        db:        db,
        broadcast: broadcast,
        interval:  5 * time.Second,
    }
}

// Start begins the vehicle movement simulation
func (s *Simulator) Start() {
    r := rand.New(rand.NewSource(time.Now().UnixNano()))
    
    for {
        time.Sleep(s.interval)
        
        // Fetch active vehicles
        rows, err := s.db.Query("SELECT id, name, status, latitude, longitude FROM vehicles WHERE status = 'Active'")
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

        // Update and broadcast
        for _, v := range vehicles {
            _, err := s.db.Exec("UPDATE vehicles SET latitude = ?, longitude = ? WHERE id = ?",
                v.Latitude, v.Longitude, v.ID)
            if err != nil {
                log.Println("Error updating vehicle:", err)
                continue
            }
            s.broadcast <- v
        }
    }
}