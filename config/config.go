package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Load loads configuration from a YAML file
// It also automatically loads .env file if it exists (non-fatal if missing)
func Load(configPath string) (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variables (override YAML values)
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
	if accessKey := os.Getenv("R2_ACCESS_KEY"); accessKey != "" {
		config.R2.AccessKey = accessKey
	}
	if secretKey := os.Getenv("R2_SECRET_KEY"); secretKey != "" {
		config.R2.SecretKey = secretKey
	}
	if endpoint := os.Getenv("R2_ENDPOINT"); endpoint != "" {
		config.R2.Endpoint = endpoint
	}
	if bucket := os.Getenv("R2_BUCKET"); bucket != "" {
		config.R2.Bucket = bucket
	}
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Server.Port = port
		}
	}
	if webhookURL := os.Getenv("WEBHOOK_URL"); webhookURL != "" {
		config.Webhook.URL = webhookURL
	}
	if v := os.Getenv("WEBHOOK_RETRY_ENABLED"); v == "false" || v == "0" {
		config.Webhook.RetryEnabled = false
	} else if v == "true" || v == "1" {
		config.Webhook.RetryEnabled = true
	}
	if v := os.Getenv("WEBHOOK_RETRY_INTERVAL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			config.Webhook.RetryIntervalMinutes = n
		}
	}
	if v := os.Getenv("WEBHOOK_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			config.Webhook.MaxRetries = n
		}
	}
	if v := os.Getenv("WEBHOOK_PENDING_DIR"); v != "" {
		config.Webhook.PendingDir = v
	}
}
