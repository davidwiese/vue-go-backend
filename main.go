package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Println("No .env file found, using environment variables")
    }
    
    dsn := os.Getenv("DB_DSN")
		if dsn == "" {
    log.Fatal("DB_DSN environment variable is not set")
		}

    db, err = sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Test the database connection
    err = db.Ping()
    if err != nil {
        log.Fatal(err)
    }

    // Define HTTP routes w/ CORS middleware
    http.Handle("/vehicles", withCORS(http.HandlerFunc(vehiclesHandler)))
		http.Handle("/vehicles/", withCORS(http.HandlerFunc(vehicleHandler))) // for /vehicles/{id}

		// DEBUGGING ENDPOINT FOR DEV ONLY
		http.HandleFunc("/debug", debugHandler)


    // Start the server
    log.Println("Server started on port 8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
		
}

// CORS middleware function
func withCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set the necessary headers
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        
        // Handle preflight OPTIONS requests
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        // Call the next handler
        next.ServeHTTP(w, r)
    })
}

// For debugging/dev only
func debugHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query("SELECT * FROM vehicles")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var vehicles []Vehicle
    for rows.Next() {
        var v Vehicle
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