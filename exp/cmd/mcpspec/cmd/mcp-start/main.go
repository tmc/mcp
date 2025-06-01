package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/mcp/cmd/mcpspec/internal/command"
	"github.com/tmc/mcp/cmd/mcpspec/internal/io"
	"github.com/tmc/mcp/cmd/mcpspec/internal/jsonrpc"
)

// ServerConfig represents the configuration for an MCP server.
type ServerConfig struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Transport   string                 `json:"transport"` // "http", "stdio", "websocket"
	Host        string                 `json:"host,omitempty"`
	Port        int                    `json:"port,omitempty"`
	Command     string                 `json:"command,omitempty"`
	Environment map[string]string      `json:"environment,omitempty"`
	Tools       []ToolDefinition       `json:"tools,omitempty"`
	InitParams  map[string]interface{} `json:"init_params,omitempty"`
}

// ToolDefinition represents the definition of an MCP tool.
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Schema      interface{} `json:"schema"`
}

// ServerState represents the state of a running server.
type ServerState struct {
	Config     *ServerConfig
	Process    *os.Process
	Stdout     io.ReadCloser
	Stdin      io.WriteCloser
	Stderr     io.ReadCloser
	StopSignal chan bool
}

// StartCommand represents the mcp-start command.
type StartCommand struct {
	command.BaseCommand
	configFile  string
	timeout     int
	server      string
	debug       bool
	noInit      bool
	dryRun      bool
	interactive bool
}

// NewStartCommand creates a new StartCommand.
func NewStartCommand() *StartCommand {
	return &StartCommand{}
}

// Name returns the command name.
func (c *StartCommand) Name() string {
	return "mcp-start"
}

// Usage returns the command usage.
func (c *StartCommand) Usage() string {
	return "Usage: mcp-start [options]\n\n" +
		"Options:\n" +
		"  -c, --config <file>      Server configuration file (required)\n" +
		"  -t, --timeout <seconds>  Server timeout in seconds (default: 0, no timeout)\n" +
		"  -s, --server <name>      Server name in configuration file\n" +
		"  -d, --debug              Enable debug output\n" +
		"  --no-init                Skip server initialization\n" +
		"  --dry-run                Parse configuration but don't start server\n" +
		"  -i, --interactive        Interactive mode (read commands from stdin)\n"
}

// Execute runs the command.
func (c *StartCommand) Execute(ctx context.Context, args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	fs.StringVar(&c.configFile, "c", "", "Server configuration file (required)")
	fs.StringVar(&c.configFile, "config", "", "Server configuration file (required)")
	fs.IntVar(&c.timeout, "t", 0, "Server timeout in seconds (default: 0, no timeout)")
	fs.IntVar(&c.timeout, "timeout", 0, "Server timeout in seconds (default: 0, no timeout)")
	fs.StringVar(&c.server, "s", "", "Server name in configuration file")
	fs.StringVar(&c.server, "server", "", "Server name in configuration file")
	fs.BoolVar(&c.debug, "d", false, "Enable debug output")
	fs.BoolVar(&c.debug, "debug", false, "Enable debug output")
	fs.BoolVar(&c.noInit, "no-init", false, "Skip server initialization")
	fs.BoolVar(&c.dryRun, "dry-run", false, "Parse configuration but don't start server")
	fs.BoolVar(&c.interactive, "i", false, "Interactive mode (read commands from stdin)")
	fs.BoolVar(&c.interactive, "interactive", false, "Interactive mode (read commands from stdin)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate flags
	if c.configFile == "" {
		return fmt.Errorf("config file is required")
	}

	// Load the configuration file
	config, err := c.loadConfig()
	if err != nil {
		return err
	}

	// Validate the configuration
	if err := c.validateConfig(config); err != nil {
		return err
	}

	// Print configuration summary
	c.printConfigSummary(config)

	// If dry-run, just return without starting the server
	if c.dryRun {
		fmt.Println("Dry run successful. Configuration is valid.")
		return nil
	}

	// Start the server
	return c.startServer(ctx, config)
}

// loadConfig loads the server configuration from the specified file.
func (c *StartCommand) loadConfig() (*ServerConfig, error) {
	data, err := os.ReadFile(c.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// validateConfig validates the server configuration.
func (c *StartCommand) validateConfig(config *ServerConfig) error {
	// Check required fields
	if config.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if config.Version == "" {
		return fmt.Errorf("server version is required")
	}
	if config.Transport == "" {
		return fmt.Errorf("transport is required")
	}

	// Validate transport-specific configuration
	switch config.Transport {
	case "http", "websocket":
		if config.Port == 0 {
			config.Port = 8080 // Default port
		}
		if config.Host == "" {
			config.Host = "localhost" // Default host
		}
	case "stdio":
		if config.Command == "" {
			return fmt.Errorf("command is required for stdio transport")
		}
	default:
		return fmt.Errorf("unsupported transport type: %s", config.Transport)
	}

	return nil
}

// printConfigSummary prints a summary of the server configuration.
func (c *StartCommand) printConfigSummary(config *ServerConfig) {
	fmt.Printf("Server: %s (version %s)\n", config.Name, config.Version)
	fmt.Printf("Transport: %s\n", config.Transport)

	switch config.Transport {
	case "http", "websocket":
		fmt.Printf("Endpoint: %s:%d\n", config.Host, config.Port)
	case "stdio":
		fmt.Printf("Command: %s\n", config.Command)
	}

	if len(config.Tools) > 0 {
		fmt.Println("Tools:")
		for _, tool := range config.Tools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
	}

	if c.debug {
		// Print environment variables
		if len(config.Environment) > 0 {
			fmt.Println("Environment:")
			for key, value := range config.Environment {
				fmt.Printf("  %s=%s\n", key, value)
			}
		}

		// Print init parameters
		if config.InitParams != nil {
			fmt.Println("Init Parameters:")
			data, err := json.MarshalIndent(config.InitParams, "  ", "  ")
			if err != nil {
				fmt.Printf("  Error marshaling init params: %v\n", err)
			} else {
				fmt.Println(string(data))
			}
		}
	}
}

// startServer starts the MCP server based on the configuration.
func (c *StartCommand) startServer(ctx context.Context, config *ServerConfig) error {
	fmt.Printf("Starting MCP server: %s (version %s)\n", config.Name, config.Version)

	var serverState *ServerState
	var err error

	// Start the server based on the transport type
	switch config.Transport {
	case "stdio":
		serverState, err = c.startStdioServer(config)
	case "http":
		serverState, err = c.startHTTPServer(config)
	case "websocket":
		serverState, err = c.startWebSocketServer(config)
	default:
		return fmt.Errorf("unsupported transport type: %s", config.Transport)
	}

	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize the server
	if !c.noInit {
		if err := c.initializeServer(serverState); err != nil {
			c.shutdownServer(serverState)
			return fmt.Errorf("failed to initialize server: %w", err)
		}
	}

	// Set up timeout if specified
	var timeoutChan <-chan time.Time
	if c.timeout > 0 {
		timeoutChan = time.After(time.Duration(c.timeout) * time.Second)
	}

	// Handle interactive mode
	var interactiveChan chan string
	if c.interactive {
		interactiveChan = make(chan string)
		go c.handleInteractive(interactiveChan, serverState)
	}

	// Start a goroutine to read server output
	outputChan := make(chan string)
	errChan := make(chan error)
	go c.readServerOutput(serverState, outputChan, errChan)

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context cancelled, shutting down server")
			c.shutdownServer(serverState)
			return nil

		case sig := <-sigChan:
			fmt.Printf("Received signal %s, shutting down server\n", sig)
			c.shutdownServer(serverState)
			return nil

		case <-timeoutChan:
			fmt.Printf("Server timeout reached (%d seconds), shutting down\n", c.timeout)
			c.shutdownServer(serverState)
			return nil

		case output := <-outputChan:
			fmt.Println("Server:", output)

		case err := <-errChan:
			fmt.Printf("Server error: %v\n", err)
			c.shutdownServer(serverState)
			return err

		case cmd := <-interactiveChan:
			if cmd == "exit" || cmd == "quit" {
				fmt.Println("Exiting interactive mode, shutting down server")
				c.shutdownServer(serverState)
				return nil
			}
			// Process interactive command
			c.handleCommand(cmd, serverState)
		}
	}
}

// startStdioServer starts an MCP server using stdio transport.
func (c *StartCommand) startStdioServer(config *ServerConfig) (*ServerState, error) {
	// Create the command
	cmd := exec.Command("sh", "-c", config.Command)

	// Set up environment variables
	env := os.Environ()
	for key, value := range config.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = env

	// Set up stdin, stdout, and stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Create the server state
	serverState := &ServerState{
		Config:     config,
		Process:    cmd.Process,
		Stdout:     stdout,
		Stdin:      stdin,
		Stderr:     stderr,
		StopSignal: make(chan bool),
	}

	return serverState, nil
}

// startHTTPServer starts an MCP server using HTTP transport.
func (c *StartCommand) startHTTPServer(config *ServerConfig) (*ServerState, error) {
	// For now, just print that we're starting an HTTP server
	fmt.Printf("Starting HTTP server on %s:%d (not yet implemented)\n", config.Host, config.Port)

	// This will need to be implemented to actually start an HTTP server
	// For now, just return a dummy state
	return &ServerState{
		Config:     config,
		StopSignal: make(chan bool),
	}, nil
}

// startWebSocketServer starts an MCP server using WebSocket transport.
func (c *StartCommand) startWebSocketServer(config *ServerConfig) (*ServerState, error) {
	// For now, just print that we're starting a WebSocket server
	fmt.Printf("Starting WebSocket server on %s:%d (not yet implemented)\n", config.Host, config.Port)

	// This will need to be implemented to actually start a WebSocket server
	// For now, just return a dummy state
	return &ServerState{
		Config:     config,
		StopSignal: make(chan bool),
	}, nil
}

// initializeServer initializes the MCP server.
func (c *StartCommand) initializeServer(serverState *ServerState) error {
	// Skip if we're not using a stdio transport
	if serverState.Config.Transport != "stdio" {
		return nil
	}

	// Create the initialize request
	initParams := serverState.Config.InitParams
	if initParams == nil {
		initParams = make(map[string]interface{})
	}

	msg, err := jsonrpc.NewRequest("initialize", initParams, 1)
	if err != nil {
		return fmt.Errorf("failed to create initialize request: %w", err)
	}

	// Send the request
	encoder := json.NewEncoder(serverState.Stdin)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("failed to send initialize request: %w", err)
	}

	// Read the response (this will be handled by the output reader goroutine)
	fmt.Println("Sent initialize request to server")

	return nil
}

// shutdownServer shuts down the MCP server.
func (c *StartCommand) shutdownServer(serverState *ServerState) {
	fmt.Println("Shutting down server...")

	// Signal the stop goroutine
	close(serverState.StopSignal)

	// Skip if we're not using a stdio transport
	if serverState.Config.Transport != "stdio" {
		return
	}

	// Send shutdown request
	msg, err := jsonrpc.NewRequest("shutdown", nil, 2)
	if err != nil {
		fmt.Printf("Failed to create shutdown request: %v\n", err)
		return
	}

	// Send the request
	encoder := json.NewEncoder(serverState.Stdin)
	if err := encoder.Encode(msg); err != nil {
		fmt.Printf("Failed to send shutdown request: %v\n", err)
	}

	// Kill the process if it still exists
	if serverState.Process != nil {
		serverState.Process.Signal(syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
		serverState.Process.Kill()
	}
}

// readServerOutput reads output from the server.
func (c *StartCommand) readServerOutput(serverState *ServerState, outputChan chan<- string, errChan chan<- error) {
	// Skip if we're not using a stdio transport
	if serverState.Config.Transport != "stdio" {
		return
	}

	// Read from stdout
	go func() {
		buffer := make([]byte, 4096)
		for {
			select {
			case <-serverState.StopSignal:
				return
			default:
				n, err := serverState.Stdout.Read(buffer)
				if err != nil {
					errChan <- fmt.Errorf("failed to read from stdout: %w", err)
					return
				}
				if n > 0 {
					outputChan <- string(buffer[:n])
				}
			}
		}
	}()

	// Read from stderr
	go func() {
		buffer := make([]byte, 4096)
		for {
			select {
			case <-serverState.StopSignal:
				return
			default:
				n, err := serverState.Stderr.Read(buffer)
				if err != nil {
					errChan <- fmt.Errorf("failed to read from stderr: %w", err)
					return
				}
				if n > 0 {
					outputChan <- "stderr: " + string(buffer[:n])
				}
			}
		}
	}()
}

// handleInteractive handles interactive mode.
func (c *StartCommand) handleInteractive(cmdChan chan<- string, serverState *ServerState) {
	fmt.Println("Interactive mode enabled. Type 'exit' or 'quit' to exit.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := scanner.Text()
		cmdChan <- cmd
	}
}

// handleCommand handles an interactive command.
func (c *StartCommand) handleCommand(cmd string, serverState *ServerState) {
	// Skip if we're not using a stdio transport
	if serverState.Config.Transport != "stdio" {
		fmt.Println("Interactive commands are only supported for stdio transport")
		return
	}

	// Parse the command
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return
	}

	switch fields[0] {
	case "call":
		if len(fields) < 2 {
			fmt.Println("Usage: call <method> [<params>]")
			return
		}
		method := fields[1]
		var params interface{}
		if len(fields) > 2 {
			if err := json.Unmarshal([]byte(strings.Join(fields[2:], " ")), &params); err != nil {
				fmt.Printf("Failed to parse params: %v\n", err)
				return
			}
		}
		msg, err := jsonrpc.NewRequest(method, params, 100)
		if err != nil {
			fmt.Printf("Failed to create request: %v\n", err)
			return
		}
		encoder := json.NewEncoder(serverState.Stdin)
		if err := encoder.Encode(msg); err != nil {
			fmt.Printf("Failed to send request: %v\n", err)
			return
		}
		fmt.Printf("Sent request: %s\n", method)

	case "notify":
		if len(fields) < 2 {
			fmt.Println("Usage: notify <method> [<params>]")
			return
		}
		method := fields[1]
		var params interface{}
		if len(fields) > 2 {
			if err := json.Unmarshal([]byte(strings.Join(fields[2:], " ")), &params); err != nil {
				fmt.Printf("Failed to parse params: %v\n", err)
				return
			}
		}
		msg, err := jsonrpc.NewNotification(method, params)
		if err != nil {
			fmt.Printf("Failed to create notification: %v\n", err)
			return
		}
		encoder := json.NewEncoder(serverState.Stdin)
		if err := encoder.Encode(msg); err != nil {
			fmt.Printf("Failed to send notification: %v\n", err)
			return
		}
		fmt.Printf("Sent notification: %s\n", method)

	case "help":
		fmt.Println("Available commands:")
		fmt.Println("  call <method> [<params>] - Send a request to the server")
		fmt.Println("  notify <method> [<params>] - Send a notification to the server")
		fmt.Println("  help - Show this help message")
		fmt.Println("  exit, quit - Exit interactive mode and shut down the server")

	default:
		fmt.Printf("Unknown command: %s\n", fields[0])
		fmt.Println("Type 'help' for available commands")
	}
}

func main() {
	if err := NewStartCommand().Execute(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
