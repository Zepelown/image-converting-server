package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func LoadFromEnv() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config

	loadR2FromEnv(&cfg)
	loadConversionFromEnv(&cfg)
	loadResizeFromEnv(&cfg)
	loadCronFromEnv(&cfg)
	loadServerFromEnv(&cfg)
	loadWebhookFromEnv(&cfg)

	setDefaults(&cfg)
	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadR2FromEnv(cfg *Config) {
	if v := os.Getenv("R2_ACCESS_KEY"); v != "" {
		cfg.R2.AccessKey = v
	}
	if v := os.Getenv("R2_SECRET_KEY"); v != "" {
		cfg.R2.SecretKey = v
	}
	if v := os.Getenv("R2_ENDPOINT"); v != "" {
		cfg.R2.Endpoint = v
	}
	if v := os.Getenv("R2_BUCKET"); v != "" {
		cfg.R2.Bucket = v
	}
}

func loadConversionFromEnv(cfg *Config) {
	if s := os.Getenv("CONVERSION_FORMATS"); s != "" {
		parts := strings.Split(s, ",")
		var formats []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				formats = append(formats, p)
			}
		}
		if len(formats) > 0 {
			cfg.Conversion.Formats = formats
		}
	}

	if v := os.Getenv("CONVERSION_QUALITY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Conversion.Quality = n
		}
	}

	if v := os.Getenv("CONVERSION_MAX_SIZE_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Conversion.MaxSizeMB = n
		}
	}
}

func loadResizeFromEnv(cfg *Config) {
	s := os.Getenv("RESIZE_PRESETS")
	if s == "" {
		return
	}

	if cfg.Resize.Presets == nil {
		cfg.Resize.Presets = make(map[string]PresetConfig)
	}

	items := strings.Split(s, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		size := strings.TrimSpace(parts[1])
		if name == "" || size == "" {
			continue
		}

		dims := strings.SplitN(size, "x", 2)
		if len(dims) != 2 {
			continue
		}

		w, err1 := strconv.Atoi(strings.TrimSpace(dims[0]))
		h, err2 := strconv.Atoi(strings.TrimSpace(dims[1]))
		if err1 != nil || err2 != nil {
			continue
		}

		cfg.Resize.Presets[name] = PresetConfig{
			Width:  w,
			Height: h,
		}
	}
}

func loadCronFromEnv(cfg *Config) {
	if v := os.Getenv("CRON_SCHEDULE"); v != "" {
		cfg.Cron.Schedule = v
	}

	if v := os.Getenv("CRON_ENABLED"); v != "" {
		if v == "true" || v == "1" {
			cfg.Cron.Enabled = true
		} else if v == "false" || v == "0" {
			cfg.Cron.Enabled = false
		}
	}
}

func loadServerFromEnv(cfg *Config) {
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = n
		}
	}

	if v := os.Getenv("SERVER_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Server.TimeoutSeconds = n
		}
	}
}

func loadWebhookFromEnv(cfg *Config) {
	if v := os.Getenv("WEBHOOK_URL"); v != "" {
		cfg.Webhook.URL = v
	}

	if v := os.Getenv("WEBHOOK_RETRY_ENABLED"); v != "" {
		if v == "true" || v == "1" {
			cfg.Webhook.RetryEnabled = true
		} else if v == "false" || v == "0" {
			cfg.Webhook.RetryEnabled = false
		}
	}

	if v := os.Getenv("WEBHOOK_RETRY_INTERVAL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			cfg.Webhook.RetryIntervalMinutes = n
		}
	}

	if v := os.Getenv("WEBHOOK_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.Webhook.MaxRetries = n
		}
	}

	if v := os.Getenv("WEBHOOK_PENDING_DIR"); v != "" {
		cfg.Webhook.PendingDir = v
	}
}
