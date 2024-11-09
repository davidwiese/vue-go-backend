package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// SetupRoutes configures all the routes for our application
func (h *Handler) SetupRoutes() {
    // Vehicle routes with CORS middleware
    http.Handle("/vehicles", withCORS(http.HandlerFunc(h.VehiclesHandler)))

    // Preferences routes with CORS middleware
    // Order matters: more specific routes first
    http.Handle("/preferences/batch", withCORS(http.HandlerFunc(h.BatchUpdatePreferences)))
    http.Handle("/preferences", withCORS(http.HandlerFunc(h.PreferencesHandler)))
    http.Handle("/preferences/", withCORS(http.HandlerFunc(h.PreferencesHandler)))

    // Report routes with CORS middleware
    fmt.Println("Registering report route: /report/generate")
    http.Handle("/report/generate", withCORS(http.HandlerFunc(h.GenerateReportHandler)))

    fmt.Println("Routes setup completed")
}

// getAllowedOrigins returns the list of allowed origins from environment variables
func getAllowedOrigins() []string {
	// Default development origin
	origins := []string{"http://localhost:5173"}
	
	// Get additional origins from environment variable
	if additionalOrigins := os.Getenv("ALLOWED_ORIGINS"); additionalOrigins != "" {
		// Split by comma and trim spaces
		for _, origin := range strings.Split(additionalOrigins, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				origins = append(origins, origin)
			}
		}
	}
	
	return origins
}

// isAllowedOrigin checks if the origin is in the allowed list
func isAllowedOrigin(origin string) bool {
	allowedOrigins := getAllowedOrigins()
	for _, allowed := range allowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}

func withCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        
        // If the origin is allowed, set it in the response header
        if isAllowedOrigin(origin) {
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            w.Header().Set("Access-Control-Allow-Credentials", "true")
        }

        // Handle preflight OPTIONS requests
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}