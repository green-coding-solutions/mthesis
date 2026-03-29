package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type DatabaseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
}

// LoadDatabaseConfig loads and validates all required DB settings from env.
func LoadDatabaseConfig() (DatabaseConfig, error) {
	if err := godotenv.Overload(".env"); err != nil {
		return DatabaseConfig{}, fmt.Errorf("load .env file: %w", err)
	}

	host, err := requiredEnv("DATABASE_HOST")
	if err != nil {
		return DatabaseConfig{}, err
	}

	portRaw, err := requiredEnv("DATABASE_PORT")
	if err != nil {
		return DatabaseConfig{}, err
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("invalid DATABASE_PORT: %w", err)
	}
	if port < 1 || port > 65535 {
		return DatabaseConfig{}, fmt.Errorf("invalid DATABASE_PORT: must be between 1 and 65535")
	}

	username, err := requiredEnv("DATABASE_USERNAME")
	if err != nil {
		return DatabaseConfig{}, err
	}

	password, err := requiredEnv("DATABASE_PASSWORD")
	if err != nil {
		return DatabaseConfig{}, err
	}

	database, err := requiredEnv("DATABASE_NAME")
	if err != nil {
		return DatabaseConfig{}, err
	}

	schema, err := requiredEnv("DATABASE_SCHEMA")
	if err != nil {
		return DatabaseConfig{}, err
	}

	return DatabaseConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Database: database,
		Schema:   schema,
	}, nil
}

// requiredEnv centralizes missing-variable validation for configuration keys.
func requiredEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}

	return value, nil
}
