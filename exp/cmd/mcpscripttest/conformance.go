package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed conformance_tests/*.txt
var conformanceTests embed.FS

// extractConformanceTests extracts the embedded conformance tests to a temporary directory
// and returns the path to that directory
func extractConformanceTests() (string, error) {
	// Create a temporary directory for the conformance tests
	tempDir, err := os.MkdirTemp("", "mcp-conformance-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}

	// Walk through the embedded files and write them to the temp directory
	err = fs.WalkDir(conformanceTests, "conformance_tests", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == "conformance_tests" {
			return nil
		}

		// Create the relative path from the root
		relPath, err := filepath.Rel("conformance_tests", path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}

		// Get the destination path in the temp directory
		destPath := filepath.Join(tempDir, relPath)

		if d.IsDir() {
			// Create the directory
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", destPath, err)
			}
		} else {
			// Read the file contents
			contents, err := conformanceTests.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read embedded file %s: %v", path, err)
			}

			// Make sure the parent directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %v", destPath, err)
			}

			// Write the file
			if err := os.WriteFile(destPath, contents, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %v", destPath, err)
			}
		}

		return nil
	})

	if err != nil {
		os.RemoveAll(tempDir) // Clean up on error
		return "", fmt.Errorf("failed to extract conformance tests: %v", err)
	}

	return tempDir, nil
}