package processor

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"image-converting-server/config"
)

// createTestImage creates a simple colored rectangle for testing
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 0, 255})
		}
	}
	return img
}

// imageToBytes converts an image to a byte slice in the specified format
func imageToBytes(t *testing.T, img image.Image, format string) []byte {
	var buf bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, img, nil)
	case "png":
		err = png.Encode(&buf, img)
	default:
		t.Fatalf("unsupported format for test: %s", format)
	}
	if err != nil {
		t.Fatalf("failed to encode image: %v", err)
	}
	return buf.Bytes()
}

func TestProcessor_Process(t *testing.T) {
	cfg := config.Config{
		Conversion: config.ConversionConfig{
			Formats: []string{"jpeg", "png"},
			Quality: 75,
		},
		Resize: config.ResizeConfig{
			Presets: map[string]config.PresetConfig{
				"thumbnail": {Width: 100, Height: 100},
			},
		},
	}
	p := NewProcessor(cfg)

	t.Run("Convert JPEG to WebP", func(t *testing.T) {
		img := createTestImage(200, 200)
		data := imageToBytes(t, img, "jpeg")

		webpData, format, err := p.Process(data, ProcessOptions{})
		if err != nil {
			t.Errorf("Process failed: %v", err)
		}
		if format != "jpeg" {
			t.Errorf("expected format jpeg, got %s", format)
		}
		if len(webpData) == 0 {
			t.Error("webpData is empty")
		}

		// Verify result is WebP
		mime := GetMimeType(webpData)
		if mime != "image/webp" {
			t.Errorf("expected image/webp result, got %s", mime)
		}
	})

	t.Run("Resize with dimensions", func(t *testing.T) {
		img := createTestImage(400, 200)
		data := imageToBytes(t, img, "png")

		webpData, _, err := p.Process(data, ProcessOptions{Width: 200, Height: 0}) // Maintain aspect ratio
		if err != nil {
			t.Errorf("Process failed: %v", err)
		}

		// Decode the result to check size
		resultImg, _, err := image.Decode(bytes.NewReader(webpData))
		if err != nil {
			t.Fatalf("failed to decode result: %v", err)
		}

		bounds := resultImg.Bounds()
		if bounds.Dx() != 200 {
			t.Errorf("expected width 200, got %d", bounds.Dx())
		}
		if bounds.Dy() != 100 {
			t.Errorf("expected height 100 (aspect ratio), got %d", bounds.Dy())
		}
	})

	t.Run("Resize with preset", func(t *testing.T) {
		img := createTestImage(300, 300)
		data := imageToBytes(t, img, "jpeg")

		webpData, _, err := p.Process(data, ProcessOptions{Preset: "thumbnail"})
		if err != nil {
			t.Errorf("Process failed: %v", err)
		}

		resultImg, _, err := image.Decode(bytes.NewReader(webpData))
		if err != nil {
			t.Fatalf("failed to decode result: %v", err)
		}

		bounds := resultImg.Bounds()
		if bounds.Dx() != 100 || bounds.Dy() != 100 {
			t.Errorf("expected 100x100, got %dx%d", bounds.Dx(), bounds.Dy())
		}
	})

	t.Run("Unsupported format", func(t *testing.T) {
		data := []byte("this is not an image")
		_, _, err := p.Process(data, ProcessOptions{})
		if err == nil {
			t.Error("expected error for unsupported format, got nil")
		}
	})
}

func TestGetMimeType(t *testing.T) {
	img := createTestImage(10, 10)
	jpegData := imageToBytes(t, img, "jpeg")
	pngData := imageToBytes(t, img, "png")

	if GetMimeType(jpegData) != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", GetMimeType(jpegData))
	}
	if GetMimeType(pngData) != "image/png" {
		t.Errorf("expected image/png, got %s", GetMimeType(pngData))
	}
}
