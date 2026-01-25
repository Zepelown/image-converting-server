package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test config
	testConfig := `r2:
  access_key: "test-access-key"
  secret_key: "test-secret-key"
  endpoint: "https://test.r2.cloudflarestorage.com"
  bucket: "test-bucket"
conversion:
  formats: ["jpeg", "png"]
  quality: 90
  max_size_mb: 100
resize:
  presets:
    thumbnail:
      width: 150
      height: 150
cron:
  schedule: "0 2 * * *"
  enabled: true
server:
  port: 8080
  timeout_seconds: 30
`

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	// Load config
	config, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate R2 config
	if config.R2.AccessKey != "test-access-key" {
		t.Errorf("Expected access_key 'test-access-key', got '%s'", config.R2.AccessKey)
	}
	if config.R2.SecretKey != "test-secret-key" {
		t.Errorf("Expected secret_key 'test-secret-key', got '%s'", config.R2.SecretKey)
	}
	if config.R2.Endpoint != "https://test.r2.cloudflarestorage.com" {
		t.Errorf("Expected endpoint 'https://test.r2.cloudflarestorage.com', got '%s'", config.R2.Endpoint)
	}
	if config.R2.Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", config.R2.Bucket)
	}

	// Validate conversion config
	if len(config.Conversion.Formats) != 2 {
		t.Errorf("Expected 2 formats, got %d", len(config.Conversion.Formats))
	}
	if config.Conversion.Quality != 90 {
		t.Errorf("Expected quality 90, got %d", config.Conversion.Quality)
	}
	if config.Conversion.MaxSizeMB != 100 {
		t.Errorf("Expected max_size_mb 100, got %d", config.Conversion.MaxSizeMB)
	}

	// Validate resize presets
	if len(config.Resize.Presets) != 1 {
		t.Errorf("Expected 1 preset, got %d", len(config.Resize.Presets))
	}
	thumbnail, ok := config.Resize.Presets["thumbnail"]
	if !ok {
		t.Error("Expected 'thumbnail' preset not found")
	} else {
		if thumbnail.Width != 150 || thumbnail.Height != 150 {
			t.Errorf("Expected thumbnail 150x150, got %dx%d", thumbnail.Width, thumbnail.Height)
		}
	}

	// Validate cron config
	if config.Cron.Schedule != "0 2 * * *" {
		t.Errorf("Expected schedule '0 2 * * *', got '%s'", config.Cron.Schedule)
	}
	if !config.Cron.Enabled {
		t.Error("Expected cron.enabled to be true")
	}

	// Validate server config
	if config.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.Port)
	}
	if config.Server.TimeoutSeconds != 30 {
		t.Errorf("Expected timeout_seconds 30, got %d", config.Server.TimeoutSeconds)
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Create a minimal config file
	tmpFile, err := os.CreateTemp("", "test-config-minimal-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write minimal config (only required fields)
	testConfig := `r2:
  access_key: "test-access-key"
  secret_key: "test-secret-key"
  endpoint: "https://test.r2.cloudflarestorage.com"
  bucket: "test-bucket"
`

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	// Load config
	config, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check defaults
	if len(config.Conversion.Formats) == 0 {
		t.Error("Expected default formats to be set")
	}
	if config.Conversion.Quality != 85 {
		t.Errorf("Expected default quality 85, got %d", config.Conversion.Quality)
	}
	if config.Conversion.MaxSizeMB != 50 {
		t.Errorf("Expected default max_size_mb 50, got %d", config.Conversion.MaxSizeMB)
	}
	if config.Cron.Schedule != "0 2 * * *" {
		t.Errorf("Expected default schedule '0 2 * * *', got '%s'", config.Cron.Schedule)
	}
	if config.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Server.Port)
	}
	if config.Server.TimeoutSeconds != 30 {
		t.Errorf("Expected default timeout_seconds 30, got %d", config.Server.TimeoutSeconds)
	}
}

func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("R2_ACCESS_KEY", "env-access-key")
	os.Setenv("R2_SECRET_KEY", "env-secret-key")
	os.Setenv("R2_ENDPOINT", "https://env.r2.cloudflarestorage.com")
	os.Setenv("R2_BUCKET", "env-bucket")
	os.Setenv("SERVER_PORT", "9000")
	defer func() {
		os.Unsetenv("R2_ACCESS_KEY")
		os.Unsetenv("R2_SECRET_KEY")
		os.Unsetenv("R2_ENDPOINT")
		os.Unsetenv("R2_BUCKET")
		os.Unsetenv("SERVER_PORT")
	}()

	// Create a config file with different values
	tmpFile, err := os.CreateTemp("", "test-config-env-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testConfig := `r2:
  access_key: "file-access-key"
  secret_key: "file-secret-key"
  endpoint: "https://file.r2.cloudflarestorage.com"
  bucket: "file-bucket"
server:
  port: 8080
`

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	// Load config
	config, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Environment variables should override file values
	if config.R2.AccessKey != "env-access-key" {
		t.Errorf("Expected env access_key 'env-access-key', got '%s'", config.R2.AccessKey)
	}
	if config.R2.SecretKey != "env-secret-key" {
		t.Errorf("Expected env secret_key 'env-secret-key', got '%s'", config.R2.SecretKey)
	}
	if config.R2.Endpoint != "https://env.r2.cloudflarestorage.com" {
		t.Errorf("Expected env endpoint 'https://env.r2.cloudflarestorage.com', got '%s'", config.R2.Endpoint)
	}
	if config.R2.Bucket != "env-bucket" {
		t.Errorf("Expected env bucket 'env-bucket', got '%s'", config.R2.Bucket)
	}
	if config.Server.Port != 9000 {
		t.Errorf("Expected env port 9000, got %d", config.Server.Port)
	}
}

func TestLoadConfigWithEnvFile(t *testing.T) {
	// Create a temporary .env file
	tmpEnvFile, err := os.CreateTemp("", ".env-*")
	if err != nil {
		t.Fatalf("Failed to create temp .env file: %v", err)
	}
	defer os.Remove(tmpEnvFile.Name())

	// Write test .env content
	envContent := `R2_ACCESS_KEY=env-file-access-key
R2_SECRET_KEY=env-file-secret-key
R2_ENDPOINT=https://env-file.r2.cloudflarestorage.com
R2_BUCKET=env-file-bucket
SERVER_PORT=9000
`

	if _, err := tmpEnvFile.WriteString(envContent); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}
	tmpEnvFile.Close()

	// Change to the directory containing the .env file
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	envDir := tmpEnvFile.Name()
	envDir = envDir[:len(envDir)-len(".env-*")+1]
	// Actually, we need to use the directory where the .env file is
	envDir = tmpEnvFile.Name()
	for i := len(envDir) - 1; i >= 0; i-- {
		if envDir[i] == '/' {
			envDir = envDir[:i]
			break
		}
	}

	os.Chdir(envDir)
	defer os.Chdir(oldDir)

	// Rename to .env in that directory
	envPath := envDir + "/.env"
	os.Rename(tmpEnvFile.Name(), envPath)
	defer os.Remove(envPath)

	// Create a minimal config file
	tmpFile, err := os.CreateTemp(envDir, "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testConfig := `r2:
  access_key: "file-access-key"
  secret_key: "file-secret-key"
  endpoint: "https://file.r2.cloudflarestorage.com"
  bucket: "file-bucket"
server:
  port: 8080
`

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	// Load config (should load .env file automatically)
	config, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// .env file values should override YAML values
	if config.R2.AccessKey != "env-file-access-key" {
		t.Errorf("Expected .env access_key 'env-file-access-key', got '%s'", config.R2.AccessKey)
	}
	if config.R2.SecretKey != "env-file-secret-key" {
		t.Errorf("Expected .env secret_key 'env-file-secret-key', got '%s'", config.R2.SecretKey)
	}
	if config.R2.Endpoint != "https://env-file.r2.cloudflarestorage.com" {
		t.Errorf("Expected .env endpoint 'https://env-file.r2.cloudflarestorage.com', got '%s'", config.R2.Endpoint)
	}
	if config.R2.Bucket != "env-file-bucket" {
		t.Errorf("Expected .env bucket 'env-file-bucket', got '%s'", config.R2.Bucket)
	}
	if config.Server.Port != 9000 {
		t.Errorf("Expected .env port 9000, got %d", config.Server.Port)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				R2: R2Config{
					AccessKey: "key",
					SecretKey: "secret",
					Endpoint:  "https://test.r2.cloudflarestorage.com",
					Bucket:    "bucket",
				},
				Conversion: ConversionConfig{
					Quality:   85,
					MaxSizeMB: 50,
				},
				Server: ServerConfig{
					Port:           8080,
					TimeoutSeconds: 30,
				},
			},
			wantErr: false,
		},
		{
			name: "missing access_key",
			config: &Config{
				R2: R2Config{
					SecretKey: "secret",
					Endpoint:  "https://test.r2.cloudflarestorage.com",
					Bucket:    "bucket",
				},
			},
			wantErr: true,
			errMsg:  "r2.access_key",
		},
		{
			name: "invalid quality",
			config: &Config{
				R2: R2Config{
					AccessKey: "key",
					SecretKey: "secret",
					Endpoint:  "https://test.r2.cloudflarestorage.com",
					Bucket:    "bucket",
				},
				Conversion: ConversionConfig{
					Quality: 150,
				},
			},
			wantErr: true,
			errMsg:  "quality",
		},
		{
			name: "invalid port",
			config: &Config{
				R2: R2Config{
					AccessKey: "key",
					SecretKey: "secret",
					Endpoint:  "https://test.r2.cloudflarestorage.com",
					Bucket:    "bucket",
				},
				Server: ServerConfig{
					Port: 70000,
				},
			},
			wantErr: true,
			errMsg:  "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() == "" {
					t.Errorf("Expected error message containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
