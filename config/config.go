package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Load loads configuration from an optional YAML file and environment variables.
// If configPath is empty or the file does not exist, configuration is built from .env/env only.
// It automatically loads .env if it exists (non-fatal if missing).
func Load(configPath string) (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	var config Config
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := yaml.Unmarshal(data, &config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Apply environment variables (override YAML values or fill when no YAML)
	applyEnvironmentVariables(&config)

	// Set default values
	setDefaults(&config)

	// Validate configuration
	if err := Validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// applyEnvironmentVariables applies environment variables to config
// Environment variables take precedence over YAML values
func applyEnvironmentVariables(config *Config) {
	loadR2FromEnv(config)
	loadConversionFromEnv(config)
	loadResizeFromEnv(config)
	loadCronFromEnv(config)
	loadServerFromEnv(config)
	loadWebhookFromEnv(config)
}
