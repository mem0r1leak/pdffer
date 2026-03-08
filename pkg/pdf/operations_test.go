package pdf

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-pdf/fpdf"
)

func createDummyPDF(t *testing.T, filename string, text string) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, text)
	if err := pdf.OutputFileAndClose(filename); err != nil {
		t.Fatalf("Failed to create dummy PDF %s: %v", filename, err)
	}
}

func createDummyImage(t *testing.T, filename string) {
	width := 200
	height := 100
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	cyan := color.RGBA{100, 200, 200, 255}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, cyan)
		}
	}

	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create dummy image %s: %v", filename, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode dummy image: %v", err)
	}
}

func TestMergePDFs(t *testing.T) {
	dir := t.TempDir()
	pdf1 := filepath.Join(dir, "1.pdf")
	createDummyPDF(t, pdf1, "Page 1")

	pdf2 := filepath.Join(dir, "2.pdf")
	createDummyPDF(t, pdf2, "Page 2")

	out := filepath.Join(dir, "merged.pdf")

	if err := MergePDFs([]string{pdf1, pdf2}, out); err != nil {
		t.Fatalf("MergePDFs failed: %v", err)
	}

	if _, err := os.Stat(out); os.IsNotExist(err) {
		t.Errorf("Output file %s not created", out)
	}
}

func TestAppendImagesToPDF(t *testing.T) {
	dir := t.TempDir()
	basePDF := filepath.Join(dir, "base.pdf")
	createDummyPDF(t, basePDF, "Base PDF")

	img1 := filepath.Join(dir, "img1.png")
	createDummyImage(t, img1)

	out := filepath.Join(dir, "appended.pdf")

	if err := AppendImagesToPDF(basePDF, []string{img1}, out); err != nil {
		t.Fatalf("AppendImagesToPDF failed: %v", err)
	}

	if _, err := os.Stat(out); os.IsNotExist(err) {
		t.Errorf("Output file %s not created", out)
	}
}
