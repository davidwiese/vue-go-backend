package models

import "time"

type UserPreference struct {
    ID          int       `json:"id"`
    DeviceID    string    `json:"device_id"`
    ClientID    string    `json:"client_id"`
    DisplayName string    `json:"display_name"`
    IsHidden    bool      `json:"is_hidden"`
    SortOrder   int       `json:"sort_order"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// PreferenceCreate represents the data needed to create a new preference
type PreferenceCreate struct {
    DeviceID    string `json:"device_id"`
    ClientID    string `json:"client_id"`
    DisplayName string `json:"display_name,omitempty"`
    IsHidden    bool   `json:"is_hidden"`
    SortOrder   int    `json:"sort_order"`
}

// PreferenceUpdate represents the data that can be updated
type PreferenceUpdate struct {
	DisplayName *string `json:"display_name,omitempty"`
	IsHidden    *bool   `json:"is_hidden,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}