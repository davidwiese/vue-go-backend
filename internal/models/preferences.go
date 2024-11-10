// preferences.go provides data structures for vehicle preferences

package models

import "time"

// UserPreference represents a stored vehicle display preference in the database.
// Used by VehiclePreferences.vue to customize how vehicles are shown in the UI.
type UserPreference struct {
    ID          int       `json:"id"`           // Primary key
    DeviceID    string    `json:"device_id"`    // Matches OneStepGPS device ID
    ClientID    string    `json:"client_id"`    // Allows multi-client preference storage
    DisplayName string    `json:"display_name"` // Custom name shown in VehicleList.vue
    IsHidden    bool      `json:"is_hidden"`
    SortOrder   int       `json:"sort_order"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// PreferenceCreate represents the data needed to create a new preference.
// Used when saving preferences in VehiclePreferences.vue via apiService.ts.
type PreferenceCreate struct {
    DeviceID    string `json:"device_id"`
    ClientID    string `json:"client_id"`
    DisplayName string `json:"display_name,omitempty"`
    IsHidden    bool   `json:"is_hidden"`
    SortOrder   int    `json:"sort_order"`
}

// PreferenceUpdate represents a partial update to existing preferences.
// Used for individual setting changes in VehiclePreferences.vue.
// Pointer types allow for null values, indicating no change needed.
type PreferenceUpdate struct {
	DisplayName *string `json:"display_name,omitempty"`
	IsHidden    *bool   `json:"is_hidden,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}