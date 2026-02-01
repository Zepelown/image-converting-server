package api

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"image-converting-server/config"
	"image-converting-server/processor"
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
