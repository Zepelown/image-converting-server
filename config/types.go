package config

// Config represents the entire configuration structure
type Config struct {
	R2         R2Config         `yaml:"r2"`
	Conversion ConversionConfig `yaml:"conversion"`
	Resize     ResizeConfig     `yaml:"resize"`
	Cron       CronConfig       `yaml:"cron"`
	Server     ServerConfig     `yaml:"server"`
	Webhook    WebhookConfig    `yaml:"webhook"`
}

// WebhookConfig contains webhook settings (URL is typically set via WEBHOOK_URL env only)
type WebhookConfig struct {
	URL                  string `yaml:"url"`
	RetryEnabled         bool   `yaml:"retry_enabled"`
	RetryIntervalMinutes int    `yaml:"retry_interval_minutes"`
	MaxRetries           int    `yaml:"max_retries"`
	PendingDir           string `yaml:"pending_dir"`
}

// R2Config contains Cloudflare R2 connection settings
type R2Config struct {
	AccessKey string   `yaml:"access_key"`
	SecretKey string   `yaml:"secret_key"`
	Endpoint  string   `yaml:"endpoint"`
	Bucket    string   `yaml:"bucket"`
	Buckets   []string `yaml:"buckets"`
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
