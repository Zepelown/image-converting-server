package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"image-converting-server/config"
	"image-converting-server/processor"
	"image-converting-server/r2"
)

// ConvertRequest represents the JSON body for POST /api/convert
type ConvertRequest struct {
	Source string `json:"source"`
}

// ConvertResponse represents the success response for /api/convert
type ConvertResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	Source        string `json:"source"`
	Destination   string `json:"destination"`
	OriginalSize  int    `json:"original_size"`
	ConvertedSize int    `json:"converted_size"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
}

// ErrorResponse represents the error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	storageClient r2.StorageClient
	processor     *processor.Processor
	config        *config.Config
}

// NewHandler creates a new Handler instance
func NewHandler(storageClient r2.StorageClient, processor *processor.Processor, config *config.Config) *Handler {
	return &Handler{
		storageClient: storageClient,
		processor:     processor,
		config:        config,
	}
}

// HandleIndex handles GET /
func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.sendError(w, http.StatusNotFound, "not_found", "The requested resource was not found")
		return
	}
	h.sendJSON(w, http.StatusOK, map[string]string{"message": "Image Converting Server"})
}

// HandleHealth handles GET /health
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	h.sendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleConvert handles GET and POST /api/convert
func (h *Handler) HandleConvert(w http.ResponseWriter, r *http.Request) {
	var source string
	var options processor.ProcessOptions

	// 1. Parse request based on method
	switch r.Method {
	case http.MethodGet:
		source = r.URL.Query().Get("source")
	case http.MethodPost:
		var req ConvertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.sendError(w, http.StatusBadRequest, "invalid_request", "Failed to parse JSON body")
			return
		}
		source = req.Source
	default:
		w.Header().Set("Allow", "GET, POST")
		h.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	if source == "" {
		h.sendError(w, http.StatusBadRequest, "missing_source", "The 'source' parameter is required")
		return
	}

	// 2. Parse resizing parameters from query string
	query := r.URL.Query()
	if widthStr := query.Get("width"); widthStr != "" {
		width, err := strconv.Atoi(widthStr)
		if err != nil || width < 0 {
			h.sendError(w, http.StatusBadRequest, "invalid_resize_params", "Invalid 'width' parameter")
			return
		}
		options.Width = width
	}
	if heightStr := query.Get("height"); heightStr != "" {
		height, err := strconv.Atoi(heightStr)
		if err != nil || height < 0 {
			h.sendError(w, http.StatusBadRequest, "invalid_resize_params", "Invalid 'height' parameter")
			return
		}
		options.Height = height
	}
	if preset := query.Get("preset"); preset != "" {
		if _, ok := h.config.Resize.Presets[preset]; !ok {
			h.sendError(w, http.StatusBadRequest, "invalid_preset", fmt.Sprintf("Preset '%s' not found", preset))
			return
		}
		options.Preset = preset
	}

	// 3. Download image
	var data []byte
	var err error
	var r2Key string

	if strings.HasPrefix(source, "r2://") {
		// Format: r2://bucket/key
		r2Key = strings.TrimPrefix(source, "r2://")
		parts := strings.SplitN(r2Key, "/", 2)
		if len(parts) < 2 {
			h.sendError(w, http.StatusBadRequest, "invalid_source_format", "Invalid R2 source format. Expected r2://bucket/key")
			return
		}
		// In this version, we ignore the bucket name and use the configured one
		// but we keep the key part
		r2Key = parts[1]
		data, err = h.storageClient.DownloadImage(r.Context(), r2Key)
		if err != nil {
			log.Printf("Failed to download from R2: %v", err)
			h.sendError(w, http.StatusNotFound, "image_not_found", "Image not found in R2 bucket")
			return
		}
	} else if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		data, err = h.downloadFromURL(source)
		if err != nil {
			log.Printf("Failed to download from URL: %v", err)
			h.sendError(w, http.StatusNotFound, "url_not_accessible", "Source URL is not accessible")
			return
		}
	} else {
		h.sendError(w, http.StatusBadRequest, "invalid_source_format", "Source must be either r2://bucket/key or http(s):// URL")
		return
	}

	originalSize := len(data)

	// 4. Process image
	webpData, _, err := h.processor.Process(data, options)
	if err != nil {
		log.Printf("Conversion failed: %v", err)
		h.sendError(w, http.StatusInternalServerError, "conversion_failed", fmt.Sprintf("Failed to convert image: %v", err))
		return
	}

	// 5. Upload to R2 (if it was an R2 source, replace it; if URL, create a new one based on URL path)
	if r2Key == "" {
		// For URL source, generate a key
		u, _ := url.Parse(source)
		r2Key = strings.TrimLeft(u.Path, "/")
		if r2Key == "" {
			r2Key = "downloaded_image"
		}
	}

	// Change extension to .webp
	ext := filepath.Ext(r2Key)
	destKey := strings.TrimSuffix(r2Key, ext) + ".webp"

	err = h.storageClient.UploadImage(r.Context(), destKey, webpData, "image/webp")
	if err != nil {
		log.Printf("Upload failed: %v", err)
		h.sendError(w, http.StatusInternalServerError, "upload_failed", "Failed to upload converted image to R2")
		return
	}

	// 6. Delete original image if it was from R2 and destination is different
	/*
		if strings.HasPrefix(source, "r2://") && r2Key != destKey {
			err = h.storageClient.DeleteObject(r.Context(), r2Key)
			if err != nil {
				log.Printf("[WARN] Failed to delete original image %s: %v", r2Key, err)
				// Don't return error here, as conversion was successful
			} else {
				log.Printf("[INFO] Deleted original image: %s", r2Key)
			}
		}
	*/

	// 7. Return response
	res := ConvertResponse{
		Success:       true,
		Message:       "Image converted successfully",
		Source:        source,
		Destination:   fmt.Sprintf("r2://%s/%s", h.config.R2.Bucket, destKey),
		OriginalSize:  originalSize,
		ConvertedSize: len(webpData),
	}

	// Add actual dimensions if possible
	// (Though processor.Process doesn't return them currently, we could decode config again)
	// For simplicity, we just use the options if they were set
	if options.Width > 0 {
		res.Width = options.Width
	}
	if options.Height > 0 {
		res.Height = options.Height
	} else if options.Preset != "" {
		if p, ok := h.config.Resize.Presets[options.Preset]; ok {
			res.Width = p.Width
			res.Height = p.Height
		}
	}

	h.sendJSON(w, http.StatusOK, res)
}

func (h *Handler) downloadFromURL(urlStr string) ([]byte, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendError(w http.ResponseWriter, status int, code, message string) {
	h.sendJSON(w, status, ErrorResponse{
		Success: false,
		Error:   code,
		Message: message,
	})
}
