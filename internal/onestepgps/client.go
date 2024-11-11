// client.go provides a client for interacting with the OneStepGPS API,
// handling real-time vehicle data retrieval and report generation.

package onestepgps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/models"
)

const (
	baseURL = "https://track.onestepgps.com/v3/api/public"
)

// Client handles authenticated communication with OneStepGPS API.
// Used by handlers.go and websocket/hub.go for vehicle data and reports.
type Client struct {
    apiKey     string
    httpClient *http.Client
}

// ReportStatus represents the status of a generated report from OneStepGPS.
// Used during report generation polling process.
type ReportStatus struct {
    Status       string                 `json:"status"`
    Progress     map[string]interface{} `json:"progress"`
    DownloadURL  string                 `json:"download_url,omitempty"`
}

// NewClient creates a new OneStepGPS API client with configured timeout.
// Called in main.go during application initialization.
func NewClient(apiKey string) *Client {
    return &Client{
        apiKey: apiKey,
        httpClient: &http.Client{
            Timeout: time.Second * 10,
        },
    }
}

// GetDevices retrieves all vehicles with their latest positions.
// Used by websocket hub for real-time updates and initial data load.
func (c *Client) GetDevices() ([]models.Vehicle, error) {
    // Build URL without api key in query param
    url := fmt.Sprintf("%s/device?latest_point=true", baseURL)
    fmt.Printf("Making request to URL: %s\n", url)
    fmt.Printf("Using API Key: %s\n", c.apiKey)  // Be careful with logging keys in production
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
    fmt.Printf("Request headers: %+v\n", req.Header)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error making request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode, string(body))
    }

    var apiResp models.APIResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }

    return apiResp.ResultList, nil
}

// GetVehicleUpdates polls for vehicle updates at specified interval.
// Used by websocket hub to maintain real-time vehicle data.
func (c *Client) GetVehicleUpdates(interval time.Duration, updates chan<- []models.Vehicle) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    // Continuous polling loop
    for range ticker.C {
        vehicles, err := c.GetDevices()
        if err != nil {
            log.Printf("Error fetching vehicle updates: %v", err)
            continue
        }
        updates <- vehicles // Send updates to WebSocket broadcast channel
    }
}

// GenerateReport initiates report generation with OneStepGPS API.
// Called by GenerateReportHandler when user requests a report in ReportDialog.vue.
func (c *Client) GenerateReport(req *models.ReportRequest) (*models.ReportResponse, error) {
    url := fmt.Sprintf("%s/report/generate", baseURL)
    
    // Prepare request body
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("error marshaling request: %w", err)
    }

    // Create and configure request
    request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }

    request.Header.Set("Content-Type", "application/json")
    request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

    // Send request and handle response
    resp, err := c.httpClient.Do(request)
    if err != nil {
        return nil, fmt.Errorf("error sending request: %w", err)
    }
    defer resp.Body.Close()

    // Check response status
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
    }

    // Parse response into ReportResponse struct
    var reportResp models.ReportResponse
    if err := json.NewDecoder(resp.Body).Decode(&reportResp); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }

    return &reportResp, nil
}

// GetReportStatus checks the generation status of a specific report.
// Used during report generation polling in GenerateReportHandler.
func (c *Client) GetReportStatus(reportID string) (*models.ReportStatus, error) {
    fmt.Printf("Getting status for report: %s\n", reportID)

    // Use the correct endpoint for report status
    url := fmt.Sprintf("%s/report-generated/%s", baseURL, reportID)
    fmt.Printf("Making request to: %s\n", url)
    
    // Create and send status check request
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error getting report status: %w", err)
    }
    defer resp.Body.Close()

    // Read and log complete response
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("error reading response body: %w", err)
    }
    fmt.Printf("Raw response from OneStepGPS: %s\n", string(body))

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode, string(body))
    }

    // Parse response into ReportStatus struct
    var status models.ReportStatus
    if err := json.Unmarshal(body, &status); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }

    return &status, nil
}


// DownloadReport downloads a generated report PDF.
// Called when report is ready in GenerateReportHandler.
func (c *Client) DownloadReport(reportID string) ([]byte, string, error) {
    url := fmt.Sprintf("%s/report-generated/export/%s?file_type=pdf", baseURL, reportID)
    fmt.Printf("Attempting to download report from: %s\n", url)

    // Create download request
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, "", fmt.Errorf("error creating download request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

    // Execute download request
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, "", fmt.Errorf("error downloading report: %w", err)
    }
    defer resp.Body.Close()

    // Handle failed download
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(bodyBytes))
    }

    // Read PDF content
    content, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, "", fmt.Errorf("error reading download response: %w", err)
    }

    contentType := resp.Header.Get("Content-Type")
    return content, contentType, nil
}