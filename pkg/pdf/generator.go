package pdf

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-pdf/fpdf"
)

// ImageFile represents a found image
type ImageFile struct {
	Path    string
	Name    string
}

// GeneratePDF creates a PDF from a list of image file paths
func GeneratePDF(filePaths []string, outputFile string) error {
	var images []ImageFile

	// 1. Validate files
	for _, path := range filePaths {
		_, err := os.Stat(path)
		if err != nil {
			// Skip files that can't be read
			continue
		}
		images = append(images, ImageFile{
			Path:    path,
			Name:    filepath.Base(path),
		})
	}

	if len(images) == 0 {
		return fmt.Errorf("no valid images found in provided paths")
	}

	// 3. Create PDF
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)

	for _, img := range images {
		// Read raw pixel dimensions using Go's image package
		// to avoid any DPI/scale issues from fpdf's RegisterImage
		f, err := os.Open(img.Path)
		if err != nil {
			continue
		}
		cfg, _, err := image.DecodeConfig(f)
		f.Close()
		if err != nil {
			continue
		}

		pixW := float64(cfg.Width)
		pixH := float64(cfg.Height)

		// Convert pixels to mm at a fixed DPI so pages have reasonable
		// physical dimensions. The exact DPI doesn't matter because
		// we size the page and the image identically — the image always
		// fills the page perfectly with no clipping.
		const dpi = 150.0
		pageW := pixW / dpi * 25.4 // pixels → inches → mm
		pageH := pixH / dpi * 25.4

		pdf.AddPageFormat("P", fpdf.SizeType{Wd: pageW, Ht: pageH})

		// Place image at (0,0) filling the entire page
		pdf.Image(img.Path, 0, 0, pageW, pageH, false, "", 0, "")
	}

	if err := pdf.OutputFileAndClose(outputFile); err != nil {
		return fmt.Errorf("failed to save PDF: %w", err)
	}

	return nil
}

// ScanDirectory finds image files in a directory and its subdirectories
func ScanDirectory(root string) ([]string, error) {
	var filePaths []string
	
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return filePaths, nil
}