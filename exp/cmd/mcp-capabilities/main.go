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
	configFile        = flag.String("config", "", "Path to MCP server configuration file")
	serverCmd         = flag.String("server", "", "Command to start the MCP server")
	timeout           = flag.Duration("timeout", 30*time.Second, "Timeout for server operations")
	output            = flag.String("output", "", "Output file for capability report. If not specified, prints to stdout")
	jsonFormat        = flag.Bool("json", false, "Output in JSON format instead of human-readable text")
	checkTools        = flag.Bool("tools", true, "Check if server supports tools")
	checkPrompts      = flag.Bool("prompts", true, "Check if server supports prompts")
	checkResources    = flag.Bool("resources", true, "Check if server supports resources")
	checkExperimental = flag.Bool("experimental", true, "Check if server has experimental capabilities")
)

// CapabilityReport represents the capabilities of an MCP server
type CapabilityReport struct {
	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
	Protocol     string                 `json:"protocol"`
	Tools        bool                   `json:"tools"`
	ToolsList    []string               `json:"toolsList,omitempty"`
	Prompts      bool                   `json:"prompts"`
	PromptsList  []string               `json:"promptsList,omitempty"`
	Resources    bool                   `json:"resources"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Instructions string                 `json:"instructions,omitempty"`
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options]\n\n", filepath.Base(os.Args[0]))
		fmt.Println("Reports capabilities of an MCP server.")
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

	// Get and output server capabilities
	report, err := getCapabilities(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining server capabilities: %v\n", err)
		os.Exit(1)
	}

	// Output capabilities
	if err := outputCapabilities(report, *output, *jsonFormat); err != nil {
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

	return client, nil
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

// getCapabilities determines the capabilities of the MCP server
func getCapabilities(ctx context.Context, client *mcp.Client) (*CapabilityReport, error) {
	// Initialize the client
	initResult, err := client.Initialize(ctx, mcp.InitializeRequest{
		ClientInfo: mcp.Implementation{
			Name:    "mcp-capabilities",
			Version: "1.0.0",
		},
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	report := &CapabilityReport{
		Protocol:     initResult.ProtocolVersion,
		Instructions: initResult.Instructions,
	}

	// Set server info
	report.ServerInfo.Name = initResult.ServerInfo.Name
	report.ServerInfo.Version = initResult.ServerInfo.Version

	// Check for experimental capabilities
	if *checkExperimental && initResult.Capabilities.Experimental != nil {
		report.Experimental = initResult.Capabilities.Experimental
	}

	// Check for tools support
	if *checkTools {
		toolsList, err := client.ListTools(ctx, mcp.ListToolsRequest{})
		if err == nil {
			report.Tools = true
			for _, tool := range toolsList.Tools {
				report.ToolsList = append(report.ToolsList, tool.Name)
			}
		} else {
			// If we get an error, assume tools are not supported
			report.Tools = false
		}
	}

	// Check for prompts support
	if *checkPrompts {
		promptsList, err := client.ListPrompts(ctx, mcp.ListPromptsRequest{})
		if err == nil {
			report.Prompts = true
			for _, prompt := range promptsList.Prompts {
				report.PromptsList = append(report.PromptsList, prompt.Name)
			}
		} else {
			// If we get an error, assume prompts are not supported
			report.Prompts = false
		}
	}

	// Check for resources support
	if *checkResources {
		_, err := client.ListResources(ctx, mcp.ListResourcesRequest{})
		if err == nil {
			report.Resources = true
		} else {
			// If we get an error, assume resources are not supported
			report.Resources = false
		}
	}

	return report, nil
}

// outputCapabilities writes the capability report to the specified output
func outputCapabilities(report *CapabilityReport, outputFile string, jsonFormat bool) error {
	var data []byte
	var err error

	if jsonFormat {
		data, err = json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	} else {
		// Create a human-readable format
		builder := strings.Builder{}

		// Server info
		builder.WriteString(fmt.Sprintf("Server: %s (version %s)\n", report.ServerInfo.Name, report.ServerInfo.Version))
		builder.WriteString(fmt.Sprintf("Protocol Version: %s\n", report.Protocol))

		// Capabilities
		builder.WriteString("\nCapabilities:\n")
		builder.WriteString(fmt.Sprintf("- Tools: %v\n", report.Tools))
		if report.Tools && len(report.ToolsList) > 0 {
			builder.WriteString("  Available tools:\n")
			for _, tool := range report.ToolsList {
				builder.WriteString(fmt.Sprintf("  - %s\n", tool))
			}
		}

		builder.WriteString(fmt.Sprintf("- Prompts: %v\n", report.Prompts))
		if report.Prompts && len(report.PromptsList) > 0 {
			builder.WriteString("  Available prompts:\n")
			for _, prompt := range report.PromptsList {
				builder.WriteString(fmt.Sprintf("  - %s\n", prompt))
			}
		}

		builder.WriteString(fmt.Sprintf("- Resources: %v\n", report.Resources))

		// Experimental capabilities
		if report.Experimental != nil && len(report.Experimental) > 0 {
			builder.WriteString("\nExperimental Capabilities:\n")
			expJson, _ := json.MarshalIndent(report.Experimental, "", "  ")
			builder.WriteString(string(expJson))
			builder.WriteString("\n")
		}

		// Server instructions
		if report.Instructions != "" {
			builder.WriteString("\nServer Instructions:\n")
			builder.WriteString(report.Instructions)
			builder.WriteString("\n")
		}

		data = []byte(builder.String())
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

	fmt.Printf("Capability report written to %s\n", outputFile)
	return nil
}
