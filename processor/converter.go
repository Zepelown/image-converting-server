package processor

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"

	"image-converting-server/config"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

// Processor handles image conversion and resizing
type Processor struct {
	cfg config.Config
}

// NewProcessor creates a new Processor instance
func NewProcessor(cfg config.Config) *Processor {
	return &Processor{
		cfg: cfg,
	}
}

// Process handles the full image processing flow: decode, resize (if needed), and convert to WebP
func (p *Processor) Process(data []byte, options ProcessOptions) ([]byte, string, error) {
	// 1. Detect format
	contentType := http.DetectContentType(data)
	if !p.isSupported(contentType) {
		return nil, "", fmt.Errorf("unsupported image format: %s", contentType)
	}

	// 2. Decode image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// 3. Resize if options provided
	if options.Width > 0 || options.Height > 0 {
		img = p.ResizeImage(img, options.Width, options.Height)
	} else if options.Preset != "" {
		if preset, ok := p.cfg.Resize.Presets[options.Preset]; ok {
			img = p.ResizeImage(img, preset.Width, preset.Height)
		}
	}

	// 4. Convert to WebP
	webpData, err := p.ConvertToWebP(img)
	if err != nil {
		return nil, "", fmt.Errorf("failed to convert to webp: %w", err)
	}

	return webpData, format, nil
}

// ConvertToWebP encodes an image to WebP format
func (p *Processor) ConvertToWebP(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := webp.Encode(&buf, img, &webp.Options{
		Lossless: false,
		Quality:  float32(p.cfg.Conversion.Quality),
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ResizeImage resizes the image while maintaining aspect ratio if one dimension is 0
func (p *Processor) ResizeImage(img image.Image, width, height int) image.Image {
	// imaging.Resize maintains aspect ratio if one of the dimensions is 0
	return imaging.Resize(img, width, height, imaging.Lanczos)
}

// isSupported checks if the content type is in the supported formats list
func (p *Processor) isSupported(contentType string) bool {
	for _, format := range p.cfg.Conversion.Formats {
		if strings.Contains(strings.ToLower(contentType), strings.ToLower(format)) {
			return true
		}
	}
	// Also allow common mappings if not explicitly in config
	if strings.HasPrefix(contentType, "image/") {
		ext := strings.TrimPrefix(contentType, "image/")
		for _, f := range p.cfg.Conversion.Formats {
			if f == ext || (f == "jpg" && ext == "jpeg") || (f == "jpeg" && ext == "jpg") {
				return true
			}
		}
	}
	return false
}

// ProcessOptions defines resizing parameters for Process method
type ProcessOptions struct {
	Width  int
	Height int
	Preset string
}

// GetImageFormat returns the format of the image data
func GetImageFormat(data []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	return format, nil
}

// GetMimeType returns the MIME type of the image data
func GetMimeType(data []byte) string {
	return http.DetectContentType(data)
}

// StreamToBytes reads an io.Reader into a byte slice
func StreamToBytes(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
