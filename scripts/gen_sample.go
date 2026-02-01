package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

func main() {
	width, height := 400, 300
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				uint8((x * 255) / width),
				uint8((y * 255) / height),
				150,
				255,
			})
		}
	}

	f, err := os.Create("sample.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	log.Println("Created sample.png")
}
