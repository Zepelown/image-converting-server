package cron

import (
	"context"
	"image-converting-server/config"
	"image-converting-server/processor"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mockStorageClient is a mock for R2 StorageClient
type mockStorageClient struct {
	downloadFunc func(ctx context.Context, key string) ([]byte, error)
	uploadFunc   func(ctx context.Context, key string, data []byte, contentType string) error
	listFunc     func(ctx context.Context, since time.Time) ([]string, error)
}

func (m *mockStorageClient) DownloadImage(ctx context.Context, key string) ([]byte, error) {
	return m.downloadFunc(ctx, key)
}
func (m *mockStorageClient) UploadImage(ctx context.Context, key string, data []byte, contentType string) error {
	return m.uploadFunc(ctx, key, data, contentType)
}
func (m *mockStorageClient) ListObjects(ctx context.Context, since time.Time) ([]string, error) {
	return m.listFunc(ctx, since)
}
func (m *mockStorageClient) TestConnection(ctx context.Context) error {
	return nil
}
func (m *mockStorageClient) DeleteObject(ctx context.Context, key string) error {
	return nil
}

func TestProcessImages(t *testing.T) {
	// 1. Setup temp directory and state file
	tempDir, err := os.MkdirTemp("", "cron_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	statePath := filepath.Join(tempDir, "state.json")

	// 2. Setup config
	cfg := &config.Config{
		Conversion: config.ConversionConfig{
			Formats: []string{"jpg", "png"},
			Quality: 85,
		},
		Cron: config.CronConfig{
			Enabled:  true,
			Schedule: "0 0 * * *",
		},
	}

	// 3. Setup Processor
	proc := processor.NewProcessor(*cfg)

	// 4. Setup Mock R2 Client
	// 1x1 transparent PNG
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}

	uploadedKeys := make(map[string]bool)
	r2Mock := &mockStorageClient{
		listFunc: func(ctx context.Context, since time.Time) ([]string, error) {
			return []string{"image1.jpg", "image2.png", "image3.webp", "other.txt"}, nil
		},
		downloadFunc: func(ctx context.Context, key string) ([]byte, error) {
			return pngData, nil
		},
		uploadFunc: func(ctx context.Context, key string, data []byte, contentType string) error {
			uploadedKeys[key] = true
			return nil
		},
	}

	// 5. Create Job and Run
	job := NewJob(cfg, r2Mock, proc, statePath)
	job.ProcessImages()

	// 6. Verify results
	if !uploadedKeys["image1.webp"] {
		t.Errorf("expected image1.webp to be uploaded")
	}
	if !uploadedKeys["image2.webp"] {
		t.Errorf("expected image2.webp to be uploaded")
	}
	if uploadedKeys["image3.webp"] {
		t.Errorf("did not expect image3.webp to be re-uploaded")
	}
	if uploadedKeys["other.webp"] {
		t.Errorf("did not expect non-image file to be converted")
	}
}

func TestLocking(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cron_lock_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	statePath := filepath.Join(tempDir, "state.json")

	cfg := &config.Config{
		Cron: config.CronConfig{Enabled: true},
	}
	job := NewJob(cfg, nil, nil, statePath)

	// Acquire lock manually
	if err := job.acquireLock(); err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}

	// Try to acquire again
	err = job.acquireLock()
	if err == nil {
		t.Error("Should have failed to acquire second lock")
	}

	// Release and try again
	job.releaseLock()
	err = job.acquireLock()
	if err != nil {
		t.Errorf("Should have acquired lock after release: %v", err)
	}
	job.releaseLock()
}
