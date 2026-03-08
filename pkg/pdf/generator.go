package pdf

import (
	"fmt"
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
		// Get image info to calculate dimensions
		info := pdf.RegisterImage(img.Path, "")
		if info == nil {
			continue // Skip invalid images
		}
		
		imgW := info.Width()
		imgH := info.Height()
		
		// Create a page that EXACTLY matches the image's aspect ratio.
		// We'll use 210mm as the base width for Portrait and 297mm for Landscape
		// to keep the physical size reasonable, but the aspect ratio will be perfect.
		var pageW, pageH float64
		orientation := "P"
		
		if imgW > imgH {
			orientation = "L"
			pageW = 297.0
			pageH = (imgH / imgW) * 297.0
		} else {
			orientation = "P"
			pageW = 210.0
			pageH = (imgH / imgW) * 210.0
		}
		
		pdf.AddPageFormat(orientation, fpdf.SizeType{Wd: pageW, Ht: pageH})

		// Place image at (0,0) with full page dimensions
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