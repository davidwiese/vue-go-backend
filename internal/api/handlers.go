package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/davidwiese/fleet-tracker-backend/internal/database"
	"github.com/davidwiese/fleet-tracker-backend/internal/models"
	"github.com/davidwiese/fleet-tracker-backend/internal/onestepgps"
)

// Handler struct to hold dependencies
type Handler struct {
	DB              *database.DB
	BroadcastChannel chan models.Vehicle
	GPSClient        *onestepgps.Client
}

// NewHandler creates a new Handler instance
func NewHandler(db *database.DB, broadcastChannel chan models.Vehicle, gpsClient *onestepgps.Client) *Handler {
	return &Handler{
		DB:               db,
		BroadcastChannel: broadcastChannel,
		GPSClient:        gpsClient,
	}
}

// VehiclesHandler handles "/vehicles" endpoint
func (h *Handler) VehiclesHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        h.getVehicles(w, r)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// getVehicles returns all vehicles from OneStepGPS
func (h *Handler) getVehicles(w http.ResponseWriter, _ *http.Request) {
    vehicles, err := h.GPSClient.GetDevices()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(vehicles)
}

// VehicleHandler handles "/vehicles/{id}" endpoint
func (h *Handler) VehicleHandler(w http.ResponseWriter, r *http.Request) {
    // Extract the device_id from the URL path
    deviceID := strings.TrimPrefix(r.URL.Path, "/vehicles/")
    if deviceID == "" {
        http.Error(w, "Invalid vehicle ID", http.StatusBadRequest)
        return
    }

    switch r.Method {
    case http.MethodGet:
        h.getVehicle(w, r, deviceID)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// getVehicle returns a single vehicle from OneStepGPS
func (h *Handler) getVehicle(w http.ResponseWriter, _ *http.Request, deviceID string) {
    // Get all vehicles and find the requested one
    vehicles, err := h.GPSClient.GetDevices()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    for _, vehicle := range vehicles {
        if vehicle.DeviceID == deviceID {
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(vehicle)
            return
        }
    }

    http.Error(w, "Vehicle not found", http.StatusNotFound)
}

// debugHandler retrieves all vehicles (for development only)
func (h *Handler) debugHandler(w http.ResponseWriter, r *http.Request) {
    vehicles, err := h.GPSClient.GetDevices()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(vehicles)
}