package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"image-converting-server/config"
	"image-converting-server/processor"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/convert_local.go <input_path> [width] [height]")
		fmt.Println("Example: go run scripts/convert_local.go my_image.jpg 800 0")
		return
	}

	inputPath := os.Args[1]
	ext := filepath.Ext(inputPath)
	outputPath := inputPath[:len(inputPath)-len(ext)] + ".webp"

	var width, height int
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &width)
	}
	if len(os.Args) >= 4 {
		fmt.Sscanf(os.Args[3], "%d", &height)
	}

	// Read input file
	data, err := ioutil.ReadFile(inputPath)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	// Create processor with minimal config
	cfg := config.Config{
		Conversion: config.ConversionConfig{
			Formats: []string{"jpeg", "jpg", "png", "gif"},
			Quality: 80,
		},
	}
	p := processor.NewProcessor(cfg)

	// Process image
	options := processor.ProcessOptions{
		Width:  width,
		Height: height,
	}

	fmt.Printf("Converting %s to %s...\n", inputPath, outputPath)
	if width > 0 || height > 0 {
		fmt.Printf("Resizing to: %dx%d (0 means auto-ratio)\n", width, height)
	}

	webpData, format, err := p.Process(data, options)
	if err != nil {
		log.Fatalf("Processing failed: %v", err)
	}

	// Write output file
	err = ioutil.WriteFile(outputPath, webpData, 0644)
	if err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	fmt.Printf("Successfully converted! (Original Format: %s)\n", format)
	fmt.Printf("Output saved to: %s\n", outputPath)
}
