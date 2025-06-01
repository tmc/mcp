package mcpscripttest

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"rsc.io/script"
)

// serverManager manages MCP server processes for tests
type serverManager struct {
	mu      sync.Mutex
	servers map[string]*exec.Cmd
	stdinPipes  map[string]io.WriteCloser
	stdoutPipes map[string]io.ReadCloser
	lastOutput  map[string]string // Tracks the last output from each server
}

// Global server manager instance
var serverMgr = &serverManager{
	servers:     make(map[string]*exec.Cmd),
	stdinPipes:  make(map[string]io.WriteCloser),
	stdoutPipes: make(map[string]io.ReadCloser),
	lastOutput:  make(map[string]string),
}

// Server capabilities detected during initialization
var serverCapabilities = make(map[string]interface{})

// registerServerCommands adds MCP server control commands to the engine
func registerServerCommands(e *script.Engine) {
	// Register the commands
	e.Cmds["mcp-server-start"] = mcpServerStartCmd
	e.Cmds["mcp-server-send"] = mcpServerSendCmd
	e.Cmds["mcp-server-stop"] = mcpServerStopCmd
	e.Cmds["mcp-server-output"] = mcpServerOutputCmd
	// e.Cmds["mcp-server-detect-capabilities"] = mcpServerDetectCapabilitiesCmd // Will be added later
	
	// Register conditions for checking server status and capabilities

	// Check if any server is running
	e.Conds["mcp_server_running"] = script.Condition("check if MCP server is running", func(s *script.State) (bool, error) {
		serverMgr.mu.Lock()
		defer serverMgr.mu.Unlock()

		return len(serverMgr.servers) > 0, nil
	})

	// Basic transport conditions - these should be detected from the server
	e.Conds["stdio"] = script.Condition("server supports stdio transport", func(s *script.State) (bool, error) {
		// We'll assume stdio is always supported when a server is running
		// since that's our default communication method
		serverMgr.mu.Lock()
		defer serverMgr.mu.Unlock()

		return len(serverMgr.servers) > 0, nil
	})

	// HTTP transport condition
	e.Conds["http"] = script.Condition("server supports http transport", func(s *script.State) (bool, error) {
		// Check if the MCP_SERVER_ADDR environment variable is set
		return os.Getenv("MCP_SERVER_ADDR") != "", nil
	})

	// SSE transport condition
	e.Conds["sse"] = script.Condition("server supports sse transport", func(s *script.State) (bool, error) {
		// This would need a more sophisticated check in a real implementation
		// For now, we'll just check if the HTTP address is set
		return os.Getenv("MCP_SERVER_ADDR") != "", nil
	})

	// Session support condition
	e.Conds["http_session"] = script.Condition("server supports http sessions", func(s *script.State) (bool, error) {
		// This would need to be detected by attempting to create a session
		return false, nil
	})

	// Multi-connection support
	e.Conds["multi_connection"] = script.Condition("server supports multiple connections", func(s *script.State) (bool, error) {
		// We'll assume multi-connection is supported if http/sse are supported
		return os.Getenv("MCP_SERVER_ADDR") != "", nil
	})

	// Special server capabilities for advanced tests
	e.Conds["test_server_delay"] = script.Condition("server supports delay simulation", func(s *script.State) (bool, error) {
		return false, nil
	})

	e.Conds["test_server_cancel"] = script.Condition("server supports cancellation", func(s *script.State) (bool, error) {
		return false, nil
	})

	e.Conds["test_server_validate_stdout"] = script.Condition("server supports stdout validation", func(s *script.State) (bool, error) {
		return false, nil
	})

	e.Conds["test_server_verbose"] = script.Condition("server supports verbose mode", func(s *script.State) (bool, error) {
		return false, nil
	})

	e.Conds["test_server_capture_headers"] = script.Condition("server supports header capture", func(s *script.State) (bool, error) {
		return false, nil
	})

	e.Conds["test_server_http_status"] = script.Condition("server supports HTTP status testing", func(s *script.State) (bool, error) {
		return false, nil
	})

	// Helper condition for MCP HTTP client
	e.Conds["has_mcp_http_client"] = script.Condition("MCP HTTP client is available", func(s *script.State) (bool, error) {
		// Check if an HTTP client command is available
		return os.Getenv("MCP_HTTP_CLIENT_CMD") != "", nil
	})

	// Helper condition for SSE client
	e.Conds["has_sse_client"] = script.Condition("SSE client is available", func(s *script.State) (bool, error) {
		// Check if an SSE client command is available
		return os.Getenv("MCP_SSE_CLIENT_CMD") != "", nil
	})
}

// mcpServerStartCmd starts an MCP server process
var mcpServerStartCmd = script.Command(
	script.CmdUsage{
		Summary: "start an MCP server process",
		Args:    "[name] command [arguments...]",
		Detail: []string{
			"Starts an MCP server process with the specified command and arguments.",
			"If name is provided, it will be used as an identifier for the server.",
			"Otherwise, the default name 'default' will be used.",
			"Use $MCP_SERVER_COMMAND to use the command specified via environment.",
		},
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		var (
			name string = "default"
			cmdArgs []string
		)

		// Check if any arguments were provided
		if len(args) == 0 {
			// No arguments provided, try to get from environment
			serverCmd := os.Getenv("MCP_SERVER_COMMAND")
			if serverCmd == "" {
				return nil, fmt.Errorf("no server command provided (set MCP_SERVER_COMMAND or provide command arguments)")
			}
			
			// Split the server command into arguments
			cmdArgs = strings.Fields(serverCmd)
		} else {
			// Arguments were provided
			// If first arg doesn't start with - or contain a /, it's a name
			if !strings.HasPrefix(args[0], "-") && !strings.Contains(args[0], "/") && !strings.HasPrefix(args[0], "$") {
				name = args[0]
				cmdArgs = args[1:]
			} else {
				cmdArgs = args
			}
		}

		// Handle variable expansion for $MCP_SERVER_COMMAND
		if len(cmdArgs) == 1 && strings.HasPrefix(cmdArgs[0], "$") {
			varName := strings.TrimPrefix(cmdArgs[0], "$")
			
			// Check environment variables
			if val := os.Getenv(varName); val != "" {
				cmdArgs = strings.Fields(val)
			} else {
				return nil, fmt.Errorf("variable %s not defined", cmdArgs[0])
			}
		}
		
		if len(cmdArgs) < 1 {
			return nil, fmt.Errorf("no command specified")
		}

		// Check if a server with this name is already running
		serverMgr.mu.Lock()
		defer serverMgr.mu.Unlock()
		
		if _, exists := serverMgr.servers[name]; exists {
			return nil, fmt.Errorf("server with name %q is already running", name)
		}

		// Look up the command path
		path, err := exec.LookPath(cmdArgs[0])
		if err != nil {
			return nil, fmt.Errorf("command not found: %v", err)
		}

		// Create the command
		cmd := exec.CommandContext(s.Context(), path, cmdArgs[1:]...)
		cmd.Dir = s.Getwd()
		cmd.Env = s.Environ()

		// Create pipes for stdin, stdout, and stderr
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			stdin.Close()
			return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			stdin.Close()
			stdout.Close()
			return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			stdin.Close()
			stdout.Close()
			stderr.Close()
			return nil, fmt.Errorf("failed to start server: %v", err)
		}

		// Register the server and pipes
		serverMgr.servers[name] = cmd
		serverMgr.stdinPipes[name] = stdin
		serverMgr.stdoutPipes[name] = stdout

		// Set up a goroutine to wait for the process to complete
		go func() {
			// Capture stderr output for debugging
			stderrOutput, _ := io.ReadAll(stderr)
			
			// Wait for the command to complete
			err := cmd.Wait()
			
			// Clean up the server registration
			serverMgr.mu.Lock()
			delete(serverMgr.servers, name)
			delete(serverMgr.stdinPipes, name)
			delete(serverMgr.stdoutPipes, name)
			serverMgr.mu.Unlock()
			
			if err != nil && s.Context().Err() != context.Canceled {
				// Log the error for debugging
				fmt.Fprintf(os.Stderr, "Server %q exited with error: %v\nStderr: %s\n", 
					name, err, string(stderrOutput))
			}
		}()

		// Return the initial stderr output
		return func(s *script.State) (string, string, error) {
			// Read a small amount from stderr to capture the startup message
			buffer := make([]byte, 1024)
			n, _ := stderr.Read(buffer)
			return "", string(buffer[:n]), nil
		}, nil
	},
)

// mcpServerSendCmd sends data to a running MCP server
var mcpServerSendCmd = script.Command(
	script.CmdUsage{
		Summary: "send data to a running MCP server",
		Args:    "[--name=server] [--timeout=seconds] [--async]",
		Detail: []string{
			"Sends data from stdin to the specified server and returns the response.",
			"If --name is not provided, the default server will be used.",
			"If --timeout is provided, the command will wait at most that many seconds for a response.",
			"If --async is provided, the command returns immediately and responses can be checked with mcp-server-output.",
		},
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		// Parse arguments
		name := "default"
		timeout := 5.0 // Default timeout in seconds
		async := false
		
		for _, arg := range args {
			if strings.HasPrefix(arg, "--name=") {
				name = strings.TrimPrefix(arg, "--name=")
			} else if strings.HasPrefix(arg, "--timeout=") {
				fmt.Sscanf(strings.TrimPrefix(arg, "--timeout="), "%f", &timeout)
			} else if arg == "--async" {
				async = true
			}
		}

		// Check for pending stdin content
		stdinStore.Lock()
		stdinContent, ok := stdinStore.pendingContent[s]
		if ok {
			delete(stdinStore.pendingContent, s)
		}
		stdinStore.Unlock()

		if !ok || stdinContent == "" {
			return nil, fmt.Errorf("no input data available")
		}

		// Get the server pipes
		serverMgr.mu.Lock()
		stdin, stdinOk := serverMgr.stdinPipes[name]
		stdout, stdoutOk := serverMgr.stdoutPipes[name]
		serverMgr.mu.Unlock()

		if !stdinOk || !stdoutOk {
			return nil, fmt.Errorf("server %q is not running", name)
		}

		// Ensure the content ends with a newline
		if !strings.HasSuffix(stdinContent, "\n") {
			stdinContent += "\n"
		}

		// Write the input to the server's stdin
		if _, err := stdin.Write([]byte(stdinContent)); err != nil {
			return nil, fmt.Errorf("failed to write to server: %v", err)
		}

		if async {
			// In async mode, return immediately
			return func(s *script.State) (string, string, error) {
				return "Request sent asynchronously\n", "", nil
			}, nil
		}

		// Otherwise, wait for a response
		return func(s *script.State) (string, string, error) {
			timeout := time.Duration(timeout * float64(time.Second))
			deadline := time.Now().Add(timeout)

			// Start reading the response
			var response strings.Builder
			buffer := make([]byte, 4096)
			responseChan := make(chan string, 1)
			errChan := make(chan error, 1)

			go func() {
				for {
					// Set a read timeout
					stdoutReadCloser := stdout.(io.ReadCloser)
					if tcpConn, ok := stdoutReadCloser.(*os.File); ok {
						tcpConn.SetReadDeadline(deadline)
					}

					n, err := stdout.Read(buffer)
					if n > 0 {
						response.Write(buffer[:n])
						// Check if we have a complete JSON response
						respStr := response.String()
						if strings.Contains(respStr, "\n") {
							responseChan <- respStr
							return
						}
					}
					if err != nil {
						if err == io.EOF || strings.Contains(err.Error(), "timeout") {
							responseChan <- response.String()
							return
						}
						errChan <- err
						return
					}
				}
			}()

			select {
			case resp := <-responseChan:
				// Save the output for potential later retrieval
				serverMgr.mu.Lock()
				serverMgr.lastOutput[name] = resp
				serverMgr.mu.Unlock()
				return resp, "", nil
			case err := <-errChan:
				return "", "", fmt.Errorf("error reading response: %v", err)
			case <-time.After(timeout):
				// Save whatever we got
				partial := response.String()
				serverMgr.mu.Lock()
				serverMgr.lastOutput[name] = partial
				serverMgr.mu.Unlock()
				return partial, "", fmt.Errorf("timeout waiting for response")
			}
		}, nil
	},
)

// mcpServerStopCmd stops a running MCP server
var mcpServerStopCmd = script.Command(
	script.CmdUsage{
		Summary: "stop a running MCP server",
		Args:    "[name]",
		Detail: []string{
			"Stops the specified MCP server process.",
			"If name is not provided, the default server will be stopped.",
		},
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		name := "default"
		if len(args) > 0 {
			name = args[0]
		}

		serverMgr.mu.Lock()
		cmd, ok := serverMgr.servers[name]
		stdinPipe, stdinOk := serverMgr.stdinPipes[name]
		stdoutPipe, stdoutOk := serverMgr.stdoutPipes[name]
		serverMgr.mu.Unlock()

		if !ok {
			return nil, fmt.Errorf("server %q is not running", name)
		}

		// Close the pipes
		if stdinOk && stdinPipe != nil {
			stdinPipe.Close()
		}
		if stdoutOk && stdoutPipe != nil {
			stdoutPipe.Close()
		}

		// Try graceful shutdown first
		cmd.Process.Signal(syscall.SIGTERM)

		// Wait for the process to exit with a timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(5 * time.Second):
			// Force kill after timeout
			cmd.Process.Kill()
			<-done
		}

		// Clean up
		serverMgr.mu.Lock()
		delete(serverMgr.servers, name)
		delete(serverMgr.stdinPipes, name)
		delete(serverMgr.stdoutPipes, name)
		delete(serverMgr.lastOutput, name)
		serverMgr.mu.Unlock()

		return func(s *script.State) (string, string, error) {
			return fmt.Sprintf("Server %q stopped\n", name), "", nil
		}, nil
	},
)

// mcpServerOutputCmd retrieves the last output from a server
var mcpServerOutputCmd = script.Command(
	script.CmdUsage{
		Summary: "get the last output from a server",
		Args:    "[name]",
		Detail: []string{
			"Retrieves the last output from the specified server.",
			"If name is not provided, the default server will be used.",
		},
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		name := "default"
		if len(args) > 0 {
			name = args[0]
		}

		serverMgr.mu.Lock()
		output, ok := serverMgr.lastOutput[name]
		serverMgr.mu.Unlock()

		if !ok {
			return nil, fmt.Errorf("no output available for server %q", name)
		}

		return func(s *script.State) (string, string, error) {
			return output, "", nil
		}, nil
	},
)