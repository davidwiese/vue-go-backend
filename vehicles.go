package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Vehicle represents a vehicle entity
// The struct tags `json:"..."` specify how fields are encoded/decoded in JSON
type Vehicle struct {
    ID        int     `json:"id"`
    Name      string  `json:"name"`
    // Status is indexed to improve performance
    Status    string  `json:"status"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Action    string  `json:"action,omitempty"` // omitted from JSON if empty
}

// Handler for "/vehicles" endpoint
func vehiclesHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        getVehicles(w, r)
    case http.MethodPost:
        createVehicle(w, r)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// Handler for "/vehicles/{id}" endpoint
func vehicleHandler(w http.ResponseWriter, r *http.Request) {
    // Extract the ID from the URL path
    idStr := strings.TrimPrefix(r.URL.Path, "/vehicles/")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid vehicle ID", http.StatusBadRequest)
        return
    }

    switch r.Method {
    case http.MethodGet:
        getVehicle(w, r, id)
    case http.MethodPut:
        updateVehicle(w, r, id)
    case http.MethodDelete:
        deleteVehicle(w, r, id)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// GET /vehicles (returns all vehicles from db)
func getVehicles(w http.ResponseWriter, _ *http.Request) {
    rows, err := db.Query("SELECT id, name, status, latitude, longitude FROM vehicles")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    vehicles := []Vehicle{}
    for rows.Next() {
        var v Vehicle
        if err := rows.Scan(&v.ID, &v.Name, &v.Status, &v.Latitude, &v.Longitude); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        vehicles = append(vehicles, v)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(vehicles)
}

// POST /vehicles (adds a new vehicle to db)
func createVehicle(w http.ResponseWriter, r *http.Request) {
    var v Vehicle
    // Decode the JSON request body into the Vehicle struct
    err := json.NewDecoder(r.Body).Decode(&v)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Insert the new vehicle into the database
    res, err := db.Exec("INSERT INTO vehicles (name, status, latitude, longitude) VALUES (?, ?, ?, ?)",
        v.Name, v.Status, v.Latitude, v.Longitude)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    // Get the ID of the newly inserted vehicle
    id, err := res.LastInsertId()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    v.ID = int(id)

    // Send the new vehicle to the broadcast channel
    broadcastChannel <- v

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(v)
}

// GET /vehicles/{id} (returns a single vehicle from db)
func getVehicle(w http.ResponseWriter, _ *http.Request, id int) {
    var v Vehicle
    // Query the vehicle by ID
    err := db.QueryRow("SELECT id, name, status, latitude, longitude FROM vehicles WHERE id = ?", id).Scan(
        &v.ID, &v.Name, &v.Status, &v.Latitude, &v.Longitude)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(v)
}

// PUT /vehicles/{id} (updates a single vehicle in db)
func updateVehicle(w http.ResponseWriter, r *http.Request, id int) {
    var v Vehicle
    // Decode the JSON request body into the Vehicle struct
    err := json.NewDecoder(r.Body).Decode(&v)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    v.ID = id  // Ensure the ID is set to the correct value

    // Update the vehicle in the database
    _, err = db.Exec("UPDATE vehicles SET name = ?, status = ?, latitude = ?, longitude = ? WHERE id = ?",
        v.Name, v.Status, v.Latitude, v.Longitude, v.ID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Send the updated vehicle to the broadcast channel
    broadcastChannel <- v

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(v)
}


// DELETE /vehicles/{id} (deletes a single vehicle from db)
func deleteVehicle(w http.ResponseWriter, _ *http.Request, id int) {
    _, err := db.Exec("DELETE FROM vehicles WHERE id = ?", id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Send a message indicating deletion
    v := Vehicle{
        ID:     id,
        Action: "delete",
    }
    broadcastChannel <- v

    w.WriteHeader(http.StatusNoContent)
}
