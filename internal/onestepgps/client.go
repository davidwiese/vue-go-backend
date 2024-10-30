package onestepgps

import (
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