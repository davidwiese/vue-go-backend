package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/database"
	"github.com/davidwiese/fleet-tracker-backend/internal/models"
	"github.com/davidwiese/fleet-tracker-backend/internal/onestepgps"
)

const (
    baseURL = "https://track.onestepgps.com/v3/api/public"
)

// Handler struct to hold dependencies
type Handler struct {
    DB               *database.DB
    BroadcastChannel chan []models.Vehicle
    GPSClient        *onestepgps.Client
    config           HandlerConfig
}

type HandlerConfig struct {
    OneStepGPSAPIKey string
    BaseURL          string
}

// NewHandler creates a new Handler instance
func NewHandler(db *database.DB, broadcastChannel chan []models.Vehicle, gpsClient *onestepgps.Client, config HandlerConfig) *Handler {
    if config.BaseURL == "" {
        config.BaseURL = baseURL
    }
    return &Handler{
        DB:               db,
        BroadcastChannel: broadcastChannel,
        GPSClient:        gpsClient,
        config:           config,
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

// GenerateReportHandler handles report generation requests
func (h *Handler) GenerateReportHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println("GenerateReportHandler called")

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Parse request body and log it
    var incomingReq struct {
        ReportSpec models.ReportSpec `json:"report_spec"`
    }
    
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Error reading request body", http.StatusBadRequest)
        return
    }
    fmt.Printf("Incoming request body: %s\n", string(body))

    if err := json.Unmarshal(body, &incomingReq); err != nil {
        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
        return
    }

    // Construct the API request
    apiReq := models.ReportRequest{
        DateTimeFrom: incomingReq.ReportSpec.DateTimeFrom,
        DateTimeTo:   incomingReq.ReportSpec.DateTimeTo,
        DeviceIDList: incomingReq.ReportSpec.DeviceIDList,
        ReportType:   "general_info",
        UserReportName: incomingReq.ReportSpec.UserReportName,
        ReportOutputFieldList: []string{
            "device_id",
            "device_name",
            "groups",
            "route_length",
            "move_duration",
            "stop_duration",
            "stop_count",
            "speed_top",
            "speed_avg",
            "speed_count",
            "engine_work",
            "engine_idle",
            "engine_time",
        },
        ReportOptions: map[string]interface{}{
            "display_decimal_places": 1,
            "duration_format": "standard",
            "min_stop_duration": map[string]interface{}{
                "value": 5,
                "unit": "m",
                "display": "5m",
            },
            "use_pdf_landscape": true,
        },
        ReportOptionsGeneralInfo: map[string]interface{}{
            "minimum_speeding_threshold": map[string]interface{}{
                "value": 50,
                "unit": "mph",
                "display": "50 mph",
            },
            "use_nonmerged_layout": false,
        },
    }

    // Generate report
    generateURL := fmt.Sprintf("%s/report/generate", h.config.BaseURL)
    jsonData, err := json.Marshal(apiReq)
    if err != nil {
        http.Error(w, "Error preparing request", http.StatusInternalServerError)
        return
    }

    fmt.Printf("Sending request to OneStepGPS: %s\n", string(jsonData))

    generateReq, _ := http.NewRequest("POST", generateURL, bytes.NewBuffer(jsonData))
    generateReq.Header.Set("Content-Type", "application/json")
    generateReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.config.OneStepGPSAPIKey))

    resp, err := http.DefaultClient.Do(generateReq)
    if err != nil {
        fmt.Printf("Error making request: %v\n", err)
        http.Error(w, fmt.Sprintf("Error generating report: %v", err), http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    // Read and log the initial response
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        http.Error(w, "Error reading response", http.StatusInternalServerError)
        return
    }
    fmt.Printf("Initial response: %s\n", string(respBody))

    var generateResponse struct {
        ReportGeneratedID string `json:"report_generated_id"`
        Status           string `json:"status"`
        Error            string `json:"error"`
    }
    if err := json.Unmarshal(respBody, &generateResponse); err != nil {
        http.Error(w, "Error parsing response", http.StatusInternalServerError)
        return
    }

    if generateResponse.Error != "" {
        fmt.Printf("Error from API: %s\n", generateResponse.Error)
        http.Error(w, generateResponse.Error, http.StatusInternalServerError)
        return
    }

    // Poll for completion
    reportID := generateResponse.ReportGeneratedID
    maxAttempts := 60
    
    for attempt := 0; attempt < maxAttempts; attempt++ {
        fmt.Printf("Checking status attempt %d/%d\n", attempt+1, maxAttempts)
        
        statusURL := fmt.Sprintf("%s/report-generated/%s", h.config.BaseURL, reportID)
        statusReq, _ := http.NewRequest("GET", statusURL, nil)
        statusReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.config.OneStepGPSAPIKey))

        statusResp, err := http.DefaultClient.Do(statusReq)
        if err != nil {
            http.Error(w, "Error checking status", http.StatusInternalServerError)
            return
        }

        statusBody, err := io.ReadAll(statusResp.Body)
        statusResp.Body.Close()
        if err != nil {
            http.Error(w, "Error reading status", http.StatusInternalServerError)
            return
        }
        
        fmt.Printf("Status response: %s\n", string(statusBody))

        var statusResponse struct {
            Status     string `json:"status"`
            Error      string `json:"error"`
            OutputPath string `json:"OutputFilePath"`
        }
        if err := json.Unmarshal(statusBody, &statusResponse); err != nil {
            http.Error(w, "Error parsing status", http.StatusInternalServerError)
            return
        }

        fmt.Printf("Report status: %s\n", statusResponse.Status)

        if statusResponse.Error != "" {
            http.Error(w, fmt.Sprintf("Report failed: %s", statusResponse.Error), http.StatusInternalServerError)
            return
        }

        // Check for "done" status instead of "finished"
        if statusResponse.Status == "done" {
            exportURL := fmt.Sprintf("%s/report-generated/export/%s?file_type=pdf&api-key=%s", 
                h.config.BaseURL, reportID, h.config.OneStepGPSAPIKey)
            fmt.Printf("Downloading report from: %s\n", exportURL)

            exportReq, _ := http.NewRequest("GET", exportURL, nil)
            exportReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.config.OneStepGPSAPIKey))

            exportResp, err := http.DefaultClient.Do(exportReq)
            if err != nil {
                http.Error(w, "Error downloading report", http.StatusInternalServerError)
                return
            }
            defer exportResp.Body.Close()

            w.Header().Set("Content-Type", "application/pdf")
            w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.pdf", reportID))
            
            _, err = io.Copy(w, exportResp.Body)
            if err != nil {
                http.Error(w, "Error streaming report", http.StatusInternalServerError)
                return
            }
            return
        }

        // Add a shorter delay between checks since we know it's actively processing
        time.Sleep(1 * time.Second)
    }

    http.Error(w, "Report generation timed out", http.StatusGatewayTimeout)
}