package api

import (
	"net/http"
)

// SetupRoutes configures all the routes for our application
func (h *Handler) SetupRoutes() {
	// Vehicle routes with CORS middleware
	http.Handle("/vehicles", withCORS(http.HandlerFunc(h.VehiclesHandler)))
	http.Handle("/vehicles/", withCORS(http.HandlerFunc(h.VehicleHandler)))
	
	// Debug endpoint (consider removing in production)
	http.HandleFunc("/debug", h.debugHandler)
}

// withCORS middleware remains the same
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}