package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config represents the entire configuration structure
type Config struct {
	R2         R2Config         `yaml:"r2"`
	Conversion ConversionConfig `yaml:"conversion"`
	Resize     ResizeConfig     `yaml:"resize"`
	Cron       CronConfig       `yaml:"cron"`
	Server     ServerConfig     `yaml:"server"`
}

// R2Config contains Cloudflare R2 connection settings
type R2Config struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Endpoint  string `yaml:"endpoint"`
	Bucket    string `yaml:"bucket"`
}

// ConversionConfig contains image conversion settings
type ConversionConfig struct {
	Formats   []string `yaml:"formats"`
	Quality   int      `yaml:"quality"`
	MaxSizeMB int      `yaml:"max_size_mb"`
}

// ResizeConfig contains image resizing preset settings
type ResizeConfig struct {
	Presets map[string]PresetConfig `yaml:"presets"`
}

// PresetConfig defines a resize preset with width and height
type PresetConfig struct {
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

// CronConfig contains cron job scheduling settings
type CronConfig struct {
	Schedule string `yaml:"schedule"`
	Enabled  bool   `yaml:"enabled"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Port           int `yaml:"port"`
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

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
}

// setDefaults sets default values for optional configuration fields
func setDefaults(config *Config) {
	// Conversion defaults
	if len(config.Conversion.Formats) == 0 {
		config.Conversion.Formats = []string{"jpeg", "jpg", "png", "gif", "bmp", "tiff"}
	}
	if config.Conversion.Quality == 0 {
		config.Conversion.Quality = 85
	}
	if config.Conversion.MaxSizeMB == 0 {
		config.Conversion.MaxSizeMB = 50
	}

	// Cron defaults
	if config.Cron.Schedule == "" {
		config.Cron.Schedule = "0 2 * * *"
	}
	// enabled defaults to true if not explicitly set to false
	// (bool zero value is false, so we need to check if it was set)
	// For now, we'll keep the YAML value as-is

	// Server defaults
	if config.Server.Port == 0 {
		config.Server.Port = 4000
	}
	if config.Server.TimeoutSeconds == 0 {
		config.Server.TimeoutSeconds = 30
	}
}

// Validate validates the configuration
func Validate(config *Config) error {
	// Validate R2 required fields
	if config.R2.AccessKey == "" {
		return fmt.Errorf("required field missing: r2.access_key")
	}
	if config.R2.SecretKey == "" {
		return fmt.Errorf("required field missing: r2.secret_key")
	}
	if config.R2.Endpoint == "" {
		return fmt.Errorf("required field missing: r2.endpoint")
	}
	if config.R2.Bucket == "" {
		return fmt.Errorf("required field missing: r2.bucket")
	}

	// Validate conversion settings
	if config.Conversion.Quality < 0 || config.Conversion.Quality > 100 {
		return fmt.Errorf("conversion.quality must be between 0 and 100, got: %d", config.Conversion.Quality)
	}
	if config.Conversion.MaxSizeMB <= 0 {
		return fmt.Errorf("conversion.max_size_mb must be positive, got: %d", config.Conversion.MaxSizeMB)
	}

	// Validate server settings
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got: %d", config.Server.Port)
	}
	if config.Server.TimeoutSeconds <= 0 {
		return fmt.Errorf("server.timeout_seconds must be positive, got: %d", config.Server.TimeoutSeconds)
	}

	// Validate resize presets
	for name, preset := range config.Resize.Presets {
		if preset.Width <= 0 {
			return fmt.Errorf("resize.presets.%s.width must be positive, got: %d", name, preset.Width)
		}
		if preset.Height <= 0 {
			return fmt.Errorf("resize.presets.%s.height must be positive, got: %d", name, preset.Height)
		}
	}

	return nil
}
