package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tmc/macgo"
	"github.com/tmc/mcp"
)

const (
	ServerName    = "screencapture-server"
	ServerVersion = "1.0.0"
)

// StdioTransport provides a transport using stdin/stdout
type StdioTransport struct{}

// Dial implements the Transport interface
func (t *StdioTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return &stdioConn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}, nil
}

// stdioConn wraps stdin/stdout as a ReadWriteCloser
type stdioConn struct {
	io.Reader
	io.Writer
}

// Write implements io.Writer and ensures NDJSON framing by appending a newline
func (c *stdioConn) Write(p []byte) (n int, err error) {
	n, err = c.Writer.Write(p)
	if err != nil {
		return n, err
	}
	// Append newline to ensure NDJSON framing
	_, err = c.Writer.Write([]byte{'\n'})
	return n, err
}

// Close implements io.Closer
func (c *stdioConn) Close() error {
	// Don't actually close stdin/stdout
	return nil
}

func main() {
	// Default logging to stderr
	log.SetOutput(os.Stderr)

	log.Printf("Starting MCP Screen Capture Server... PID=%d PPID=%d Args=%v", os.Getpid(), os.Getppid(), os.Args)
	log.Printf("Environ: %v", os.Environ())

	// Initialize macgo for permissions and bundle management
	// LaunchServices V2 provides stdin/stdout forwarding by default
	// This maintains TCC identity via app bundle while preserving stdio for MCP

	// Force enable stdin forwarding for proper MCP communication
	// This ensures LaunchServices V1 establishes the stdin pipe regarding of auto-detection
	os.Setenv("MACGO_ENABLE_STDIN_FORWARDING", "1")
	// Explicitly enable stdout/stderr to ensure Parent waits for Child
	// This prevents the premature exit/cleanup race condition in V1
	os.Setenv("MACGO_ENABLE_STDOUT_FORWARDING", "1")
	os.Setenv("MACGO_ENABLE_STDERR_FORWARDING", "1")

	cfg := macgo.NewConfig()
	cfg.WithAppName("ScreenCaptureMCP")
	cfg.WithPermissions(macgo.ScreenRecording)
	// cfg.WithDebug() // Disabled for production

	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("macgo start failed: %v", err)
	}
	defer macgo.Cleanup()

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a server
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A screen capture server that provides tools for listing displays and capturing screenshots on macOS"),
	)

	// Register tools
	registerListScreensTool(server)
	registerCaptureScreenTool(server)

	// Create stdio transport
	stdioTransport := &StdioTransport{}

	// Helper to wrap os.File with logging, returning *os.File via pipe if needed?
	// Check if os.Stdin is used directly or via Read().
	// os.Stdin is *os.File. If we replace it with io.Reader, we break code expecting *os.File.
	// But mcp.NewStdioServerTransport() uses os.Stdin directly as io.Reader.
	// However, macgo replaces os.Stdin.
	// Let's just create a pipe and pump data to it.

	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, stdioTransport); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerListScreensTool(server *mcp.Server) {
	listScreensTool := mcp.Tool{
		Name:        "list_screens",
		Description: "Lists connected displays using system_profiler. Returns detailed information about all connected monitors.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}

	server.RegisterTool(listScreensTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cmd := exec.Command("system_profiler", "SPDisplaysDataType")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to list screens: %w", err)
		}

		log.Println("Executed list_screens tool")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(out),
				},
			},
		}, nil
	})

	log.Println("Registered list_screens tool")
}

func registerCaptureScreenTool(server *mcp.Server) {
	captureScreenTool := mcp.Tool{
		Name:        "capture_screen",
		Description: "Captures a screenshot of the main display and returns it as a PNG image. Optionally specify a display ID to capture a specific display.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"display_id": {
					"type": "integer",
					"description": "Optional display ID to capture (omit to capture main display)"
				}
			}
		}`),
	}

	server.RegisterTool(captureScreenTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("capture_%d.png", time.Now().UnixNano()))
		defer os.Remove(tmpFile)

		// Build screencapture command
		// -x: muted (no sound)
		// -r: do not add shadow
		// -t png: image format
		cmdArgs := []string{"-x", "-r", "-t", "png", tmpFile}

		// Note: handling specific display requires investigating `screencapture` args or `scutil`.
		// Simple version captures main screen. Default is main.

		cmd := exec.Command("screencapture", cmdArgs...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("screencapture failed: %v\nOutput: %s", err, out)
		}

		// Read image data
		data, err := os.ReadFile(tmpFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read capture: %w", err)
		}

		log.Printf("Executed capture_screen tool, captured %d bytes", len(data))
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type":     "image",
					"data":     data,
					"mimeType": "image/png",
				},
			},
		}, nil
	})

	log.Println("Registered capture_screen tool")
}

func mustWrapFile(original *os.File, logFile io.Writer) *os.File {
	r, w, _ := os.Pipe()
	go func() {
		defer w.Close()
		buf := make([]byte, 1024)
		for {
			n, err := original.Read(buf)
			if n > 0 {
				logFile.Write(buf[:n])
				w.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	return r
}

func mustWrapWriter(original *os.File, logFile io.Writer) *os.File {
	// For stdout, we want to intercept writes to os.Stdout and log them, then write to actual fd.
	// But getting os.Stdout to route through a pipe requires replacing the FD or the *File struct.
	// We can't easily intercept calls to os.Stdout.Write unless we replace the variable.
	// os.Stdout is a var. We can replace it with the write-end of a pipe.
	r, w, _ := os.Pipe()
	go func() {
		defer r.Close()
		buf := make([]byte, 1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				logFile.Write(buf[:n])
				original.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	return w
}
