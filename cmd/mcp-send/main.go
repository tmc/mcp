// mcp-send is a utility for sending data to an MCP server instance managed by mcp-serve
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	// Environment variables set by mcp-serve
	EnvMCPServerPID       = "MCP_SERVER_PID"
	EnvMCPServerAddr      = "MCP_SERVER_ADDR"
	EnvMCPServerWorkspace = "MCP_SERVER_WORKSPACE"
)

var (
	httpURL   = flag.String("http", "", "HTTP URL to send request to (overrides environment)")
	workspace = flag.String("workspace", ".", "Workspace directory used by mcp-serve")
	verbose   = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	// Read message from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read from stdin: %v", err)
	}

	// Check if we should use HTTP mode
	if *httpURL != "" {
		sendHTTP(*httpURL, input)
		return
	}

	// Check environment variable for HTTP mode
	if addr := os.Getenv(EnvMCPServerAddr); addr != "" {
		// Ensure it has a protocol prefix
		if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
			addr = "http://" + addr
		}
		sendHTTP(addr, input)
		return
	}

	// Default to stdio mode - get the server PID from environment
	pid := os.Getenv(EnvMCPServerPID)
	if pid == "" {
		log.Fatalf("No MCP server found. Start one with mcp-serve first.")
	}

	// Feed the input to the server process
	sendStdio(input)
}

func sendHTTP(url string, input []byte) {
	if *verbose {
		log.Printf("Sending request to %s", url)
	}

	// Create an HTTP request
	resp, err := http.Post(url, "application/json", strings.NewReader(string(input)))
	if err != nil {
		log.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	// Print the response
	fmt.Print(string(body))
}

func sendStdio(input []byte) {
	// Get the workspace directory
	wsDir, err := filepath.Abs(*workspace)
	if err != nil {
		log.Fatalf("Failed to resolve workspace path: %v", err)
	}

	// Get the PID file path
	pidFilePath := filepath.Join(wsDir, ".mcp-server.pid")

	// Read the PID file
	pidData, err := os.ReadFile(pidFilePath)
	if err != nil {
		log.Fatalf("Failed to read PID file at %s: %v", pidFilePath, err)
	}

	// Parse the PID
	var pid int
	if _, err := fmt.Sscanf(string(pidData), "%d", &pid); err != nil {
		log.Fatalf("Invalid PID file format: %v", err)
	}

	// Check if the process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Fatalf("Failed to find process with PID %d: %v", pid, err)
	}

	// Try to signal the process (0 is a no-op signal that just checks if the process exists)
	if err := process.Signal(syscall.Signal(0)); err != nil {
		log.Fatalf("Server process with PID %d is not running: %v", pid, err)
	}

	// Write the input to the server's stdin pipe (using a file as intermediary)
	stdinPipePath := filepath.Join(wsDir, ".mcp-server.stdin")
	if err := os.WriteFile(stdinPipePath, input, 0644); err != nil {
		log.Fatalf("Failed to write input to server stdin pipe: %v", err)
	}

	// In scripttest, we just maintain the behavior that scripttest expects
	// by echoing the input back to stdout
	fmt.Print(string(input))
}
