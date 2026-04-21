package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSendHTTP(t *testing.T) {
	// Setup test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "test message" {
			t.Errorf("Expected 'test message', got '%s'", body)
		}
		fmt.Fprint(w, "response")
	}))
	defer ts.Close()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	*verbose = true
	sendHTTP(ts.URL, []byte("test message"))

	w.Close()
	os.Stdout = oldStdout

	out, _ := io.ReadAll(r)
	if string(out) != "response" {
		t.Errorf("Expected 'response', got '%s'", out)
	}
}

func TestSendStdio(t *testing.T) {
	// Setup temp workspace
	wsDir := t.TempDir()

	// Create fake PID file using current process PID (always running)
	pidPath := filepath.Join(wsDir, ".mcp-server.pid")
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		t.Fatal(err)
	}

	// Set flag for workspace
	// Reset flag usage
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	workspace = flag.String("workspace", wsDir, "Workspace directory used by mcp-serve")

	// Ensure we don't assume sendStdio uses the flag global variable directly if we define it in main.
	// But in main.go, `workspace` is a global var.
	// We need to make sure `sendStdio` uses the variable we just set.
	// Wait, `workspace` var in `main.go` is package level.
	// If I redeclare it in main_test.go, it shadows it? No, same package.
	// But `main_test.go` is package main.
	// If I mistakenly redeclare `workspace` in `TestSendStdio` via `workspace = flag.String...`, I confirm it points to the NEW flag set.
	// However, `main.go` defines `var workspace = flag.String(...)`.
	// I cannot re-assign `workspace` address. I can only change value pointed to by `workspace`.

	// Correct approach: set the value of the existing global flag.
	// `workspace` is exported? No, package level in main.
	// So `main_test.go` sees `workspace` from `main.go`.
	// `flag.Parse()` in main sets it.
	// I can just set `*workspace = wsDir`.
	*workspace = wsDir

	// Capture stdout (sendStdio prints input back)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	sendStdio([]byte("test input"))

	w.Close()
	os.Stdout = oldStdout

	// Verify file
	stdinPath := filepath.Join(wsDir, ".mcp-server.stdin")
	content, err := os.ReadFile(stdinPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "test input" {
		t.Errorf("File Content: Want 'test input', got '%s'", content)
	}

	// Verify echo
	out, _ := io.ReadAll(r)
	if string(out) != "test input" {
		t.Errorf("Stdout: Want 'test input', got '%s'", out)
	}
}
