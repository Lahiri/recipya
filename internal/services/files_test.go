package services

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"github.com/gen2brain/webp"
	"github.com/google/uuid"
	"github.com/reaper47/recipya/internal/app"
	"github.com/reaper47/recipya/internal/models"
)

func TestRecipeToPDF(t *testing.T) {
	tempDir := t.TempDir()
	oldImagesDir := app.ImagesDir
	app.ImagesDir = tempDir
	t.Cleanup(func() { app.ImagesDir = oldImagesDir })

	presentImage := uuid.New()
	writeTestWebP(t, filepath.Join(tempDir, presentImage.String()+app.ImageExt))
	corruptImage := uuid.New()
	err := os.WriteFile(filepath.Join(tempDir, corruptImage.String()+app.ImageExt), []byte("not an image"), 0o600)
	if err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	tests := []struct {
		name      string
		images    []uuid.UUID
		wantImage bool
	}{
		{name: "no image"},
		{name: "missing image", images: []uuid.UUID{uuid.New()}},
		{name: "corrupt image", images: []uuid.UUID{corruptImage}},
		{name: "present image", images: []uuid.UUID{presentImage}, wantImage: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipe := &models.Recipe{
				Name:         "Pasta al Pomodoro",
				Category:     "Dinner",
				Description:  "A simple tomato pasta.",
				Images:       tt.images,
				Ingredients:  []string{"200g pasta", "100g tomato sauce"},
				Instructions: []string{"Boil pasta.", "Toss with sauce."},
				URL:          "https://example.com/pasta",
				Yield:        2,
			}

			data := recipeToPDF(recipe)
			if !bytes.HasPrefix(data, []byte("%PDF")) {
				t.Fatalf("recipeToPDF() did not return PDF data")
			}

			hasImage := bytes.Contains(data, []byte("/Subtype /Image"))
			if hasImage != tt.wantImage {
				t.Fatalf("recipeToPDF() image presence = %t; want %t", hasImage, tt.wantImage)
			}
		})
	}
}

func writeTestWebP(t *testing.T, path string) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 320, 180))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 200, G: 62, B: 42, A: 255}}, image.Point{}, draw.Src)

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create() error = %v", err)
	}
	defer file.Close()

	err = webp.Encode(file, img, webp.Options{Quality: 80})
	if err != nil {
		t.Fatalf("webp.Encode() error = %v", err)
	}
}
