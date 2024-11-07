package onestepgps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/models"
)

const (
	baseURL = "https://track.onestepgps.com/v3/api/public"
)

// Client handles API communication with OneStepGPS
type Client struct {
    apiKey     string
    httpClient *http.Client
}

// NewClient creates a new OneStepGPS API client
func NewClient(apiKey string) *Client {
    return &Client{
        apiKey: apiKey,
        httpClient: &http.Client{
            Timeout: time.Second * 10,
        },
    }
}

// GetDevices retrieves all devices with their latest positions
func (c *Client) GetDevices() ([]models.Vehicle, error) {
    // Build URL with all needed parameters
    url := fmt.Sprintf("%s/device?latest_point=true&api-key=%s", baseURL, c.apiKey)
    
    resp, err := c.httpClient.Get(url)
    if err != nil {
        return nil, fmt.Errorf("error making request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
    }

    var apiResp models.APIResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }

    return apiResp.ResultList, nil
}

// GetVehicleUpdates polls for vehicle updates
func (c *Client) GetVehicleUpdates(interval time.Duration, updates chan<- []models.Vehicle) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for range ticker.C {
        vehicles, err := c.GetDevices()
        if err != nil {
            log.Printf("Error fetching vehicle updates: %v", err)
            continue
        }
        updates <- vehicles
    }
}

// GenerateReport sends a report generation request to OneStepGPS API
func (c *Client) GenerateReport(spec *models.ReportSpec) (*models.ReportResponse, error) {
    url := fmt.Sprintf("%s/report/generate", baseURL)
    
    // Create request body
    reqBody := models.ReportRequest{
        ReportSpec: *spec,
    }
    
    // Convert request to JSON
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return nil, fmt.Errorf("error marshaling request: %w", err)
    }

    // Create request
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

    // Send request
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error sending request: %w", err)
    }
    defer resp.Body.Close()

    // Check response status
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
    }

    // Decode response
    var reportResp models.ReportResponse
    if err := json.NewDecoder(resp.Body).Decode(&reportResp); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }

    return &reportResp, nil
}