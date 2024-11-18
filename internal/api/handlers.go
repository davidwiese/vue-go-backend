// handlers.go handles all HTTP endpoints and business logic

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


// Handler manages API endpoints and holds required dependencies.
// Used throughout the application to handle HTTP requests.
type Handler struct {
    DB               *database.DB
    BroadcastChannel chan []models.Vehicle
    GPSClient        *onestepgps.Client
    config           HandlerConfig
}

// HandlerConfig holds API configuration settings
type HandlerConfig struct {
    OneStepGPSAPIKey string
    BaseURL          string
}

// NewHandler creates and initializes a Handler with required dependencies.
// Called in main.go to set up the application's request handler.
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

// VehiclesHandler handles GET requests to "/vehicles" endpoint.
// Used by frontend's fetchVehicles() in HomeView.vue to get initial vehicle data.
func (h *Handler) VehiclesHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        h.getVehicles(w, r)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// getVehicles fetches all vehicles from OneStepGPS API and returns them to the client.
func (h *Handler) getVehicles(w http.ResponseWriter, _ *http.Request) {
    vehicles, err := h.GPSClient.GetDevices()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(vehicles)
}

// PreferencesHandler manages all preference-related requests.
// Handles CRUD operations for vehicle display preferences from VehiclePreferences.vue.
func (h *Handler) PreferencesHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println("PreferencesHandler called with path:", r.URL.Path)

     // Extract deviceID from URL path if present
	path := strings.TrimPrefix(r.URL.Path, "/api/preferences")
	deviceID := strings.TrimPrefix(path, "/")

    // Route to appropriate handler based on HTTP method
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

// BatchUpdatePreferences handles bulk preference updates in a single transaction.
// Called from VehiclePreferences.vue when performing operations like "Show All" or "Hide All".
func (h *Handler) BatchUpdatePreferences(w http.ResponseWriter, r *http.Request) {
    var preferences []models.PreferenceCreate
    if err := json.NewDecoder(r.Body).Decode(&preferences); err != nil {
        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
        return
    }

    // Validate request
    if len(preferences) == 0 {
        http.Error(w, "No preferences provided", http.StatusBadRequest)
        return
    }

    // Start database transaction
    tx, err := h.DB.Begin()
    if err != nil {
        http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
        return
    }
    defer tx.Rollback() // Rollback transaction if error occurs/not committed

    // Process each preference in the transaction
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

// getAllPreferences fetches all preferences for the current client.
func (h *Handler) getAllPreferences(w http.ResponseWriter, r *http.Request) {
    // Get client_id from query parameter
    clientID := r.URL.Query().Get("client_id")
    if clientID == "" {
        clientID = "default"
    }

    // Fetch preferences from database
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

// getPreference fetches a single preference by device ID and client ID.
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

// createPreference creates a new preference for the current client.
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


// updatePreference updates an existing preference for the current client.
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


// deletePreference deletes a preference for the current client.
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

// GenerateReportHandler processes report generation requests from ReportDialog.vue.
// It handles the entire report generation lifecycle:
// 1. Initiates report generation with OneStepGPS
// 2. Polls for completion
// 3. Downloads and streams the completed report to the client
func (h *Handler) GenerateReportHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println("GenerateReportHandler called")

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Parse and validate the incoming request
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

    // Construct API request using the incoming spec directly
    apiReq := models.ReportRequest{
    DateTimeFrom: incomingReq.ReportSpec.DateTimeFrom,
    DateTimeTo:   incomingReq.ReportSpec.DateTimeTo,
    DeviceIDList: incomingReq.ReportSpec.DeviceIDList,
    ReportType:   "general_info",  // Hardcoded to always use general_info
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

    // Initialize report generation with OneStepGPS API
    generateURL := fmt.Sprintf("%s/report/generate", h.config.BaseURL)
    jsonData, err := json.Marshal(apiReq)
    if err != nil {
        http.Error(w, "Error preparing request", http.StatusInternalServerError)
        return
    }

    fmt.Printf("Sending request to OneStepGPS: %s\n", string(jsonData))

    // Send report generation request to OneStepGPS API
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

    // Parse initial response to get report ID
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        http.Error(w, "Error reading response", http.StatusInternalServerError)
        return
    }
    fmt.Printf("Initial response: %s\n", string(respBody))

    // Define a local struct to match the JSON response structure from OneStepGPS
    var generateResponse struct {
        ReportGeneratedID string `json:"report_generated_id"`
        Status           string `json:"status"`
        Error            string `json:"error"`
    }

    // Unmarshal (parse) the JSON response into our struct
    if err := json.Unmarshal(respBody, &generateResponse); err != nil {
        http.Error(w, "Error parsing response", http.StatusInternalServerError)
        return
    }

    // Check if the API returned an error message
    if generateResponse.Error != "" {
        fmt.Printf("Error from API: %s\n", generateResponse.Error)
        http.Error(w, generateResponse.Error, http.StatusInternalServerError)
        return
    }

    // Store the report ID for polling
    // Reports are generated asynchronously, so we need to poll for completion
    reportID := generateResponse.ReportGeneratedID
    maxAttempts := 60 // Will try for 60 seconds before (1 attempt per second)
    
    // Start polling loop - similar to setInterval in JavaScript
    // but using a for loop with sleep instead
    for attempt := 0; attempt < maxAttempts; attempt++ {
        fmt.Printf("Checking status attempt %d/%d\n", attempt+1, maxAttempts)
        
        // Construct URL for status check endpoint
        statusURL := fmt.Sprintf("%s/report-generated/%s", h.config.BaseURL, reportID)

        // Create new GET request with authorization header
        statusReq, _ := http.NewRequest("GET", statusURL, nil)
        statusReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.config.OneStepGPSAPIKey))

        // Send the request and get the response
        statusResp, err := http.DefaultClient.Do(statusReq)
        if err != nil {
            http.Error(w, "Error checking status", http.StatusInternalServerError)
            return
        }

        // Read response body into []byte
        statusBody, err := io.ReadAll(statusResp.Body)
        statusResp.Body.Close()
        if err != nil {
            http.Error(w, "Error reading status", http.StatusInternalServerError)
            return
        }
        
        fmt.Printf("Status response: %s\n", string(statusBody))

        // Define a local struct to match the JSON response structure from OneStepGPS
        var statusResponse struct {
            Status     string `json:"status"`
            Error      string `json:"error"`
            OutputPath string `json:"OutputFilePath"`
        }

        // Parse status response JSON into struct
        if err := json.Unmarshal(statusBody, &statusResponse); err != nil {
            http.Error(w, "Error parsing status", http.StatusInternalServerError)
            return
        }

        fmt.Printf("Report status: %s\n", statusResponse.Status)

        // Check for API errors in status response
        if statusResponse.Error != "" {
            http.Error(w, fmt.Sprintf("Report failed: %s", statusResponse.Error), http.StatusInternalServerError)
            return
        }

        // If report is complete, download and send to client
        if statusResponse.Status == "done" {
            // Add a small delay to ensure PDF is fully generated
            time.Sleep(2 * time.Second)
            
            // Construct download URL with report ID and PDF format
            exportURL := fmt.Sprintf("%s/report-generated/export/%s?file_type=pdf&api-key=%s", 
                h.config.BaseURL, reportID, h.config.OneStepGPSAPIKey)
            fmt.Printf("Downloading report from: %s\n", exportURL)

            // Create GET request for PDF download
            exportReq, _ := http.NewRequest("GET", exportURL, nil)
            exportReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.config.OneStepGPSAPIKey))

            // Download the PDF
            exportResp, err := http.DefaultClient.Do(exportReq)
            if err != nil {
                http.Error(w, "Error downloading report", http.StatusInternalServerError)
                return
            }
            defer exportResp.Body.Close()

            // Stream PDF to client
            w.Header().Set("Content-Type", "application/pdf")
            w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.pdf", reportID))
            
            _, err = io.Copy(w, exportResp.Body)
            if err != nil {
                http.Error(w, "Error streaming report", http.StatusInternalServerError)
                return
            }
            return
        }

        time.Sleep(1 * time.Second) // Wait before next polling attempt
    }

    http.Error(w, "Report generation timed out", http.StatusGatewayTimeout)
}