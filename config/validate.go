package config

import "fmt"

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

	if config.Webhook.RetryEnabled {
		if config.Webhook.RetryIntervalMinutes < 1 {
			return fmt.Errorf("webhook.retry_interval_minutes must be at least 1, got: %d", config.Webhook.RetryIntervalMinutes)
		}
		if config.Webhook.MaxRetries < 0 {
			return fmt.Errorf("webhook.max_retries must be non-negative, got: %d", config.Webhook.MaxRetries)
		}
	}

	return nil
}
