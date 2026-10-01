package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMCPServeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary workspace
	wsDir, err := os.MkdirTemp("", "mcp-serve-test")
	if err != nil {
		t.Fatalf("Failed to create temp workspace: %v", err)
	}
	defer os.RemoveAll(wsDir)

	// Build the mcp-serve binary for testing
	binPath := filepath.Join(wsDir, "mcp-serve")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build mcp-serve: %v\nOutput: %s", err, out)
	}

	// Helper to run mcp-serve
	runServe := func(args ...string) error {
		args = append([]string{"-workspace", wsDir, "-v"}, args...)
		cmd := exec.Command(binPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// 1. Start a persistent process (sleep 10)
	// We run this in background (startServer doesn't block if we run it directly?
	// Wait, main blocks on cmd.Wait(). So `startServer` BLOCKS.
	// So we need to start it in a separate process or goroutine.
	// The CLI usage is `mcp-serve [flags] -- command`.
	// If we run `mcp-serve -- sh -c "sleep 10"`, it blocks until sleep finishes.

	// So for "Start", we should start it asynchronously.
	startCmd := exec.Command(binPath, "-workspace", wsDir, "-v", "--", "sh", "-c", "sleep 10")
	// cmd.Start() starts it but doesn't wait.
	if err := startCmd.Start(); err != nil {
		t.Fatalf("Failed to start server process: %v", err)
	}
	defer func() {
		if startCmd.Process != nil {
			startCmd.Process.Kill()
		}
	}()

	// Wait for PID file to appear
	pidFile := filepath.Join(wsDir, PidFile)
	timeout := time.After(5 * time.Second)
	found := false
	for !found {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for PID file")
		case <-time.After(100 * time.Millisecond):
			if _, err := os.Stat(pidFile); err == nil {
				found = true
			}
		}
	}

	// 2. Check Status
	if err := runServe("-status"); err != nil {
		t.Errorf("Status check failed: %v", err)
	}

	// 3. Stop Server
	if err := runServe("-stop"); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Verify PID file is gone
	timeout = time.After(5 * time.Second)
	gone := false
	for !gone {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for PID file removal")
		case <-time.After(100 * time.Millisecond):
			if _, err := os.Stat(pidFile); os.IsNotExist(err) {
				gone = true
			}
		}
	}

	// 4. Verify process stopped (wait for startCmd to exit)
	done := make(chan error, 1)
	go func() {
		done <- startCmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			// It might return error because it was killed or exited with signal
			// t.Logf("Server exited with: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Server process did not exit after stop command")
	}
}

// Simple unit test for readPidFile
func TestReadPidFile(t *testing.T) {
	// Create a temp file
	tmpfile, err := os.CreateTemp("", "pidfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Write PID
	pid := 12345
	if err := writePidFile(tmpfile.Name(), pid); err != nil {
		t.Fatal(err)
	}

	// Read PID
	readPid, err := readPidFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if readPid != pid {
		t.Errorf("got %d, want %d", readPid, pid)
	}
}
