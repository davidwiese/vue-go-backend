package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
    DBConfig    DatabaseConfig
    APIConfig   APIConfig
    WebSocket   WebSocketConfig
    Simulation  SimulationConfig
}

type DatabaseConfig struct {
    DSN             string
    MaxConnections  int
    ConnectTimeout  int
}

type APIConfig struct {
    Port            string
    AllowedOrigins  []string
    ReadTimeout     int
    WriteTimeout    int
}

type WebSocketConfig struct {
    ReadBufferSize  int
    WriteBufferSize int
    AllowedOrigins  []string
}

type SimulationConfig struct {
    UpdateInterval  int // seconds
    MovementRadius  float64
}

// LoadConfig loads configuration from environment variables with sensible defaults
func LoadConfig() (*Config, error) {
	// Initially load from env vars
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("DB_DSN environment variable is not set")
	}

	maxConn := getEnvInt("DB_MAX_CONNECTIONS", 10)
    connTimeout := getEnvInt("DB_CONNECT_TIMEOUT", 10)

    // API configuration
    port := getEnvStr("API_PORT", "5000")
    origins := getEnvSlice("ALLOWED_ORIGINS", []string{"http://localhost:5173"})
    readTimeout := getEnvInt("API_READ_TIMEOUT", 10)
    writeTimeout := getEnvInt("API_WRITE_TIMEOUT", 10)

    // WebSocket configuration
    wsReadBuffer := getEnvInt("WS_READ_BUFFER", 1024)
    wsWriteBuffer := getEnvInt("WS_WRITE_BUFFER", 1024)
    wsOrigins := getEnvSlice("WS_ALLOWED_ORIGINS", []string{"http://localhost:5173"})

    // Simulation configuration
    simInterval := getEnvInt("SIM_UPDATE_INTERVAL", 5)
    simRadius := getEnvFloat("SIM_MOVEMENT_RADIUS", 0.01)

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
        },
        WebSocket: WebSocketConfig{
            ReadBufferSize:  wsReadBuffer,
            WriteBufferSize: wsWriteBuffer,
            AllowedOrigins:  wsOrigins,
        },
        Simulation: SimulationConfig{
            UpdateInterval: simInterval,
            MovementRadius: simRadius,
        },
    }, nil
}

// Helper functions to get environment variables with defaults
func getEnvStr(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}

func getEnvInt(key string, fallback int) int {
    if value, exists := os.LookupEnv(key); exists {
        if intVal, err := strconv.Atoi(value); err == nil {
            return intVal
        }
    }
    return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
    if value, exists := os.LookupEnv(key); exists {
        if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
            return floatVal
        }
    }
    return fallback
}

func getEnvSlice(key string, fallback []string) []string {
    if value, exists := os.LookupEnv(key); exists {
        if value != "" {
            return []string{value}
        }
    }
    return fallback
}