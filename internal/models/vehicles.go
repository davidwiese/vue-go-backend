package models

// Vehicle represents a vehicle entity
type Vehicle struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Status    string  `json:"status"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Action    string  `json:"action,omitempty"` // omitted from JSON if empty
}

// IsActive checks if the vehicle is in active status
func (v *Vehicle) IsActive() bool {
	return v.Status == "Active"
}

// UpdatePosition updates the vehicle's position by the given deltas
func (v *Vehicle) UpdatePosition(deltaLat, deltaLon float64) {
	v.Latitude += deltaLat
	v.Longitude += deltaLon
}