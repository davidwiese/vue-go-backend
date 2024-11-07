package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/davidwiese/fleet-tracker-backend/internal/database"
	"github.com/davidwiese/fleet-tracker-backend/internal/models"
	"github.com/davidwiese/fleet-tracker-backend/internal/onestepgps"
)

// Handler struct to hold dependencies
type Handler struct {
    DB               *database.DB
    BroadcastChannel chan []models.Vehicle  // Update this line
    GPSClient        *onestepgps.Client
}

// NewHandler creates a new Handler instance
func NewHandler(db *database.DB, broadcastChannel chan []models.Vehicle, gpsClient *onestepgps.Client) *Handler {
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

// getAllPreferences returns all preferences
func (h *Handler) getAllPreferences(w http.ResponseWriter, r *http.Request) {
    // Get client_id from query parameter
    clientID := r.URL.Query().Get("client_id")
    if clientID == "" {
        clientID = "default"
    }

    // Update your DB function to accept clientID
    preferences, err := h.DB.GetAllPreferencesForClient(clientID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if preferences == nil {
        preferences = []models.UserPreference{}
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(preferences)
}

func (h *Handler) getPreference(w http.ResponseWriter, r *http.Request, deviceID string) {
    clientID := r.URL.Query().Get("client_id")
    if clientID == "" {
        clientID = "default"
    }

    pref, err := h.DB.GetPreferenceByDeviceAndClientID(deviceID, clientID, nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if pref == nil {
        http.Error(w, "Preference not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(pref)
}

// createPreference creates a new preference
func (h *Handler) createPreference(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Creating preference...")
    var newPref models.PreferenceCreate
    if err := json.NewDecoder(r.Body).Decode(&newPref); err != nil {
        fmt.Printf("Error decoding request body: %v\n", err)
        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
        return
    }

    fmt.Printf("Received preference create request: %+v\n", newPref)

    // Set default values if not provided
    if newPref.SpeedUnit == "" {
        newPref.SpeedUnit = "mph"
    }
    if newPref.DistanceUnit == "" {
        newPref.DistanceUnit = "miles"
    }
    if newPref.TemperatureUnit == "" {
        newPref.TemperatureUnit = "F"
    }

    if newPref.ClientID == "" {
        newPref.ClientID = "default"
    }

    pref, err := h.DB.CreatePreference(&newPref, nil)  // Pass nil as execer
    if err != nil {
        fmt.Printf("Error creating preference: %v\n", err)
        http.Error(w, fmt.Sprintf("Error creating preference: %v", err), http.StatusInternalServerError)
        return
    }

    fmt.Printf("Successfully created/updated preference: %+v\n", pref)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(pref)
}


// updatePreference updates an existing preference
func (h *Handler) updatePreference(w http.ResponseWriter, r *http.Request, deviceID string) {
    clientID := r.URL.Query().Get("client_id")
    if clientID == "" {
        clientID = "default"
    }

    var updates models.PreferenceUpdate
    if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Validate the new unit values if they are being updated
    if updates.SpeedUnit != nil && !isValidSpeedUnit(*updates.SpeedUnit) {
        http.Error(w, "Invalid speed unit", http.StatusBadRequest)
        return
    }
    if updates.DistanceUnit != nil && !isValidDistanceUnit(*updates.DistanceUnit) {
        http.Error(w, "Invalid distance unit", http.StatusBadRequest)
        return
    }
    if updates.TemperatureUnit != nil && !isValidTemperatureUnit(*updates.TemperatureUnit) {
        http.Error(w, "Invalid temperature unit", http.StatusBadRequest)
        return
    }

    // Try to get existing preference first
    existing, err := h.DB.GetPreferenceByDeviceAndClientID(deviceID, clientID, nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if existing == nil {
        http.Error(w, "Preference not found", http.StatusNotFound)
        return
    }

    pref, err := h.DB.UpdatePreferenceByDeviceAndClientID(deviceID, clientID, &updates, nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    fmt.Printf("Preference updated: %+v\n", pref)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(pref)
}


// deletePreference deletes a preference
func (h *Handler) deletePreference(w http.ResponseWriter, r *http.Request, deviceID string) {
    clientID := r.URL.Query().Get("client_id")
    if clientID == "" {
        clientID = "default"
    }

    err := h.DB.DeletePreference(deviceID, clientID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

// PreferencesHandler handles all preference-related requests
func (h *Handler) PreferencesHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println("PreferencesHandler called with path:", r.URL.Path)
	path := strings.TrimPrefix(r.URL.Path, "/preferences")
	deviceID := strings.TrimPrefix(path, "/")

	switch r.Method {
	case http.MethodGet:
		if deviceID == "" {
			h.getAllPreferences(w, r)
		} else {
			h.getPreference(w, r, deviceID)
		}
	case http.MethodPost:
		h.createPreference(w, r)
	case http.MethodPut:
		if deviceID == "" {
			http.Error(w, "Device ID required", http.StatusBadRequest)
			return
		}
		h.updatePreference(w, r, deviceID)
	case http.MethodDelete:
		if deviceID == "" {
			http.Error(w, "Device ID required", http.StatusBadRequest)
			return
		}
		h.deletePreference(w, r, deviceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Batch update preferences
func (h *Handler) BatchUpdatePreferences(w http.ResponseWriter, r *http.Request) {
    var preferences []models.PreferenceCreate
    if err := json.NewDecoder(r.Body).Decode(&preferences); err != nil {
        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
        return
    }

    if len(preferences) == 0 {
        http.Error(w, "No preferences provided", http.StatusBadRequest)
        return
    }

    // Start transaction
    tx, err := h.DB.Begin()
    if err != nil {
        http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
        return
    }
    defer tx.Rollback()

    // Process each preference
    for _, pref := range preferences {
        // Set default values if not provided
        if pref.SpeedUnit == "" {
            pref.SpeedUnit = "mph"
        }
        if pref.DistanceUnit == "" {
            pref.DistanceUnit = "miles"
        }
        if pref.TemperatureUnit == "" {
            pref.TemperatureUnit = "F"
        }
        // Validate units
        if !isValidSpeedUnit(pref.SpeedUnit) || !isValidDistanceUnit(pref.DistanceUnit) || !isValidTemperatureUnit(pref.TemperatureUnit) {
            http.Error(w, "Invalid unit preference in batch update", http.StatusBadRequest)
            return
        }

        _, err := h.DB.CreatePreference(&pref, tx)
        if err != nil {
            http.Error(w, fmt.Sprintf("Error updating preference: %v", err), http.StatusInternalServerError)
            return
        }
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        http.Error(w, fmt.Sprintf("Error committing transaction: %v", err), http.StatusInternalServerError)
        return
    }

    // Get updated preferences
    clientID := preferences[0].ClientID // All preferences should have same clientID
    updatedPrefs, err := h.DB.GetAllPreferencesForClient(clientID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error fetching updated preferences: %v", err), http.StatusInternalServerError)
        return
    }

    // Return updated preferences
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(updatedPrefs)
}

// Helper functions to validate units
func isValidSpeedUnit(unit string) bool {
    return unit == "mph" || unit == "km/h"
}

func isValidDistanceUnit(unit string) bool {
    return unit == "miles" || unit == "kilometers"
}

func isValidTemperatureUnit(unit string) bool {
    return unit == "F" || unit == "C"
}
