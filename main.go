package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pdffer/pkg/pdf"
	"pdffer/pkg/tui"
	"pdffer/pkg/utils"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		runTUI()
		return
	}

	command := args[0]
	switch command {
	case "merge":
		if len(args) < 3 {
			printUsage()
			os.Exit(1)
		}

		outputFile, _ := utils.ExpandPath(args[1])
		inputFiles := args[2:]
		var expandedInputs []string
		for _, f := range inputFiles {
			ef, _ := utils.ExpandPath(f)
			expandedInputs = append(expandedInputs, ef)
		}

		fmt.Printf("Merging %d files into %s...\n", len(expandedInputs), outputFile)
		err := pdf.MergePDFs(expandedInputs, outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error merging PDFs: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Success!")

	case "append":
		if len(args) < 4 {
			printUsage()
			os.Exit(1)
		}
		basePDF, _ := utils.ExpandPath(args[1])
		outputFile, _ := utils.ExpandPath(args[2])
		imageInputs := args[3:]

		// Expand directories in imageInputs
		var allImages []string
		for _, path := range imageInputs {
			epath, _ := utils.ExpandPath(path)
			info, err := os.Stat(epath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Skipping %s: %v\n", epath, err)
				continue
			}
			if info.IsDir() {
				imgs, err := pdf.ScanDirectory(epath)
				if err == nil {
					allImages = append(allImages, imgs...)
				}
			} else {
				allImages = append(allImages, epath)
			}
		}

		fmt.Printf("Appending %d images to %s -> %s...\n", len(allImages), basePDF, outputFile)
		err := pdf.AppendImagesToPDF(basePDF, allImages, outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error appending images: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Success!")

	case "help":
		printUsage()

	default:
		// Legacy Mode: Input images -> Output PDF
		runLegacyCLI(args)
	}
}

func runTUI() {
	p := tea.NewProgram(tui.InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  pdffer                                     # Run TUI mode")
	fmt.Println("  pdffer [images...] [output.pdf]            # Create PDF from images (Legacy)")
	fmt.Println("  pdffer merge <output.pdf> <in1.pdf> ...    # Merge PDFs")
	fmt.Println("  pdffer append <base.pdf> <out.pdf> <imgs>  # Append images to PDF")
}

func runLegacyCLI(args []string) {
	var inputPaths []string
	outputFile := "output.pdf"

	// Check if the last argument is likely an output file
	lastArg := args[len(args)-1]
	if len(args) > 1 && strings.HasSuffix(strings.ToLower(lastArg), ".pdf") {
		outputFile, _ = utils.ExpandPath(lastArg)
		inputPaths = args[:len(args)-1]
	} else {
		inputPaths = args
	}

	var allImages []string

	fmt.Printf("Scanning inputs...\n")
	for _, path := range inputPaths {
		epath, _ := utils.ExpandPath(path)
		info, err := os.Stat(epath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Skipping %s: %v\n", epath, err)
			continue
		}

		if info.IsDir() {
			images, err := pdf.ScanDirectory(epath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to scan directory %s: %v\n", epath, err)
			} else {
				allImages = append(allImages, images...)
			}
		} else {
			// Assume it's an image file if it's not a directory
			ext := strings.ToLower(filepath.Ext(epath))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
				allImages = append(allImages, epath)
			} else {
				// Add anyway, let generator filter/fail
				allImages = append(allImages, epath)
			}
		}
	}

	if len(allImages) == 0 {
		fmt.Println("No valid images found.")
		os.Exit(1)
	}

	fmt.Printf("Found %d images. Generating PDF to %s...\n", len(allImages), outputFile)
	err := pdf.GeneratePDF(allImages, outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Success! PDF saved to %s\n", outputFile)
}
