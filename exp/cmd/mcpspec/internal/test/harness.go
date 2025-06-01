package test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Harness provides a testing environment for MCP commands.
type Harness struct {
	t          *testing.T
	stdin      *bytes.Buffer
	stdout     *bytes.Buffer
	stderr     *bytes.Buffer
	tempDir    string
	tempFiles  map[string]string
	origStdin  *os.File
	origStdout *os.File
	origStderr *os.File
}

// NewHarness creates a new test harness for the given test.
func NewHarness(t *testing.T) *Harness {
	tempDir, err := os.MkdirTemp("", "mcpspec-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	return &Harness{
		t:         t,
		stdin:     &bytes.Buffer{},
		stdout:    &bytes.Buffer{},
		stderr:    &bytes.Buffer{},
		tempDir:   tempDir,
		tempFiles: make(map[string]string),
	}
}

// Setup prepares the harness for a test run.
func (h *Harness) Setup() {
	h.origStdin = os.Stdin
	h.origStdout = os.Stdout
	h.origStderr = os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		h.t.Fatalf("failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	go func() {
		defer w.Close()
		io.Copy(w, h.stdin)
	}()

	r, w, err = os.Pipe()
	if err != nil {
		h.t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w
	go func() {
		io.Copy(h.stdout, r)
	}()

	r, w, err = os.Pipe()
	if err != nil {
		h.t.Fatalf("failed to create stderr pipe: %v", err)
	}
	os.Stderr = w
	go func() {
		io.Copy(h.stderr, r)
	}()
}

// Teardown cleans up the harness after a test run.
func (h *Harness) Teardown() {
	os.Stdin = h.origStdin
	os.Stdout = h.origStdout
	os.Stderr = h.origStderr

	if h.tempDir != "" {
		os.RemoveAll(h.tempDir)
	}
}

// WriteStdin writes data to the test's stdin.
func (h *Harness) WriteStdin(data string) {
	h.stdin.WriteString(data)
}

// WriteFile creates a temporary file with the given content.
func (h *Harness) WriteFile(name, content string) string {
	path := filepath.Join(h.tempDir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		h.t.Fatalf("failed to write temp file %s: %v", name, err)
	}
	h.tempFiles[name] = path
	return path
}

// ReadFile reads the content of a temporary file.
func (h *Harness) ReadFile(name string) string {
	path, ok := h.tempFiles[name]
	if !ok {
		path = filepath.Join(h.tempDir, name)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		h.t.Fatalf("failed to read temp file %s: %v", name, err)
	}
	return string(data)
}

// GetStdout returns the captured stdout output.
func (h *Harness) GetStdout() string {
	return h.stdout.String()
}

// GetStderr returns the captured stderr output.
func (h *Harness) GetStderr() string {
	return h.stderr.String()
}

// ClearOutput clears captured stdout and stderr.
func (h *Harness) ClearOutput() {
	h.stdout.Reset()
	h.stderr.Reset()
}

// TempDir returns the path to the temporary directory.
func (h *Harness) TempDir() string {
	return h.tempDir
}

// GetStdoutWriter returns a writer that writes to the captured stdout.
func (h *Harness) GetStdoutWriter() io.Writer {
	return h.stdout
}

// GetStderrWriter returns a writer that writes to the captured stderr.
func (h *Harness) GetStderrWriter() io.Writer {
	return h.stderr
}
