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

// Client handles API communication with OneStepGPS
type Client struct {
    apiKey     string
    httpClient *http.Client
}

// ReportStatus represents the status of a generated report
type ReportStatus struct {
    Status       string                 `json:"status"`
    Progress     map[string]interface{} `json:"progress"`
    DownloadURL  string                 `json:"download_url,omitempty"`
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
// GenerateReport sends a report generation request to OneStepGPS API
func (c *Client) GenerateReport(req *models.ReportRequest) (*models.ReportResponse, error) {
    url := fmt.Sprintf("%s/report/generate", baseURL)
    
    // Convert request to JSON
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("error marshaling request: %w", err)
    }

    // Create request
    request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }

    // Set headers
    request.Header.Set("Content-Type", "application/json")
    request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

    // Send request
    resp, err := c.httpClient.Do(request)
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

// GetReportStatus checks the status of a generated report
func (c *Client) GetReportStatus(reportID string) (*models.ReportStatus, error) {
    fmt.Printf("Getting status for report: %s\n", reportID)

    // Use the correct endpoint for report status
    url := fmt.Sprintf("%s/report-generated/%s", baseURL, reportID)
    fmt.Printf("Making request to: %s\n", url)
    
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

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("error reading response body: %w", err)
    }
    fmt.Printf("Raw response from OneStepGPS: %s\n", string(body))

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode, string(body))
    }

    var status models.ReportStatus
    if err := json.Unmarshal(body, &status); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }

    return &status, nil
}


// DownloadReport downloads the generated report
func (c *Client) DownloadReport(reportID string) ([]byte, string, error) {
    url := fmt.Sprintf("%s/report-generated/export/%s?file_type=pdf", baseURL, reportID)
    fmt.Printf("Attempting to download report from: %s\n", url)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, "", fmt.Errorf("error creating download request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, "", fmt.Errorf("error downloading report: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(bodyBytes))
    }

    content, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, "", fmt.Errorf("error reading download response: %w", err)
    }

    contentType := resp.Header.Get("Content-Type")
    return content, contentType, nil
}