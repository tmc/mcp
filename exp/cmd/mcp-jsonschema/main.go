package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/mcp"
)

var (
	configFile  = flag.String("config", "", "Path to MCP server configuration file")
	serverCmd   = flag.String("server", "", "Command to start the MCP server")
	timeout     = flag.Duration("timeout", 30*time.Second, "Timeout for server operations")
	output      = flag.String("output", "", "Output file for JSON schema. If not specified, prints to stdout")
	prettyPrint = flag.Bool("pretty", true, "Pretty-print the JSON output")
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options]\n\n", filepath.Base(os.Args[0]))
		fmt.Println("Extracts JSON schema from an MCP server or configuration file.")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *configFile == "" && *serverCmd == "" {
		fmt.Fprintln(os.Stderr, "Error: Either --config or --server must be specified")
		flag.Usage()
		os.Exit(1)
	}

	if *configFile != "" && *serverCmd != "" {
		fmt.Fprintln(os.Stderr, "Error: Only one of --config or --server can be specified")
		flag.Usage()
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	var client *mcp.Client
	var err error

	// Connect to server based on the provided flags
	if *serverCmd != "" {
		client, err = connectToServer(ctx, *serverCmd)
	} else {
		client, err = connectToConfig(ctx, *configFile)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to MCP server: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Get tools and extract schemas
	schemas, err := extractSchemas(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting schemas: %v\n", err)
		os.Exit(1)
	}

	// Output schemas
	if err := outputSchemas(schemas, *output, *prettyPrint); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}
}

// connectToServer starts the provided server command and connects to it
func connectToServer(ctx context.Context, serverCmd string) (*mcp.Client, error) {
	// Split the command into command and arguments
	parts := strings.Fields(serverCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty server command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the server process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	// Create a custom transport that communicates over the command's stdin/stdout
	transport := &stdioTransport{
		reader: stdout,
		writer: stdin,
	}

	// Create and connect the client
	client, err := mcp.NewClient(context.Background(), transport)
	if err != nil {
		cmd.Process.Kill() // Kill the process if we fail to connect
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Initialize the client
	result, err := client.Initialize(ctx, mcp.InitializeRequest{
		ClientInfo: mcp.Implementation{
			Name:    "mcp-jsonschema",
			Version: "1.0.0",
		},
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
	})
	if err != nil {
		client.Close()
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	// Log the server info
	fmt.Printf("Connected to %s (version %s)\n", result.ServerInfo.Name, result.ServerInfo.Version)

	return client, nil
}

// stdioTransport implements the mcp.Transport interface
type stdioTransport struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

// Dial implements the mcp.Transport interface
func (t *stdioTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return &stdioReadWriteCloser{
		reader: t.reader,
		writer: t.writer,
	}, nil
}

// stdioReadWriteCloser implements io.ReadWriteCloser
type stdioReadWriteCloser struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (rwc *stdioReadWriteCloser) Read(p []byte) (int, error) {
	return rwc.reader.Read(p)
}

func (rwc *stdioReadWriteCloser) Write(p []byte) (int, error) {
	return rwc.writer.Write(p)
}

func (rwc *stdioReadWriteCloser) Close() error {
	err1 := rwc.reader.Close()
	err2 := rwc.writer.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// connectToConfig reads the specified config file and connects to the server
func connectToConfig(ctx context.Context, configFile string) (*mcp.Client, error) {
	// Read the configuration file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the configuration
	var config struct {
		Name      string            `json:"name"`
		Version   string            `json:"version"`
		Command   string            `json:"command"`
		Transport string            `json:"transport"`
		Env       map[string]string `json:"environment"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Command == "" {
		return nil, fmt.Errorf("command not specified in config file")
	}

	// Launch the server process
	parts := strings.Fields(config.Command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty server command in config file")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	// Set environment variables if specified
	if len(config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the server process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	// Create a transport based on the config
	var transport mcp.Transport
	switch strings.ToLower(config.Transport) {
	case "", "stdio":
		transport = &stdioTransport{
			reader: stdout,
			writer: stdin,
		}
	default:
		cmd.Process.Kill()
		return nil, fmt.Errorf("unsupported transport type: %s", config.Transport)
	}

	// Create and connect the client
	client, err := mcp.NewClient(context.Background(), transport)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Initialize the client
	result, err := client.Initialize(ctx, mcp.InitializeRequest{
		ClientInfo: mcp.Implementation{
			Name:    "mcp-jsonschema",
			Version: "1.0.0",
		},
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
	})
	if err != nil {
		client.Close()
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	// Log the server info
	fmt.Printf("Connected to %s (version %s)\n", result.ServerInfo.Name, result.ServerInfo.Version)

	return client, nil
}

// extractSchemas retrieves tools from the server and extracts their JSON schemas
func extractSchemas(ctx context.Context, client *mcp.Client) (map[string]json.RawMessage, error) {
	// List available tools
	result, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// Extract schema for each tool
	schemas := make(map[string]json.RawMessage)
	for _, tool := range result.Tools {
		if tool.InputSchema != nil {
			schemas[tool.Name] = tool.InputSchema
		}
	}

	return schemas, nil
}

// outputSchemas writes the schemas to the specified output
func outputSchemas(schemas map[string]json.RawMessage, outputFile string, prettyPrint bool) error {
	var data []byte
	var err error

	if prettyPrint {
		data, err = json.MarshalIndent(schemas, "", "  ")
	} else {
		data, err = json.Marshal(schemas)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal schemas: %w", err)
	}

	if outputFile == "" {
		// Write to stdout
		fmt.Println(string(data))
		return nil
	}

	// Write to file
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	fmt.Printf("JSON schemas written to %s\n", outputFile)
	return nil
}
