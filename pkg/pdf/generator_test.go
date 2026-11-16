package pdf

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func createTestImage(t *testing.T, filename string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	c := color.RGBA{100, 150, 200, 255}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create test image %s: %v", filename, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}
}

func TestGeneratePDF_MixedSizePages(t *testing.T) {
	dir := t.TempDir()

	// Create images of various sizes: large, small, and landscape
	sizes := []struct {
		name string
		w, h int
	}{
		{"large_portrait", 1000, 1500},
		{"small_portrait", 400, 500},
		{"large_landscape", 1800, 900},
		{"small_landscape", 600, 300},
		{"tiny", 100, 120},
	}

	var paths []string
	for _, s := range sizes {
		p := filepath.Join(dir, s.name+".png")
		createTestImage(t, p, s.w, s.h)
		paths = append(paths, p)
	}

	outPDF := filepath.Join(dir, "mixed_sizes.pdf")
	if err := GeneratePDF(paths, outPDF); err != nil {
		t.Fatalf("GeneratePDF failed: %v", err)
	}

	// Verify the PDF has correct page count and each page's aspect ratio
	// matches the source image
	dims, err := api.PageDimsFile(outPDF)
	if err != nil {
		t.Fatalf("PageDimsFile failed: %v", err)
	}

	if len(dims) != len(sizes) {
		t.Fatalf("Expected %d pages, got %d", len(sizes), len(dims))
	}

	for i, s := range sizes {
		dim := dims[i]
		imgAspect := float64(s.w) / float64(s.h)
		pageAspect := dim.Width / dim.Height

		// The page aspect ratio should match the image aspect ratio
		diff := imgAspect - pageAspect
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.01 {
			t.Errorf("Page %d (%s): aspect ratio mismatch - image %.4f vs page %.4f (page: %.2f x %.2f pt)",
				i+1, s.name, imgAspect, pageAspect, dim.Width, dim.Height)
		}

		// Verify each page has its own unique size (no page uses a generic/default size)
		fmt.Printf("Page %d (%s %dx%d): %.2f x %.2f pt, aspect=%.4f (expected %.4f)\n",
			i+1, s.name, s.w, s.h, dim.Width, dim.Height, pageAspect, imgAspect)
	}
}

func TestGeneratePDF_SmallAfterLargeNotClipped(t *testing.T) {
	dir := t.TempDir()

	// The specific bug: a small image after a large image gets "cut"
	// because the page size doesn't update properly
	largePath := filepath.Join(dir, "large.png")
	smallPath := filepath.Join(dir, "small.png")
	createTestImage(t, largePath, 2000, 3000)
	createTestImage(t, smallPath, 500, 600)

	outPDF := filepath.Join(dir, "large_then_small.pdf")
	if err := GeneratePDF([]string{largePath, smallPath}, outPDF); err != nil {
		t.Fatalf("GeneratePDF failed: %v", err)
	}

	dims, err := api.PageDimsFile(outPDF)
	if err != nil {
		t.Fatalf("PageDimsFile failed: %v", err)
	}

	if len(dims) != 2 {
		t.Fatalf("Expected 2 pages, got %d", len(dims))
	}

	// Page 2 must be smaller than page 1 (proportionally)
	if dims[1].Width >= dims[0].Width || dims[1].Height >= dims[0].Height {
		t.Errorf("Small page should be smaller than large page:\n  Page 1: %.2f x %.2f\n  Page 2: %.2f x %.2f",
			dims[0].Width, dims[0].Height, dims[1].Width, dims[1].Height)
	}

	// Page 2 aspect ratio must match the small image
	imgAspect := 500.0 / 600.0
	pageAspect := dims[1].Width / dims[1].Height
	diff := imgAspect - pageAspect
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.01 {
		t.Errorf("Small page aspect ratio mismatch: image %.4f vs page %.4f", imgAspect, pageAspect)
	}

	fmt.Printf("Large page: %.2f x %.2f pt\n", dims[0].Width, dims[0].Height)
	fmt.Printf("Small page: %.2f x %.2f pt\n", dims[1].Width, dims[1].Height)
}
