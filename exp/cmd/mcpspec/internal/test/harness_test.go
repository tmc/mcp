package test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHarness(t *testing.T) {
	h := NewHarness(t)
	defer h.Teardown()
	h.Setup()

	// Test stdin/stdout capture
	h.WriteStdin("test input\n")
	fmt.Println("test output")
	fmt.Fprintln(os.Stderr, "test error")

	if h.GetStdout() != "test output\n" {
		t.Errorf("stdout capture failed, got: %q", h.GetStdout())
	}

	if h.GetStderr() != "test error\n" {
		t.Errorf("stderr capture failed, got: %q", h.GetStderr())
	}

	// Test file operations
	testFile := h.WriteFile("test.txt", "file content")
	if content := h.ReadFile("test.txt"); content != "file content" {
		t.Errorf("file read/write failed, got: %q", content)
	}

	// Verify file exists at the returned path
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("file was not created at %s", testFile)
	}

	// Test temp directory
	if tempDir := h.TempDir(); tempDir == "" {
		t.Error("temp directory not created")
	} else {
		// Create a file in the temp directory and verify it exists
		testPath := filepath.Join(tempDir, "subdir/nested.txt")
		os.MkdirAll(filepath.Dir(testPath), 0755)
		os.WriteFile(testPath, []byte("nested content"), 0644)
		if _, err := os.Stat(testPath); os.IsNotExist(err) {
			t.Errorf("failed to create nested file in temp directory")
		}
	}

	// Test clearing output
	h.ClearOutput()
	if h.GetStdout() != "" || h.GetStderr() != "" {
		t.Error("ClearOutput failed to clear stdout/stderr")
	}
}
