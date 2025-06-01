package test

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/command"
)

// TestCommand is a simple command implementation for testing.
type TestCommand struct {
	command.BaseCommand
	name      string
	output    string
	exitCode  int
	readStdin bool
}

// Name returns the command name.
func (c *TestCommand) Name() string {
	return c.name
}

// Execute runs the command.
func (c *TestCommand) Execute(ctx context.Context, args []string) error {
	if c.readStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Read from stdin: %s", data)
	} else {
		fmt.Fprint(os.Stdout, c.output)
	}

	if c.exitCode != 0 {
		return fmt.Errorf("command exited with code %d", c.exitCode)
	}
	return nil
}

// Usage returns the command usage.
func (c *TestCommand) Usage() string {
	return fmt.Sprintf("Usage: %s [args]", c.name)
}

func TestRunner(t *testing.T) {
	// Test basic command execution
	cmd := &TestCommand{
		name:     "test",
		output:   "Hello, world!",
		exitCode: 0,
	}

	stdout, stderr, err := RunCommand(t, cmd, []string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if stdout != "Hello, world!" {
		t.Errorf("unexpected stdout: got %q, want %q", stdout, "Hello, world!")
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: got %q, want %q", stderr, "")
	}

	// Test command with error
	errCmd := &TestCommand{
		name:     "error",
		output:   "Error output",
		exitCode: 1,
	}

	stdout, stderr, err = RunCommand(t, errCmd, []string{})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if stdout != "Error output" {
		t.Errorf("unexpected stdout: got %q, want %q", stdout, "Error output")
	}

	// Test command with input
	inputCmd := &TestCommand{
		name:      "input",
		readStdin: true,
		exitCode:  0,
	}

	stdout, stderr, err = RunCommandWithInput(t, inputCmd, []string{}, "test input")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if stdout != "Read from stdin: test input" {
		t.Errorf("unexpected stdout: got %q, want %q", stdout, "Read from stdin: test input")
	}

	// Test command with files
	filesCmd := &TestCommand{
		name:     "files",
		output:   "Files command",
		exitCode: 0,
	}

	files := map[string]string{
		"test1.txt": "content 1",
		"test2.txt": "content 2",
	}

	stdout, stderr, tempDir, err := RunCommandWithFiles(t, filesCmd, []string{}, files)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if stdout != "Files command" {
		t.Errorf("unexpected stdout: got %q, want %q", stdout, "Files command")
	}

	// Verify files were created in the temp directory
	for name, content := range files {
		path := fmt.Sprintf("%s/%s", tempDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read file %s: %v", path, err)
			continue
		}
		if string(data) != content {
			t.Errorf("unexpected file content: got %q, want %q", string(data), content)
		}
	}
}
