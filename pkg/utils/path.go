package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath resolves paths, handling '~' as the user's home directory.
// It supports mid-path expansion (e.g., "/tmp/../~/Documents").
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path, err
	}

	// Normalize separators to slash for splitting
	parts := strings.Split(filepath.ToSlash(path), "/")
	
	var result string

	// Preserve absolute path root (Unix: starts with empty string after split)
	if filepath.IsAbs(path) && !strings.Contains(path, ":") {
		result = string(os.PathSeparator)
	}

	for _, part := range parts {
		if part == "" {
			continue
		}
		
		if part == "~" {
			result = home
			continue
		}

		if result == "" {
			result = part
		} else {
			result = filepath.Join(result, part)
		}
	}

	// Handle the case where the path is just "/"
	if result == "" && filepath.IsAbs(path) {
		return string(os.PathSeparator), nil
	}
	
	return filepath.Clean(result), nil
}
