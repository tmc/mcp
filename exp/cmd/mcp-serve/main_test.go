package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMCPServeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "mcp-serve")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	logPath := filepath.Join(dir, "server.log")
	log, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	cmd := exec.Command(binary, "-workspace", dir, "-v", "--", "sleep", "60")
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
		close(done)
	}()
	pidFile := filepath.Join(dir, PidFile)
	t.Cleanup(func() {
		if pid, err := readPidFile(pidFile); err == nil {
			if process, err := os.FindProcess(pid); err == nil {
				process.Kill()
			}
		}
		cmd.Process.Kill()
		<-done
	})
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := readPidFile(pidFile); err == nil {
			break
		}
		select {
		case err := <-done:
			output, _ := os.ReadFile(logPath)
			t.Fatalf("server exited before writing PID: %v\n%s", err, output)
		case <-timer.C:
			output, _ := os.ReadFile(logPath)
			t.Fatalf("timeout waiting for PID\n%s", output)
		case <-ticker.C:
		}
	}
	run := func(args ...string) {
		t.Helper()
		args = append([]string{"-workspace", dir, "-v"}, args...)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if output, err := exec.CommandContext(ctx, binary, args...).CombinedOutput(); err != nil {
			t.Fatalf("mcp-serve %v: %v\n%s", args, err, output)
		}
	}
	run("-status")
	run("-stop")
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("PID file remains after stop: %v", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server process did not exit after stop")
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
