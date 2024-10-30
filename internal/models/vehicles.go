package models

import "time"

// Vehicle represents the essential vehicle information from OneStepGPS
type Vehicle struct {
    DeviceID     string     `json:"device_id"`
    DisplayName  string     `json:"display_name"`
    ActiveState  string     `json:"active_state"`
    Online       bool       `json:"online"`
    LastLocation *Location  `json:"latest_device_point"`
    DriveState   DriveState `json:"device_state"`
}

// Location represents a point-in-time vehicle location
type Location struct {
    Timestamp time.Time `json:"dt_tracker"`
    Latitude  float64   `json:"lat"`
    Longitude float64   `json:"lng"`
    Altitude  *float64  `json:"altitude,omitempty"`
    Heading   int       `json:"angle"`
    Speed     float64   `json:"speed"`
    Detail    LocationDetail `json:"device_point_detail"`
}

// LocationDetail contains additional point information
type LocationDetail struct {
    Speed struct {
        Value   float64 `json:"value"`
        Unit    string  `json:"unit"`
        Display string  `json:"display"`
    } `json:"speed"`
    FuelPercent    *float64 `json:"fuel_percent,omitempty"`
    EngineOn       *bool    `json:"vbus_engine_on,omitempty"`
    InMotion       *bool    `json:"vbus_in_motion,omitempty"`
}

// DriveState represents the vehicle's current driving status
type DriveState struct {
    Status     string `json:"drive_status"` // "off", "idle", "driving"
    StatusID   string `json:"drive_status_id"`
    Distance   struct {
        Value   float64 `json:"value"`
        Unit    string  `json:"unit"`
        Display string  `json:"display"`
    } `json:"drive_status_distance"`
    BeginTime time.Time `json:"drive_status_begin_time"`
}

// APIResponse represents the top-level response from OneStepGPS API
type APIResponse struct {
    ResultList []Vehicle `json:"result_list"`
}

// Measurement represents their standard measurement format
type Measurement struct {
    Value   float64 `json:"value"`
    Unit    string  `json:"unit"`
    Display string  `json:"display"`
}