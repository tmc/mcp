package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents an MCP server configuration
type Config struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
	Command      string            `json:"command"`
	Transport    string            `json:"transport"`
	Environment  map[string]string `json:"environment,omitempty"`
	Instructions string            `json:"instructions,omitempty"`
	Tools        []Tool            `json:"tools,omitempty"`
	Prompts      []Prompt          `json:"prompts,omitempty"`
}

// Tool represents a tool definition in the configuration
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// Prompt represents a prompt definition in the configuration
type Prompt struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Arguments   map[string]string `json:"arguments,omitempty"`
}

var (
	create    = flag.String("create", "", "Create a new MCP server configuration file")
	edit      = flag.String("edit", "", "Edit an existing MCP server configuration file")
	validate  = flag.String("validate", "", "Validate an MCP server configuration file")
	format    = flag.String("format", "", "Format an MCP server configuration file")
	template  = flag.String("template", "", "Use a template to create a configuration (basic, filesystem, etc.)")
	name      = flag.String("name", "mcp-server", "Server name for new configurations")
	version   = flag.String("version", "1.0.0", "Server version for new configurations")
	command   = flag.String("command", "", "Server command for new configurations")
	transport = flag.String("transport", "stdio", "Transport type (stdio, http, sse)")
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options]\n\n", filepath.Base(os.Args[0]))
		fmt.Println("Manages MCP server configurations.")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Count the number of primary operations requested
	opCount := 0
	for _, op := range []string{*create, *edit, *validate, *format} {
		if op != "" {
			opCount++
		}
	}

	if opCount == 0 {
		fmt.Fprintln(os.Stderr, "Error: One of --create, --edit, --validate, or --format must be specified")
		flag.Usage()
		os.Exit(1)
	}

	if opCount > 1 {
		fmt.Fprintln(os.Stderr, "Error: Only one operation can be performed at a time")
		flag.Usage()
		os.Exit(1)
	}

	// Execute the requested operation
	var err error
	switch {
	case *create != "":
		err = createConfig(*create, *template)
	case *edit != "":
		err = editConfig(*edit)
	case *validate != "":
		err = validateConfig(*validate)
	case *format != "":
		err = formatConfig(*format)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// createConfig creates a new configuration file
func createConfig(filename, templateName string) error {
	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file %s already exists", filename)
	}

	config := Config{
		Name:      *name,
		Version:   *version,
		Command:   *command,
		Transport: *transport,
	}

	if templateName != "" {
		switch strings.ToLower(templateName) {
		case "basic":
			config = getBasicTemplate()
		case "filesystem":
			config = getFilesystemTemplate()
		case "calculator":
			config = getCalculatorTemplate()
		default:
			return fmt.Errorf("unknown template: %s", templateName)
		}
	}

	// Create the file
	return writeConfig(filename, config)
}

// editConfig opens a config file for editing
func editConfig(filename string) error {
	// Read the existing config
	config, err := readConfig(filename)
	if err != nil {
		return err
	}

	// Apply command line flags if provided
	if *name != "mcp-server" {
		config.Name = *name
	}
	if *version != "1.0.0" {
		config.Version = *version
	}
	if *command != "" {
		config.Command = *command
	}
	if *transport != "stdio" {
		config.Transport = *transport
	}

	// Write the updated config back
	return writeConfig(filename, config)
}

// validateConfig validates a configuration file
func validateConfig(filename string) error {
	config, err := readConfig(filename)
	if err != nil {
		return err
	}

	// Basic validation
	if config.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if config.Version == "" {
		return fmt.Errorf("server version is required")
	}
	if config.Command == "" {
		return fmt.Errorf("server command is required")
	}
	if config.Transport == "" {
		return fmt.Errorf("transport type is required")
	}

	// Validate transport type
	switch strings.ToLower(config.Transport) {
	case "stdio", "http", "sse":
		// These are supported
	default:
		return fmt.Errorf("unsupported transport type: %s", config.Transport)
	}

	// Tool name uniqueness
	toolNames := make(map[string]bool)
	for _, tool := range config.Tools {
		if tool.Name == "" {
			return fmt.Errorf("tool name is required")
		}
		if toolNames[tool.Name] {
			return fmt.Errorf("duplicate tool name: %s", tool.Name)
		}
		toolNames[tool.Name] = true
	}

	// Prompt name uniqueness
	promptNames := make(map[string]bool)
	for _, prompt := range config.Prompts {
		if prompt.Name == "" {
			return fmt.Errorf("prompt name is required")
		}
		if promptNames[prompt.Name] {
			return fmt.Errorf("duplicate prompt name: %s", prompt.Name)
		}
		promptNames[prompt.Name] = true
	}

	fmt.Printf("Configuration file %s is valid\n", filename)
	return nil
}

// formatConfig formats a configuration file
func formatConfig(filename string) error {
	config, err := readConfig(filename)
	if err != nil {
		return err
	}

	// Write the config back, which will format it
	return writeConfig(filename, config)
}

// readConfig reads a configuration from a file
func readConfig(filename string) (Config, error) {
	var config Config

	data, err := os.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return config, nil
}

// writeConfig writes a configuration to a file
func writeConfig(filename string, config Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Configuration written to %s\n", filename)
	return nil
}

// Template configurations

func getBasicTemplate() Config {
	return Config{
		Name:         "basic-server",
		Version:      "1.0.0",
		Description:  "A basic MCP server template",
		Command:      "./server",
		Transport:    "stdio",
		Instructions: "This is a basic MCP server that provides a simple echo tool.",
		Tools: []Tool{
			{
				Name:        "echo",
				Description: "Echoes back the input",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"message": {
							"type": "string",
							"description": "The message to echo back"
						}
					},
					"required": ["message"]
				}`),
			},
		},
	}
}

func getFilesystemTemplate() Config {
	return Config{
		Name:         "filesystem-server",
		Version:      "1.0.0",
		Description:  "A filesystem access MCP server",
		Command:      "./filesystem-server",
		Transport:    "stdio",
		Instructions: "This server provides access to the filesystem through MCP tools.",
		Tools: []Tool{
			{
				Name:        "listFiles",
				Description: "Lists files in a directory",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "The directory path to list"
						}
					},
					"required": ["path"]
				}`),
			},
			{
				Name:        "readFile",
				Description: "Reads the contents of a file",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "The file path to read"
						}
					},
					"required": ["path"]
				}`),
			},
			{
				Name:        "writeFile",
				Description: "Writes content to a file",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "The file path to write to"
						},
						"content": {
							"type": "string",
							"description": "The content to write to the file"
						}
					},
					"required": ["path", "content"]
				}`),
			},
		},
	}
}

func getCalculatorTemplate() Config {
	return Config{
		Name:         "calculator-server",
		Version:      "1.0.0",
		Description:  "A simple calculator MCP server",
		Command:      "./calculator-server",
		Transport:    "stdio",
		Instructions: "This server provides basic arithmetic operations through MCP tools.",
		Tools: []Tool{
			{
				Name:        "add",
				Description: "Adds two numbers",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"a": {
							"type": "number",
							"description": "First number"
						},
						"b": {
							"type": "number",
							"description": "Second number"
						}
					},
					"required": ["a", "b"]
				}`),
			},
			{
				Name:        "subtract",
				Description: "Subtracts the second number from the first",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"a": {
							"type": "number",
							"description": "First number"
						},
						"b": {
							"type": "number",
							"description": "Second number"
						}
					},
					"required": ["a", "b"]
				}`),
			},
			{
				Name:        "multiply",
				Description: "Multiplies two numbers",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"a": {
							"type": "number",
							"description": "First number"
						},
						"b": {
							"type": "number",
							"description": "Second number"
						}
					},
					"required": ["a", "b"]
				}`),
			},
			{
				Name:        "divide",
				Description: "Divides the first number by the second",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"a": {
							"type": "number",
							"description": "First number"
						},
						"b": {
							"type": "number",
							"description": "Second number (must not be zero)"
						}
					},
					"required": ["a", "b"]
				}`),
			},
		},
	}
}
