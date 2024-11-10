// config.go manages application configuration loading from environment variables,
// providing structured access to database, API, WebSocket, and other settings.

package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration settings
type Config struct {
    DBConfig    DatabaseConfig    // Database connection settings
    APIConfig   APIConfig         // API and server settings
    WebSocket   WebSocketConfig   // WebSocket connection settings
}

// DatabaseConfig holds MySQL database connection settings
type DatabaseConfig struct {
    DSN             string      // Database connection string
    MaxConnections  int         // Maximum number of concurrent DB connections
    ConnectTimeout  int         // Timeout in seconds for DB connection attempts
}

// APIConfig holds HTTP server and API settings
// Used by main.go for server setup and routes.go for CORS
type APIConfig struct {
    Port            string      // Server port (default 5000)
    AllowedOrigins  []string    // CORS allowed origins
    ReadTimeout     int         // Timeout for reading requests
    WriteTimeout    int         // Timeout for writing responses
    GPSApiKey       string      // OneStepGPS API authentication key
}

// WebSocketConfig holds WebSocket server settings
// Used by websocket/hub.go for real-time vehicle updates
type WebSocketConfig struct {
    ReadBufferSize  int         // Size of read buffer for WebSocket connections
    WriteBufferSize int         // Size of write buffer for WebSocket connections
    AllowedOrigins  []string    // Origins allowed to connect via WebSocket
}

// LoadConfig loads all configuration from environment variables
// Returns error if required variables are missing
func LoadConfig() (*Config, error) {
	// Initially load from env vars
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("DB_DSN environment variable is not set")
	}

    // Load database settings with defaults
	maxConn := getEnvInt("DB_MAX_CONNECTIONS", 10)
    connTimeout := getEnvInt("DB_CONNECT_TIMEOUT", 10)

    // Load API settings with development defaults
    port := getEnvStr("API_PORT", "5000")
    origins := getEnvSlice("ALLOWED_ORIGINS", []string{"http://localhost:5173"})
    readTimeout := getEnvInt("API_READ_TIMEOUT", 10)
    writeTimeout := getEnvInt("API_WRITE_TIMEOUT", 10)

    // GPS API key is required
    gpsApiKey := os.Getenv("GPS_API_KEY")
    if gpsApiKey == "" {
        return nil, fmt.Errorf("GPS_API_KEY environment variable is not set")
    }

    // Load WebSocket settings with defaults
    wsReadBuffer := getEnvInt("WS_READ_BUFFER", 1024)
    wsWriteBuffer := getEnvInt("WS_WRITE_BUFFER", 1024)
    wsOrigins := getEnvSlice("WS_ALLOWED_ORIGINS", []string{"http://localhost:5173"})

    // Construct and return complete config struct
    return &Config{
        DBConfig: DatabaseConfig{
            DSN:            dsn,
            MaxConnections: maxConn,
            ConnectTimeout: connTimeout,
        },
        APIConfig: APIConfig{
            Port:           port,
            AllowedOrigins: origins,
            ReadTimeout:    readTimeout,
            WriteTimeout:   writeTimeout,
            GPSApiKey:      gpsApiKey,
        },
        WebSocket: WebSocketConfig{
            ReadBufferSize:  wsReadBuffer,
            WriteBufferSize: wsWriteBuffer,
            AllowedOrigins:  wsOrigins,
        },
    }, nil
}

// Helper functions to get string environment variables with fallback
func getEnvStr(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}

// Helper function to get integer environment variable with fallback
// Converts string env var to int, returns fallback if conversion fails
func getEnvInt(key string, fallback int) int {
    if value, exists := os.LookupEnv(key); exists {
        if intVal, err := strconv.Atoi(value); err == nil {
            return intVal
        }
    }
    return fallback
}

// Helper function to get string slice environment variable with fallback
// Currently only supports single value, returns as single-item slice
func getEnvSlice(key string, fallback []string) []string {
    if value, exists := os.LookupEnv(key); exists {
        if value != "" {
            return []string{value}
        }
    }
    return fallback
}