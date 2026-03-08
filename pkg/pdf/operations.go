package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// MergePDFs combines multiple PDF files into a single output file
func MergePDFs(inputFiles []string, outputFile string) error {
	if len(inputFiles) < 2 {
		return fmt.Errorf("need at least 2 files to merge")
	}

	// Verify all inputs exist and are PDFs
	for _, f := range inputFiles {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("file not found: %s", f)
		}
		if !strings.HasSuffix(strings.ToLower(f), ".pdf") {
			return fmt.Errorf("file is not a PDF: %s", f)
		}
	}

	// Create configuration (nil means default)
	var conf *model.Configuration = nil

	// Perform the merge
	// api.MergeCreateFile takes (outputFile, inputFiles, config)
	err := api.MergeCreateFile(inputFiles, outputFile, false, conf)
	if err != nil {
		return fmt.Errorf("failed to merge PDFs: %w", err)
	}

	return nil
}

// AppendImagesToPDF adds images to the end of an existing PDF
// It does this by generating a temporary PDF from the images and merging it
func AppendImagesToPDF(basePDFPath string, imagePaths []string, outputFile string) error {
	// 1. Verify base PDF
	if _, err := os.Stat(basePDFPath); err != nil {
		return fmt.Errorf("base PDF not found: %s", basePDFPath)
	}

	if len(imagePaths) == 0 {
		return fmt.Errorf("no images provided to append")
	}

	// 2. Generate a temporary PDF for the images
	tempPDF := filepath.Join(os.TempDir(), fmt.Sprintf("temp_images_%d.pdf", os.Getpid()))
	defer os.Remove(tempPDF) // Clean up temp file

	err := GeneratePDF(imagePaths, tempPDF)
	if err != nil {
		return fmt.Errorf("failed to generate temp PDF from images: %w", err)
	}

	// 3. Merge [basePDF, tempPDF] -> outputFile
	return MergePDFs([]string{basePDFPath, tempPDF}, outputFile)
}

// ScanPDFs finds .pdf files in a directory and its subdirectories
func ScanPDFs(root string) ([]string, error) {
	var filePaths []string
	
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == ".pdf" {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return filePaths, nil
}
