package main

import (
	"database/sql"
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

    // Define HTTP routes
    http.HandleFunc("/vehicles", vehiclesHandler)
    http.HandleFunc("/vehicles/", vehicleHandler) // for /vehicles/{id}

    // Start the server
    log.Println("Server started on port 8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
