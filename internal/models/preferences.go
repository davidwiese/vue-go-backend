package models

import "time"

type UserPreference struct {
    ID              int       `json:"id"`
    DeviceID        string    `json:"device_id"`
    ClientID        string    `json:"client_id"`
    DisplayName     string    `json:"display_name"`
    IsHidden        bool      `json:"is_hidden"`
    SortOrder       int       `json:"sort_order"`
    SpeedUnit       string    `json:"speed_unit"`
    DistanceUnit    string    `json:"distance_unit"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

// PreferenceCreate represents the data needed to create a new preference
type PreferenceCreate struct {
    DeviceID        string `json:"device_id"`
    ClientID        string `json:"client_id"`
    DisplayName     string `json:"display_name,omitempty"`
    IsHidden        bool   `json:"is_hidden"`
    SortOrder       int    `json:"sort_order"`
    SpeedUnit       string `json:"speed_unit,omitempty"`
    DistanceUnit    string `json:"distance_unit,omitempty"`
}

// PreferenceUpdate represents the data that can be updated
// Using pointers allows you to differentiate between a field 
// that's not provided (nil) and a field that's intentionally 
// set to a zero value (e.g., empty string or false)
type PreferenceUpdate struct {
	DisplayName     *string `json:"display_name,omitempty"`
	IsHidden        *bool   `json:"is_hidden,omitempty"`
	SortOrder       *int    `json:"sort_order,omitempty"`
    SpeedUnit       *string `json:"speed_unit,omitempty"`
    DistanceUnit    *string `json:"distance_unit,omitempty"`
}