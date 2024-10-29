package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBConfig DatabaseConfig
	APIConfig APIConfig
}

type DatabaseConfig struct {
	DSN string
}

type APIConfig struct {
	Port string
}

func LoadConfig() (*Config, error) {
	// Initially load from env vars
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("DB_DSN environment variable is not set")
	}

	return &Config{
		DBConfig: DatabaseConfig{
			DSN: dsn,
		},
		APIConfig: APIConfig{
			Port: "5000",
		},
	}, nil
}