// Command mcpspy tests
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestJSONScanner(t *testing.T) {
	// Find existing mcpspy binary
	mcpspyPath, err := exec.LookPath("./mcpspy")
	if err != nil {
		// Try to build it
		buildCmd := exec.Command("go", "build")
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build mcpspy: %v\n%s", err, output)
		}

		// Check again
		mcpspyPath, err = exec.LookPath("./mcpspy")
		if err != nil {
			t.Fatalf("Unable to find mcpspy binary: %v", err)
		}
	}

	// Get absolute path to the binary
	mcpspyPath, err = filepath.Abs(mcpspyPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path to mcpspy: %v", err)
	}

	// Main test file
	testFile := "testdata/all_json_scanner_tests.txt"

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "mcpspy-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Copy test data to the temp directory
	if err := copyDir("testdata", filepath.Join(tempDir, "testdata")); err != nil {
		t.Fatalf("Failed to copy test data: %v", err)
	}

	// Copy the test script to the temp directory
	testScript := filepath.Join(tempDir, "script.txt")
	testData, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test script: %v", err)
	}

	// Replace references to mcpspy with absolute path
	testData = bytes.ReplaceAll(testData, []byte("mcpspy "), []byte(mcpspyPath+" "))

	if err := os.WriteFile(testScript, testData, 0644); err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}

	// Make the script executable
	if err := os.Chmod(testScript, 0755); err != nil {
		t.Fatalf("Failed to make test script executable: %v", err)
	}

	// Run the test in the temporary directory
	cmd := exec.Command("/bin/bash", testScript)
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "PATH="+os.Getenv("PATH"))

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Test failed: %v\nOutput:\n%s", err, output)
	} else {
		t.Logf("Test passed. Output:\n%s", output)
	}
}

// Helper function to recursively copy directories
func copyDir(src, dst string) error {
	// Create the destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Read the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
