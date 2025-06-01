package test

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/command"
)

// Runner provides utilities for testing command execution.
type Runner struct {
	harness *Harness
}

// NewRunner creates a new command runner with the given test harness.
func NewRunner(h *Harness) *Runner {
	return &Runner{
		harness: h,
	}
}

// Run executes a command with the given arguments and returns its stdout.
func (r *Runner) Run(cmd command.Command, args []string) (string, error) {
	// Save original stdin/stdout/stderr
	origStdin := os.Stdin
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Set up pipes for stdin/stdout/stderr
	os.Stdin = os.NewFile(uintptr(0), "/dev/stdin")
	os.Stdout = os.NewFile(uintptr(1), "/dev/stdout")
	os.Stderr = os.NewFile(uintptr(2), "/dev/stderr")

	// Create a context for the command
	ctx := context.Background()

	// Execute the command
	err := cmd.Execute(ctx, args)

	// Restore original stdin/stdout/stderr
	os.Stdin = origStdin
	os.Stdout = origStdout
	os.Stderr = origStderr

	// Return the captured stdout and any error
	return r.harness.GetStdout(), err
}

// RunWithInput executes a command with the given arguments and input, returning its stdout.
func (r *Runner) RunWithInput(cmd command.Command, args []string, input string) (string, error) {
	r.harness.WriteStdin(input)
	return r.Run(cmd, args)
}

// RunCommand is a simplified helper for common testing scenarios.
func RunCommand(t *testing.T, cmd command.Command, args []string) (stdout, stderr string, err error) {
	h := NewHarness(t)
	defer h.Teardown()
	h.Setup()

	// Execute the command
	ctx := context.Background()
	err = cmd.Execute(ctx, args)

	return h.GetStdout(), h.GetStderr(), err
}

// RunCommandWithInput is a simplified helper that provides input to the command.
func RunCommandWithInput(t *testing.T, cmd command.Command, args []string, input string) (stdout, stderr string, err error) {
	h := NewHarness(t)
	defer h.Teardown()
	h.Setup()

	// Write to stdin
	h.WriteStdin(input)

	// Execute the command
	ctx := context.Background()
	err = cmd.Execute(ctx, args)

	return h.GetStdout(), h.GetStderr(), err
}

// RunCommandWithFiles sets up files before running a command.
func RunCommandWithFiles(t *testing.T, cmd command.Command, args []string, files map[string]string) (stdout, stderr string, tempDir string, err error) {
	h := NewHarness(t)
	defer h.Teardown()
	h.Setup()

	// Create all specified files
	for name, content := range files {
		h.WriteFile(name, content)
	}

	// Execute the command
	ctx := context.Background()
	err = cmd.Execute(ctx, args)

	return h.GetStdout(), h.GetStderr(), h.TempDir(), err
}

// CaptureOutput returns a reader and writer for capturing command output.
func CaptureOutput() (io.Reader, io.Writer, func()) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(fmt.Sprintf("failed to create pipe: %v", err))
	}

	origStdout := os.Stdout
	os.Stdout = w

	cleanup := func() {
		w.Close()
		os.Stdout = origStdout
	}

	return r, w, cleanup
}
