package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Vehicle represents a vehicle entity
type Vehicle struct {
    ID        int     `json:"id"`
    Name      string  `json:"name"`
    Status    string  `json:"status"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Action    string  `json:"action,omitempty"`
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

// GET /vehicles
func getVehicles(w http.ResponseWriter, r *http.Request) {
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

// POST /vehicles
func createVehicle(w http.ResponseWriter, r *http.Request) {
    var v Vehicle
    err := json.NewDecoder(r.Body).Decode(&v)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    res, err := db.Exec("INSERT INTO vehicles (name, status, latitude, longitude) VALUES (?, ?, ?, ?)",
        v.Name, v.Status, v.Latitude, v.Longitude)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

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

// GET /vehicles/{id}
func getVehicle(w http.ResponseWriter, r *http.Request, id int) {
    var v Vehicle
    err := db.QueryRow("SELECT id, name, status, latitude, longitude FROM vehicles WHERE id = ?", id).Scan(
        &v.ID, &v.Name, &v.Status, &v.Latitude, &v.Longitude)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(v)
}

// PUT /vehicles/{id}
func updateVehicle(w http.ResponseWriter, r *http.Request, id int) {
    var v Vehicle
    err := json.NewDecoder(r.Body).Decode(&v)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    v.ID = id

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


// DELETE /vehicles/{id}
func deleteVehicle(w http.ResponseWriter, r *http.Request, id int) {
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
