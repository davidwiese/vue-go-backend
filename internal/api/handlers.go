package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/davidwiese/fleet-tracker-backend/internal/database"
	"github.com/davidwiese/fleet-tracker-backend/internal/models"
)

// Handler struct to hold dependencies
type Handler struct {
	DB              *database.DB
	BroadcastChannel chan models.Vehicle
}

// NewHandler creates a new Handler instance
func NewHandler(db *database.DB, broadcastChannel chan models.Vehicle) *Handler {
	return &Handler{
		DB:               db,
		BroadcastChannel: broadcastChannel,
	}
}

// VehiclesHandler handles "/vehicles" endpoint
func (h *Handler) VehiclesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getVehicles(w, r)
	case http.MethodPost:
		h.createVehicle(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// VehicleHandler handles "/vehicles/{id}" endpoint
func (h *Handler) VehicleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/vehicles/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid vehicle ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getVehicle(w, r, id)
	case http.MethodPut:
		h.updateVehicle(w, r, id)
	case http.MethodDelete:
		h.deleteVehicle(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getVehicles returns all vehicles from db
func (h *Handler) getVehicles(w http.ResponseWriter, _ *http.Request) {
	rows, err := h.DB.Query("SELECT id, name, status, latitude, longitude FROM vehicles")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	vehicles := []models.Vehicle{}
	for rows.Next() {
		var v models.Vehicle
		if err := rows.Scan(&v.ID, &v.Name, &v.Status, &v.Latitude, &v.Longitude); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vehicles = append(vehicles, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vehicles)
}

// createVehicle adds a new vehicle to db
func (h *Handler) createVehicle(w http.ResponseWriter, r *http.Request) {
	var v models.Vehicle
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.DB.Exec("INSERT INTO vehicles (name, status, latitude, longitude) VALUES (?, ?, ?, ?)",
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

	h.BroadcastChannel <- v

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// getVehicle returns a single vehicle from db
func (h *Handler) getVehicle(w http.ResponseWriter, _ *http.Request, id int) {
	var v models.Vehicle
	err := h.DB.QueryRow("SELECT id, name, status, latitude, longitude FROM vehicles WHERE id = ?", id).Scan(
		&v.ID, &v.Name, &v.Status, &v.Latitude, &v.Longitude)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// updateVehicle updates a single vehicle in db
func (h *Handler) updateVehicle(w http.ResponseWriter, r *http.Request, id int) {
	var v models.Vehicle
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v.ID = id

	_, err := h.DB.Exec("UPDATE vehicles SET name = ?, status = ?, latitude = ?, longitude = ? WHERE id = ?",
		v.Name, v.Status, v.Latitude, v.Longitude, v.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.BroadcastChannel <- v

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// deleteVehicle deletes a single vehicle from db
func (h *Handler) deleteVehicle(w http.ResponseWriter, _ *http.Request, id int) {
	_, err := h.DB.Exec("DELETE FROM vehicles WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	v := models.Vehicle{
		ID:     id,
		Action: "delete",
	}
	h.BroadcastChannel <- v

	w.WriteHeader(http.StatusNoContent)
}

// debugHandler retrieves all vehicles (for development only)
func (h *Handler) debugHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT * FROM vehicles")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var vehicles []models.Vehicle
	for rows.Next() {
		var v models.Vehicle
		err := rows.Scan(&v.ID, &v.Name, &v.Status, &v.Latitude, &v.Longitude)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vehicles = append(vehicles, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vehicles)
}