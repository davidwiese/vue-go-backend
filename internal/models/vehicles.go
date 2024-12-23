// vehicles.go provides data structures for vehicle data from OneStepGPS

package models

import "time"

// Vehicle represents the essential vehicle information from OneStepGPS API.
// Used when receiving vehicle updates through WebSocket in HomeView.vue
// Transform API response to our domain model
type Vehicle struct {
    DeviceID     string     `json:"device_id"`
    DisplayName  string     `json:"display_name"`
    ActiveState  string     `json:"active_state"`
    Online       bool       `json:"online"`
    LastLocation *Location  `json:"latest_device_point"`
    DriveState   DriveState `json:"device_state"`
}

// Location represents a point-in-time vehicle location.
// Used by MapView.vue to position markers and display info windows
type Location struct {
    Timestamp time.Time `json:"dt_tracker"`
    Latitude  float64   `json:"lat"`
    Longitude float64   `json:"lng"`
    Altitude  *float64  `json:"altitude,omitempty"`
    Heading   int       `json:"angle"`
    Speed     float64   `json:"speed"`
    Detail    LocationDetail `json:"device_point_detail"`
}

// LocationDetail contains additional point information displayed in map info windows
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

// DriveState represents the vehicle's current driving status.
// Used in VehicleCard.vue for status display
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

// APIResponse represents the top-level response from OneStepGPS API.
// Used when fetching vehicle data in api/handlers.go
type APIResponse struct {
    ResultList []Vehicle `json:"result_list"`
}

// Measurement represents OneStepGPS's standard measurement format.
// Used throughout the API for consistent unit representation
type Measurement struct {
    Value   float64 `json:"value"`
    Unit    string  `json:"unit"`
    Display string  `json:"display"`
}