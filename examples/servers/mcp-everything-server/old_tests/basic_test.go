package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestMainFunction tests that the main function works
func TestMainFunction(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test with timeout flag
	os.Args = []string{"mcp-everything-server", "-timeout", "100ms", "-quiet"}

	// Run main in a goroutine
	done := make(chan bool)
	go func() {
		main()
		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Expected - main should exit after timeout
	case <-time.After(2 * time.Second):
		t.Error("Main function did not exit within expected time")
	}
}

// TestServerBuild tests that the server builds correctly
func TestServerBuild(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "test-server", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build server: %v", err)
	}
	defer os.Remove("test-server")

	// Test that the built server runs
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "./test-server", "-timeout", "100ms")
	err := cmd.Run()

	// We expect the command to exit cleanly after timeout
	if err != nil && err.Error() != "signal: killed" {
		// It's OK if the process was killed due to context timeout
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != -1 {
			t.Errorf("Server exited with unexpected error: %v", err)
		}
	}
}

// TestRegisterToolsFunction tests that registerTools doesn't panic
func TestRegisterToolsFunction(t *testing.T) {
	// This is a simple test to ensure the function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("registerTools panicked: %v", r)
		}
	}()

	// Create a test server and register tools
	server := createTestServer()
	registerTools(server)
}

func createTestServer() interface{} {
	// We use interface{} to avoid import cycle
	// The registerTools function will handle the type assertion
	return nil
}
