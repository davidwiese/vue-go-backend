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

// Route represents a single route
type Route struct {
    path    string
    method  string
    handler http.HandlerFunc
}

// SetupRoutes configures all the routes for our application
func (h *Handler) SetupRoutes() {
    // Define route groups
    groups := []RouteGroup{
        {
            prefix: "/vehicles",
            handler: h,
            routes: []Route{
                {
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
                    path:    "/batch",
                    method:  http.MethodPost,
                    handler: h.BatchUpdatePreferences,
                },
                {
                    path:    "",
                    method:  "*", // Special case for PreferencesHandler which handles multiple methods
                    handler: h.PreferencesHandler,
                },
                {
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
                    path:    "/generate",
                    method:  http.MethodPost,
                    handler: h.GenerateReportHandler,
                },
            },
        },
    }

    // Register all routes with CORS middleware
    for _, group := range groups {
        for _, route := range group.routes {
            fullPath := group.prefix + route.path
            fmt.Printf("Registering route: %s\n", fullPath)
            http.Handle(fullPath, withCORS(methodHandler(route.method, route.handler)))
        }
    }

    fmt.Println("Routes setup completed")
}

// methodHandler creates a handler that checks the HTTP method
func methodHandler(allowedMethod string, handler http.HandlerFunc) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if allowedMethod != "*" && r.Method != allowedMethod {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        handler(w, r)
    })
}

// CORS middleware functions remain the same
func withCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        
        if isAllowedOrigin(origin) {
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            w.Header().Set("Access-Control-Allow-Credentials", "true")
        }

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
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