// Package cmd2mcpserver provides functionality to convert Go command-line tools into MCP servers
package cmd2mcpserver

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"
)

// Config represents the configuration for generating an MCP server
type Config struct {
	BinaryPath   string
	OutputDir    string
	ModuleName   string
	ServerName   string
	ToolName     string
	Description  string
}

// Generator generates an MCP server from a Go binary
type Generator struct {
	config    *Config
	flags     []FlagDef
	usesStdin bool
}

// FlagDef represents a command-line flag definition
type FlagDef struct {
	Name        string
	Type        string
	Default     string
	Description string
	Required    bool
}

// NewGenerator creates a new MCP server generator
func NewGenerator(config *Config) *Generator {
	return &Generator{
		config:    config,
		flags:     []FlagDef{},
		usesStdin: false,
	}
}

// SetFlags sets the flag definitions
func (g *Generator) SetFlags(flags []FlagDef) {
	g.flags = flags
}

// SetUsesStdin sets whether the program uses stdin
func (g *Generator) SetUsesStdin(usesStdin bool) {
	g.usesStdin = usesStdin
}

// GetToolDefinition returns the MCP tool definition
func (g *Generator) GetToolDefinition() map[string]interface{} {
	// Build input schema
	properties := make(map[string]interface{})
	required := []string{}

	// Add stdin parameter if the program uses stdin
	if g.usesStdin {
		properties["stdin"] = map[string]interface{}{
			"type":        "string",
			"description": "Optional input data to provide to the command via stdin",
		}
	}

	for _, flag := range g.flags {
		prop := map[string]interface{}{
			"type":        flag.Type,
			"description": flag.Description,
		}

		// Set default based on type
		switch flag.Type {
		case "boolean":
			if flag.Default == "true" {
				prop["default"] = true
			} else {
				prop["default"] = false
			}
		case "integer":
			// Parse integer default - for now just use the string
			prop["default"] = flag.Default
		default:
			// String default - remove quotes if present
			defaultVal := flag.Default
			if len(defaultVal) > 2 && defaultVal[0] == '"' && defaultVal[len(defaultVal)-1] == '"' {
				defaultVal = defaultVal[1:len(defaultVal)-1]
			}
			prop["default"] = defaultVal
		}

		properties[flag.Name] = prop

		if flag.Required {
			required = append(required, flag.Name)
		}
	}

	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}

	// Define the default output/return type
	returnType := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"stdout": map[string]interface{}{
				"type":        "string",
				"description": "Standard output from the command",
			},
			"stderr": map[string]interface{}{
				"type":        "string",
				"description": "Standard error output from the command",
			},
			"returnCode": map[string]interface{}{
				"type":        "integer",
				"description": "Exit code of the command (0 for success)",
			},
		},
		"required": []string{"stdout", "stderr", "returnCode"},
	}

	toolDef := map[string]interface{}{
		"name":        g.config.ToolName,
		"description": g.config.Description,
		"inputSchema": inputSchema,
		"returnType":  returnType,
	}

	return toolDef
}

// Generate creates the MCP server module
func (g *Generator) Generate() error {
	// Create output directory
	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Extract flag definitions from binary if not already set
	if len(g.flags) == 0 {
		if err := g.extractFlags(); err != nil {
			return fmt.Errorf("failed to extract flags: %w", err)
		}
	}

	// Create go.mod
	if err := g.createGoMod(); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	// Generate MCP server code
	if err := g.generateServer(); err != nil {
		return fmt.Errorf("failed to generate server: %w", err)
	}

	// Get the binary as a tool
	if err := g.installTool(); err != nil {
		return fmt.Errorf("failed to install tool: %w", err)
	}

	return nil
}

// extractFlags analyzes the binary to find flag definitions
func (g *Generator) extractFlags() error {
	// For now, we'll use a simple approach
	// In a real implementation, we would analyze the binary's source
	// or use reflection to extract flag definitions
	
	// This is a placeholder that would be replaced with actual analysis
	g.flags = []FlagDef{
		{Name: "help", Type: "bool", Default: "false", Description: "Show help"},
	}
	
	return nil
}

// createGoMod creates the go.mod file
func (g *Generator) createGoMod() error {
	modPath := filepath.Join(g.config.OutputDir, "go.mod")
	content := fmt.Sprintf(`module %s

go 1.22

require github.com/tmc/mcp latest
`, g.config.ModuleName)

	return os.WriteFile(modPath, []byte(content), 0644)
}

// generateServer generates the MCP server code
func (g *Generator) generateServer() error {
	serverPath := filepath.Join(g.config.OutputDir, "main.go")

	content, err := g.generateServerContent()
	if err != nil {
		return fmt.Errorf("failed to generate server content: %w", err)
	}

	return os.WriteFile(serverPath, []byte(content), 0644)
}

// installTool installs the binary as a tool
func (g *Generator) installTool() error {
	// Copy the binary to the tools directory
	toolsDir := filepath.Join(g.config.OutputDir, "bin")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools directory: %w", err)
	}

	// Copy binary
	destPath := filepath.Join(toolsDir, filepath.Base(g.config.BinaryPath))
	src, err := os.Open(g.config.BinaryPath)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination binary: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	return nil
}

// GenerateTxtar generates the MCP server as a txtar format string
func (g *Generator) GenerateTxtar() (string, error) {
	var buf bytes.Buffer

	// Generate the content for each file

	// Extract flags if not already set
	if len(g.flags) == 0 {
		if err := g.extractFlags(); err != nil {
			return "", fmt.Errorf("failed to extract flags: %w", err)
		}
	}

	// Generate go.mod content
	goModContent := fmt.Sprintf(`module %s

go 1.22

require github.com/tmc/mcp latest
`, g.config.ModuleName)

	// Generate main.go content
	mainContent, err := g.generateServerContent()
	if err != nil {
		return "", fmt.Errorf("failed to generate server content: %w", err)
	}

	// Create txtar format
	fmt.Fprintf(&buf, "# Generated MCP server for %s\n\n", g.config.ToolName)
	fmt.Fprintf(&buf, "-- go.mod --\n%s", goModContent)
	fmt.Fprintf(&buf, "-- main.go --\n%s", mainContent)

	// Add binary info
	fmt.Fprintf(&buf, "-- README.md --\n")
	fmt.Fprintf(&buf, "# %s MCP Server\n\n", g.config.ToolName)
	fmt.Fprintf(&buf, "Generated MCP server wrapper for `%s`.\n\n", g.config.BinaryPath)
	fmt.Fprintf(&buf, "## Usage\n\n")
	fmt.Fprintf(&buf, "1. Place the original binary at: `bin/%s`\n", filepath.Base(g.config.BinaryPath))
	fmt.Fprintf(&buf, "2. Run: `go run .`\n\n")
	fmt.Fprintf(&buf, "## Configuration\n\n")
	fmt.Fprintf(&buf, "- Tool name: %s\n", g.config.ToolName)
	fmt.Fprintf(&buf, "- Description: %s\n", g.config.Description)

	return buf.String(), nil
}

// generateServerContent generates just the server code as a string
func (g *Generator) generateServerContent() (string, error) {
	tmpl := `package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/tmc/mcp"
)

type {{.ServerName}}Tool struct {
	binaryPath string
}

func New{{.ServerName}}Tool(binaryPath string) *{{.ServerName}}Tool {
	return &{{.ServerName}}Tool{
		binaryPath: binaryPath,
	}
}

func (t *{{.ServerName}}Tool) Name() string {
	return "{{.ToolName}}"
}

func (t *{{.ServerName}}Tool) Description() string {
	return "{{.Description}}"
}

func (t *{{.ServerName}}Tool) InputSchema() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			{{range .Flags}}
			"{{.Name}}": map[string]any{
				"type": "{{.Type}}",
				"description": "{{.Description}}",
				{{if ne .Default ""}}
				"default": {{.Default}},
				{{end}}
			},
			{{end}}
		},
		"required": []string{
			{{range .Flags}}{{if .Required}}"{{.Name}}",{{end}}{{end}}
		},
	}
}

func (t *{{.ServerName}}Tool) Execute(ctx context.Context, params any) (any, error) {
	args, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	// Build command arguments
	cmdArgs := []string{}
	{{range .Flags}}
	if val, ok := args["{{.Name}}"]; ok {
		{{if eq .Type "boolean"}}
		if boolVal, ok := val.(bool); ok && boolVal {
			cmdArgs = append(cmdArgs, "-{{.Name}}")
		}
		{{else}}
		cmdArgs = append(cmdArgs, "-{{.Name}}="+fmt.Sprintf("%v", val))
		{{end}}
	}
	{{end}}

	// Execute the command
	cmd := exec.CommandContext(ctx, t.binaryPath, cmdArgs...)

	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Handle stdin if provided
	if stdinData, ok := args["stdin"]; ok && stdinData != nil {
		if stdinStr, ok := stdinData.(string); ok && stdinStr != "" {
			cmd.Stdin = strings.NewReader(stdinStr)
		}
	}

	err := cmd.Run()

	// Build result with stdout, stderr, and return code
	result := map[string]any{
		"stdout": stdout.String(),
		"stderr": stderr.String(),
		"returnCode": 0,
	}

	if err != nil {
		// Try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			result["returnCode"] = exitError.ExitCode()
		} else {
			result["returnCode"] = -1 // Unknown error
		}
	}

	return result, nil
}

func main() {
	binaryPath := flag.String("binary", "{{.BinaryPath}}", "Path to the wrapped binary")
	flag.Parse()

	// Create server
	server := mcp.NewServer(
		mcp.WithName("{{.ToolName}}-mcp"),
		mcp.WithVersion("1.0.0"),
	)

	// Add tool
	tool := New{{.ServerName}}Tool(*binaryPath)
	server.AddTool(tool)

	// Start server
	if err := server.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
`

	t, err := template.New("server").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	data := struct {
		ServerName  string
		ToolName    string
		Description string
		BinaryPath  string
		Flags       []FlagDef
	}{
		ServerName:  g.config.ServerName,
		ToolName:    g.config.ToolName,
		Description: g.config.Description,
		BinaryPath:  g.config.BinaryPath,
		Flags:       g.flags,
	}

	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}