package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMainFunction(t *testing.T) {
	// Skip this test if being run by go test ./...
	if os.Getenv("FULL_TEST") != "1" {
		t.Skip("Skipping test; set FULL_TEST=1 to run")
	}

	// Build the mcpdeadcode binary
	cmd := exec.Command("go", "build", "-o", "mcpdeadcode_test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build mcpdeadcode: %v", err)
	}
	defer os.Remove("mcpdeadcode_test")

	// Run mcpdeadcode with -h flag to test basic functionality
	cmd = exec.Command("./mcpdeadcode_test", "-h")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run mcpdeadcode -h: %v\nOutput: %s", err, output)
	}

	// Check the help output contains expected text
	if !containsAny(string(output), "Usage:", "deadcode") {
		t.Errorf("Help output does not contain expected text: %s", output)
	}
}

// TestDeadcodeCheck runs a deadcode check on the current package
func TestDeadcodeCheck(t *testing.T) {
	// Skip this test if being run by go test ./...
	if os.Getenv("FULL_TEST") != "1" {
		t.Skip("Skipping test; set FULL_TEST=1 to run")
	}

	// Make sure deadcode is installed
	_, err := exec.LookPath("deadcode")
	if err != nil {
		t.Log("Installing deadcode...")
		cmd := exec.Command("go", "install", "golang.org/x/tools/cmd/deadcode@latest")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to install deadcode: %v", err)
		}
	}

	// Run deadcode on the current package
	cmd := exec.Command("deadcode", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If we have compile errors, that's okay
		if containsAny(string(output), "packages contain errors") {
			t.Log("Note: Packages contain compile errors, which may affect deadcode analysis")
			return
		}
		t.Fatalf("Failed to run deadcode: %v\nOutput: %s", err, output)
	}

	// Don't fail the test if deadcode finds issues
	// Just log them so we're aware
	if len(output) > 0 {
		t.Logf("Deadcode found potential issues: %s", output)
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
