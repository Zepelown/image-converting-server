package config

import "strings"

// setDefaults sets default values for optional configuration fields
func setDefaults(config *Config) {
	// R2 defaults/compatibility
	config.R2.Bucket = strings.TrimSpace(config.R2.Bucket)
	config.R2.Buckets = normalizeBuckets(config.R2.Buckets)
	if len(config.R2.Buckets) == 0 && config.R2.Bucket != "" {
		config.R2.Buckets = []string{config.R2.Bucket}
	}
	if config.R2.Bucket == "" && len(config.R2.Buckets) > 0 {
		config.R2.Bucket = config.R2.Buckets[0]
	}

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

	// Webhook retry defaults (set retry_enabled: true in yaml or WEBHOOK_RETRY_ENABLED=true to enable)
	if config.Webhook.RetryIntervalMinutes == 0 {
		config.Webhook.RetryIntervalMinutes = 5
	}
	if config.Webhook.MaxRetries == 0 {
		config.Webhook.MaxRetries = 5
	}
	if config.Webhook.PendingDir == "" {
		config.Webhook.PendingDir = "data/webhook_pending"
	}
}
