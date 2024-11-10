// routes.go provides routing and middleware configuration.
// It maps frontend requests to their corresponding handlers and manages CORS policies
// for both development and production environments.

package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// RouteGroup represents a group of related routes
type RouteGroup struct {
    prefix  string
    handler *Handler
    routes  []Route
}

// Route represents a single endpoint configuration
type Route struct {
    path    string
    method  string
    handler http.HandlerFunc
}

// SetupRoutes configures all API endpoints for the application
// Called in main.go during server initialization
func (h *Handler) SetupRoutes() {
    // Define route groups with their respective endpoints
    groups := []RouteGroup{
        {
            prefix: "/vehicles",
            handler: h,
            routes: []Route{
                {
                    // Used in HomeView.vue: fetchVehicles() to get initial vehicle data
                    // GET /vehicles - Returns list of all vehicles
                    path:    "",
                    method:  http.MethodGet,
                    handler: h.VehiclesHandler,
                },
            },
        },
        {
            prefix: "/preferences",
            handler: h,
            routes: []Route{
                {
                    // Used in VehiclePreferences.vue: savePreferencesBatch() in apiService.ts
                    // POST /preferences/batch - Bulk update vehicle preferences
                    path:    "/batch",
                    method:  http.MethodPost,
                    handler: h.BatchUpdatePreferences,
                },
                {
                    // Used in VehiclePreferences.vue for CRUD operations
                    // Multiple methods (GET, POST) for /preferences
                    // GET: getPreferences() in apiService.ts
                    // POST: savePreference() in apiService.ts
                    path:    "",
                    method:  "*", // Allows multiple HTTP methods
                    handler: h.PreferencesHandler,
                },
                {
                    // Handles operations on specific preferences by ID
                    // Used for PUT/DELETE operations in VehiclePreferences.vue
                    path:    "/",
                    method:  "*", // Handles requests with IDs
                    handler: h.PreferencesHandler,
                },
            },
        },
        {
            prefix: "/report",
            handler: h,
            routes: []Route{
                {
                    // Used in ReportDialog.vue: generateReport()
                    // POST /report/generate - Generates and returns PDF report
                    path:    "/generate",
                    method:  http.MethodPost,
                    handler: h.GenerateReportHandler,
                },
            },
        },
    }

    // Registers each route with middleware
    for _, group := range groups {
        for _, route := range group.routes {
            fullPath := group.prefix + route.path
            fmt.Printf("Registering route: %s\n", fullPath)
            // Apply CORS and method checking middleware to each route
            http.Handle(fullPath, withCORS(methodHandler(route.method, route.handler)))
        }
    }

    fmt.Println("Routes setup completed")
}

// methodHandler ensures requests use the allowed HTTP method
func methodHandler(allowedMethod string, handler http.HandlerFunc) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Allow all methods if "*" is specified, otherwise check for match
        if allowedMethod != "*" && r.Method != allowedMethod {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        handler(w, r)
    })
}

// withCORS adds CORS headers to responses
func withCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        
        // Set CORS headers if origin is allowed
        if isAllowedOrigin(origin) {
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            w.Header().Set("Access-Control-Allow-Credentials", "true")
        }

        // Handle preflight requests (OPTIONS method)
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}

// getAllowedOrigins retrieves allowed CORS origins from environment
// Development: localhost:5173
// Production: S3 bucket URL
func getAllowedOrigins() []string {
	// Default development origin
	origins := []string{"http://localhost:5173"}
	
	// Add production origins from environment variable
	if additionalOrigins := os.Getenv("ALLOWED_ORIGINS"); additionalOrigins != "" {
		// Split comma-separated origins and clean them
		for _, origin := range strings.Split(additionalOrigins, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				origins = append(origins, origin)
			}
		}
	}
	
	return origins
}

// isAllowedOrigin checks if an origin is allowed for CORS
// Used by withCORS middleware to validate request origins
func isAllowedOrigin(origin string) bool {
	allowedOrigins := getAllowedOrigins()
	for _, allowed := range allowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}