// mcp-serve is a utility for managing MCP server instances for testing
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// PidFile is the path to the file storing the server process ID
	PidFile = ".mcp-server.pid"
	// StdoutFile is the path to the file storing stdout from the server
	StdoutFile = ".mcp-server.stdout"
	// StderrFile is the path to the file storing stderr from the server
	StderrFile = ".mcp-server.stderr"
	// StdinFile is the path to the file storing stdin for the server
	StdinFile = ".mcp-server.stdin"

	// Environment variables
	// EnvMCPEndpoint is the environment variable to store the base path for FIFOs
	EnvMCPEndpoint = "MCP_ENDPOINT"
	// EnvMCPServerPID is the environment variable to store the server PID
	EnvMCPServerPID = "MCP_SERVER_PID"
	// EnvMCPServerPort is the environment variable to store the server port (if using TCP)
	EnvMCPServerPort = "MCP_SERVER_PORT"
	// EnvMCPServerAddr is the environment variable to store the server address (if HTTP)
	EnvMCPServerAddr = "MCP_SERVER_ADDR"
	// EnvMCPServerWorkspace is the environment variable to store the workspace directory
	EnvMCPServerWorkspace = "MCP_SERVER_WORKSPACE"
)

var (
	stop      = flag.Bool("stop", false, "Stop the running server")
	status    = flag.Bool("status", false, "Check if server is running")
	send      = flag.Bool("send", false, "Send data from stdin to the server")
	timeout   = flag.Duration("timeout", 10*time.Second, "Timeout for stopping the server")
	workspace = flag.String("workspace", ".", "Workspace directory for server data")
	verbose   = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	// Set up workspace directory
	wsDir, err := filepath.Abs(*workspace)
	if err != nil {
		log.Fatalf("Failed to resolve workspace path: %v", err)
	}

	if err := os.MkdirAll(wsDir, 0755); err != nil {
		log.Fatalf("Failed to create workspace directory: %v", err)
	}

	pidFile := filepath.Join(wsDir, PidFile)
	stdoutFile := filepath.Join(wsDir, StdoutFile)
	stderrFile := filepath.Join(wsDir, StderrFile)
	stdinFile := filepath.Join(wsDir, StdinFile)

	// Handle command based on flags
	switch {
	case *stop:
		stopServer(pidFile, *timeout)
	case *status:
		checkStatus(pidFile)
	case *send:
		sendToServer(wsDir)
	default:
		startServer(pidFile, stdoutFile, stderrFile, stdinFile, flag.Args())
	}
}

func startServer(pidFile, stdoutFile, stderrFile, stdinFile string, args []string) {
	// Check if server is already running
	if pid, err := readPidFile(pidFile); err == nil {
		if isProcessRunning(pid) {
			log.Printf("Server already running with PID %d", pid)
			return
		}
		// PID file exists but process is not running, clean up
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to remove stale PID file: %v", err)
		}
	}

	// No arguments provided
	if len(args) == 0 {
		log.Fatalf("No command provided. Usage: mcp-serve [flags] -- command [args...]")
	}

	// Open stdout file in append mode for capturing server output
	stdout, err := os.OpenFile(stdoutFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to create stdout file: %v", err)
	}
	defer stdout.Close()

	stderr, err := os.OpenFile(stderrFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to create stderr file: %v", err)
	}
	defer stderr.Close()

	// Prepare the command
	cmd := exec.Command(args[0], args[1:]...)

	// Create pipes for communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}

	cmd.Stderr = stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Write PID to file
	if err := writePidFile(pidFile, cmd.Process.Pid); err != nil {
		log.Printf("Warning: Failed to write PID file: %v", err)
	}

	// Start goroutine to capture stdout in real-time
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := stdoutPipe.Read(buf)
			if err != nil {
				if err != io.EOF {
					if *verbose {
						log.Printf("Error reading stdout: %v", err)
					}
				}
				break
			}

			if n > 0 {
				stdout.Write(buf[:n])
				stdout.Sync()
			}
		}
	}()

	// Start goroutine to monitor stdin file and send to process
	go func() {
		for {
			// Check if stdin file exists with content
			if info, err := os.Stat(stdinFile); err == nil && info.Size() > 0 {
				data, err := os.ReadFile(stdinFile)
				if err == nil && len(data) > 0 {
					if *verbose {
						log.Printf("Sending %d bytes to server", len(data))
					}
					stdin.Write(data)
					// Clear the file after reading
					os.WriteFile(stdinFile, []byte{}, 0644)
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Set environment variables for other tools to use
	os.Setenv(EnvMCPServerPID, fmt.Sprintf("%d", cmd.Process.Pid))
	os.Setenv(EnvMCPEndpoint, filepath.Join(filepath.Dir(pidFile), ".mcp-server"))
	os.Setenv(EnvMCPServerWorkspace, filepath.Dir(pidFile))

	// Try to detect if this is an HTTP server (from arguments)
	for i, arg := range args {
		if (arg == "--http" || arg == "-http") && i+1 < len(args) {
			os.Setenv(EnvMCPServerAddr, args[i+1])
			break
		} else if strings.HasPrefix(arg, "--http=") {
			addr := strings.TrimPrefix(arg, "--http=")
			os.Setenv(EnvMCPServerAddr, addr)
			break
		}
	}

	// Set up signal handling for graceful shutdown
	setupSignalHandler(cmd.Process, pidFile)

	if *verbose {
		log.Printf("Started server with PID %d: %s", cmd.Process.Pid, strings.Join(args, " "))
		log.Printf("Set environment variables: %s=%d, %s=%s",
			EnvMCPServerPID, cmd.Process.Pid,
			EnvMCPServerWorkspace, filepath.Dir(pidFile))
		if addr := os.Getenv(EnvMCPServerAddr); addr != "" {
			log.Printf("Set %s=%s", EnvMCPServerAddr, addr)
		}
	}

	// Wait for the process to complete (this will block)
	if err := cmd.Wait(); err != nil {
		if *verbose {
			log.Printf("Server process exited with error: %v", err)
		}
	}

	// Clean up PID file when process exits
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Failed to remove PID file: %v", err)
	}
}

func sendToServer(wsDir string) {
	pidFile := filepath.Join(wsDir, PidFile)
	pid, err := readPidFile(pidFile)
	if err != nil {
		log.Fatalf("Failed to read PID file: %v", err)
	}

	if !isProcessRunning(pid) {
		log.Fatalf("No server running with PID %d", pid)
	}

	// Read the message from stdin
	stdinBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read from stdin: %v", err)
	}

	if len(stdinBytes) == 0 {
		log.Fatalf("No data provided on stdin")
	}

	// Ensure the message ends with a newline
	if stdinBytes[len(stdinBytes)-1] != '\n' {
		stdinBytes = append(stdinBytes, '\n')
	}

	// Write to stdin file
	stdinFile := filepath.Join(wsDir, StdinFile)
	if err := os.WriteFile(stdinFile, stdinBytes, 0644); err != nil {
		log.Fatalf("Failed to write to stdin file: %v", err)
	}

	if *verbose {
		log.Printf("Sent %d bytes to server via stdin file", len(stdinBytes))
	}

	// Wait a bit for the server to process and respond
	time.Sleep(500 * time.Millisecond)

	// Read the response from stdout file
	stdoutFile := filepath.Join(wsDir, StdoutFile)

	// Try to find the response for our request
	file, err := os.Open(stdoutFile)
	if err != nil {
		log.Fatalf("Failed to open stdout file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var response string
	for scanner.Scan() {
		line := scanner.Text()
		// Look for JSON-RPC response
		if strings.Contains(line, `"jsonrpc"`) && strings.Contains(line, `"id"`) {
			response = line
		}
	}

	if response != "" {
		fmt.Println(response)
	} else {
		// If no valid response found, just output the whole file
		data, _ := os.ReadFile(stdoutFile)
		fmt.Print(string(data))
	}
}

func stopServer(pidFile string, timeout time.Duration) {
	pid, err := readPidFile(pidFile)
	if err != nil {
		log.Fatalf("Failed to read PID file: %v", err)
	}

	if !isProcessRunning(pid) {
		log.Printf("No server running with PID %d", pid)
		// Clean up stale PID file
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to remove stale PID file: %v", err)
		}
		return
	}

	// Send SIGTERM to the process
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Fatalf("Failed to find process with PID %d: %v", pid, err)
	}

	if *verbose {
		log.Printf("Sending SIGTERM to server with PID %d", pid)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		log.Fatalf("Failed to send SIGTERM to process: %v", err)
	}

	// Wait for the process to exit or timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if !isProcessRunning(pid) {
				if *verbose {
					log.Printf("Server with PID %d has exited", pid)
				}
				break
			}

			select {
			case <-ctx.Done():
				if *verbose {
					log.Printf("Timeout waiting for server to exit, sending SIGKILL")
				}
				// Force kill if timeout
				if err := process.Kill(); err != nil {
					log.Printf("Warning: Failed to SIGKILL process: %v", err)
				}
				return
			case <-time.After(100 * time.Millisecond):
				// Continue checking
			}
		}
	}()

	wg.Wait()

	// Clean up PID file
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Failed to remove PID file: %v", err)
	}

	// Clean up environment variables
	os.Unsetenv(EnvMCPServerPID)
	os.Unsetenv(EnvMCPServerWorkspace)
	os.Unsetenv(EnvMCPServerAddr)
	os.Unsetenv(EnvMCPServerPort)

	log.Printf("Server stopped")
}

func checkStatus(pidFile string) {
	pid, err := readPidFile(pidFile)
	if err != nil {
		fmt.Println("No server running")
		os.Exit(1)
	}

	if !isProcessRunning(pid) {
		fmt.Println("No server running")
		// Clean up stale PID file
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to remove stale PID file: %v", err)
		}
		os.Exit(1)
	}

	fmt.Printf("Server running with PID %d\n", pid)
}

func readPidFile(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, fmt.Errorf("invalid PID file format: %v", err)
	}

	return pid, nil
}

func writePidFile(pidFile string, pid int) error {
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix systems, FindProcess always succeeds, so we need to send signal 0
	// to check if the process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func setupSignalHandler(process *os.Process, pidFile string) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		if *verbose {
			log.Printf("Received signal %v, stopping server", sig)
		}

		// Forward the signal to the server process
		if err := process.Signal(sig); err != nil {
			log.Printf("Warning: Failed to forward signal to process: %v", err)
		}

		// Wait briefly for the process to exit
		time.Sleep(500 * time.Millisecond)

		// Clean up PID file
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to remove PID file: %v", err)
		}

		// Exit with the same signal code
		os.Exit(128 + int(sig.(syscall.Signal)))
	}()
}
