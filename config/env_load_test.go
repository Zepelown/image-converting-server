package config

import "testing"

func TestLoadFromEnv_Minimal(t *testing.T) {
	t.Setenv("R2_ACCESS_KEY", "key")
	t.Setenv("R2_SECRET_KEY", "secret")
	t.Setenv("R2_ENDPOINT", "https://example.r2.cloudflarestorage.com")
	t.Setenv("R2_BUCKET", "bucket")
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.R2.AccessKey != "key" {
		t.Errorf("R2.AccessKey = %q, want %q", cfg.R2.AccessKey, "key")
	}
}

func TestLoadFromEnv_ConversionFromEnv(t *testing.T) {
	t.Setenv("R2_ACCESS_KEY", "key")
	t.Setenv("R2_SECRET_KEY", "secret")
	t.Setenv("R2_ENDPOINT", "https://example")
	t.Setenv("R2_BUCKET", "bucket")

	t.Setenv("CONVERSION_FORMATS", "jpeg,png")
	t.Setenv("CONVERSION_QUALITY", "90")
	t.Setenv("CONVERSION_MAX_SIZE_MB", "100")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if len(cfg.Conversion.Formats) != 2 {
		t.Fatalf("Formats len = %d, want 2", len(cfg.Conversion.Formats))
	}
	if cfg.Conversion.Quality != 90 {
		t.Errorf("Quality = %d, want 90", cfg.Conversion.Quality)
	}
	if cfg.Conversion.MaxSizeMB != 100 {
		t.Errorf("MaxSizeMB = %d, want 100", cfg.Conversion.MaxSizeMB)
	}
}
