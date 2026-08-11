package api

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"image-converting-server/config"
	"image-converting-server/processor"
	"image-converting-server/r2"
	"image-converting-server/webhook"
)

// mockStorageClient is a mock implementation of the r2.StorageClient interface
type mockStorageClient struct {
	downloadFunc func(ctx context.Context, key string) ([]byte, error)
	uploadFunc   func(ctx context.Context, key string, data []byte, contentType string) error
	listFunc     func(ctx context.Context, since time.Time) ([]string, error)
	testFunc     func(ctx context.Context) error
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
	return m.testFunc(ctx)
}

func (m *mockStorageClient) DeleteObject(ctx context.Context, key string) error {
	return nil
}

func TestHandleHealth(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestHandleIndex(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	h.HandleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["message"] != "Image Converting Server" {
		t.Errorf("expected message 'Image Converting Server', got %s", resp["message"])
	}
}

func TestHandleConvert_R2(t *testing.T) {
	// Setup 1x1 pixel PNG
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	png.Encode(&buf, img)
	imgData := buf.Bytes()

	cfg := &config.Config{
		R2: config.R2Config{Bucket: "test-bucket"},
		Conversion: config.ConversionConfig{
			Formats:   []string{"png", "jpeg"},
			Quality:   80,
			MaxSizeMB: 1,
		},
		Resize: config.ResizeConfig{
			Presets: map[string]config.PresetConfig{
				"thumb": {Width: 100, Height: 100},
			},
		},
	}

	mockStorage := &mockStorageClient{
		downloadFunc: func(ctx context.Context, key string) ([]byte, error) {
			if key != "test.png" {
				t.Errorf("expected key test.png, got %s", key)
			}
			return imgData, nil
		},
		uploadFunc: func(ctx context.Context, key string, data []byte, contentType string) error {
			if key != "test.webp" {
				t.Errorf("expected key test.webp, got %s", key)
			}
			if contentType != "image/webp" {
				t.Errorf("expected content type image/webp, got %s", contentType)
			}
			return nil
		},
	}

	proc := processor.NewProcessor(*cfg)
	h := NewHandler(mockStorage, proc, cfg)

	// Test case: POST /api/convert
	reqBody := ConvertRequest{Source: "r2://test-bucket/test.png"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/convert", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleConvert(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp ConvertResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Error("expected success true")
	}
	if resp.Destination != "r2://test-bucket/test.webp" {
		t.Errorf("unexpected destination: %s", resp.Destination)
	}
}

func TestHandleConvert_R2RoutesByBucket(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	png.Encode(&buf, img)
	imgData := buf.Bytes()

	cfg := &config.Config{
		R2: config.R2Config{Bucket: "images-a", Buckets: []string{"images-a", "images-b"}},
		Conversion: config.ConversionConfig{
			Formats:   []string{"png"},
			Quality:   80,
			MaxSizeMB: 1,
		},
	}

	defaultDownloaded := false
	targetDownloaded := false
	targetUploaded := false
	defaultStorage := &mockStorageClient{
		downloadFunc: func(ctx context.Context, key string) ([]byte, error) {
			defaultDownloaded = true
			return nil, nil
		},
		uploadFunc: func(ctx context.Context, key string, data []byte, contentType string) error {
			t.Fatal("default bucket should not receive upload")
			return nil
		},
	}
	targetStorage := &mockStorageClient{
		downloadFunc: func(ctx context.Context, key string) ([]byte, error) {
			targetDownloaded = true
			if key != "test.png" {
				t.Errorf("expected key test.png, got %s", key)
			}
			return imgData, nil
		},
		uploadFunc: func(ctx context.Context, key string, data []byte, contentType string) error {
			targetUploaded = true
			if key != "test.webp" {
				t.Errorf("expected key test.webp, got %s", key)
			}
			return nil
		},
	}

	proc := processor.NewProcessor(*cfg)
	h := NewMultiBucketHandler(map[string]r2.StorageClient{
		"images-a": defaultStorage,
		"images-b": targetStorage,
	}, proc, cfg)

	reqBody := ConvertRequest{Source: "r2://images-b/test.png"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/convert", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleConvert(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if defaultDownloaded {
		t.Error("default bucket should not receive download")
	}
	if !targetDownloaded || !targetUploaded {
		t.Errorf("expected target bucket download/upload, got download=%v upload=%v", targetDownloaded, targetUploaded)
	}
}

func TestHandleConvert_InvalidSource(t *testing.T) {
	cfg := &config.Config{}
	h := NewHandler(nil, nil, cfg)

	reqBody := ConvertRequest{Source: "invalid-source"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/convert", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleConvert(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "invalid_source_format" {
		t.Errorf("expected error invalid_source_format, got %s", resp.Error)
	}
}

func TestHandleConvert_GET(t *testing.T) {
	// Setup 1x1 pixel PNG
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	png.Encode(&buf, img)
	imgData := buf.Bytes()

	cfg := &config.Config{
		R2: config.R2Config{Bucket: "test-bucket"},
		Conversion: config.ConversionConfig{
			Formats: []string{"png"},
			Quality: 80,
		},
	}

	mockStorage := &mockStorageClient{
		downloadFunc: func(ctx context.Context, key string) ([]byte, error) {
			return imgData, nil
		},
		uploadFunc: func(ctx context.Context, key string, data []byte, contentType string) error {
			return nil
		},
	}

	proc := processor.NewProcessor(*cfg)
	h := NewHandler(mockStorage, proc, cfg)

	req := httptest.NewRequest("GET", "/api/convert?source=r2://test-bucket/test.png&width=50", nil)
	w := httptest.NewRecorder()

	h.HandleConvert(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp ConvertResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Width != 50 {
		t.Errorf("expected width 50, got %d", resp.Width)
	}
}

func TestHandleConvert_URL(t *testing.T) {
	// Setup a mock image server
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{0, 255, 0, 255})
	var buf bytes.Buffer
	png.Encode(&buf, img)
	imgData := buf.Bytes()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(imgData)
	}))
	defer server.Close()

	cfg := &config.Config{
		R2: config.R2Config{Bucket: "test-bucket"},
		Conversion: config.ConversionConfig{
			Formats: []string{"png"},
			Quality: 80,
		},
	}

	mockStorage := &mockStorageClient{
		uploadFunc: func(ctx context.Context, key string, data []byte, contentType string) error {
			if key != "image.webp" {
				t.Errorf("expected key image.webp, got %s", key)
			}
			return nil
		},
	}

	proc := processor.NewProcessor(*cfg)
	h := NewHandler(mockStorage, proc, cfg)

	reqBody := ConvertRequest{Source: server.URL + "/image.png"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/convert", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleConvert(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleTriggerWebhook_NotConfigured(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{URL: ""},
	}
	h := NewHandler(nil, nil, cfg)

	body, _ := json.Marshal(TriggerWebhookRequest{
		Images: []webhook.ImageEntry{
			{Source: "a.jpg", Destination: "a.webp"},
		},
	})
	req := httptest.NewRequest("POST", "/api/webhook/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTriggerWebhook(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "webhook_not_configured" {
		t.Errorf("expected error webhook_not_configured, got %s", resp.Error)
	}
}

func TestHandleTriggerWebhook_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{Webhook: config.WebhookConfig{URL: "https://example.com/hook"}}
	h := NewHandler(nil, nil, cfg)

	req := httptest.NewRequest("GET", "/api/webhook/send", nil)
	w := httptest.NewRecorder()

	h.HandleTriggerWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleTriggerWebhook_EmptyImages(t *testing.T) {
	cfg := &config.Config{Webhook: config.WebhookConfig{URL: "https://example.com/hook"}}
	h := NewHandler(nil, nil, cfg)

	body, _ := json.Marshal(TriggerWebhookRequest{Images: []webhook.ImageEntry{}})
	req := httptest.NewRequest("POST", "/api/webhook/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTriggerWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "empty_images" {
		t.Errorf("expected error empty_images, got %s", resp.Error)
	}
}

func TestHandleTriggerWebhook_Success(t *testing.T) {
	received := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{Webhook: config.WebhookConfig{URL: server.URL}}
	h := NewHandler(nil, nil, cfg)

	reqBody := TriggerWebhookRequest{
		Images: []webhook.ImageEntry{
			{Source: "uploads/a.jpg", Destination: "uploads/a.webp"},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/webhook/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTriggerWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	select {
	case data := <-received:
		var payload struct {
			Event          string `json:"event"`
			ProcessedCount int    `json:"processed_count"`
			FailedCount    int    `json:"failed_count"`
			Images         []struct {
				Source      string `json:"source"`
				Destination string `json:"destination"`
			} `json:"images"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Fatalf("unmarshal webhook body: %v", err)
		}
		if payload.Event != "manual.triggered" {
			t.Errorf("expected event manual.triggered, got %s", payload.Event)
		}
		if payload.ProcessedCount != 1 || payload.FailedCount != 0 {
			t.Errorf("expected processed_count=1 failed_count=0, got %d %d", payload.ProcessedCount, payload.FailedCount)
		}
		if len(payload.Images) != 1 || payload.Images[0].Source != "uploads/a.jpg" || payload.Images[0].Destination != "uploads/a.webp" {
			t.Errorf("unexpected images: %+v", payload.Images)
		}
	default:
		t.Error("webhook server did not receive request")
	}
}
