package cmd2mcpserver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/cmd2mcpserver"
	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestGenerateServer(t *testing.T) {
	// Create a temporary directory for output
	tmpDir := t.TempDir()
	
	config := &cmd2mcpserver.Config{
		BinaryPath:  "/bin/echo",
		OutputDir:   filepath.Join(tmpDir, "echo-server"),
		ModuleName:  "github.com/test/echo-server",
		ServerName:  "Echo",
		ToolName:    "echo",
		Description: "Echo command wrapper",
	}
	
	generator := cmd2mcpserver.NewGenerator(config)
	
	// Set some test flags
	flags := []cmd2mcpserver.FlagDef{
		{
			Name:        "text",
			Type:        "string",
			Default:     `"hello"`,
			Description: "Text to echo",
			Required:    true,
		},
		{
			Name:        "no-newline",
			Type:        "boolean",
			Default:     "false",
			Description: "Do not print newline",
			Required:    false,
		},
	}
	generator.SetFlags(flags)
	
	// Generate the server
	err := generator.Generate()
	if err != nil {
		t.Fatalf("Failed to generate server: %v", err)
	}
	
	// Check that files were created
	expectedFiles := []string{
		"go.mod",
		"main.go",
	}
	
	for _, file := range expectedFiles {
		path := filepath.Join(config.OutputDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", file)
		}
	}
	
	// Read and check the generated main.go
	mainPath := filepath.Join(config.OutputDir, "main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}
	
	// Basic content checks
	contentStr := string(content)
	t.Logf("Generated content:\n%s", contentStr)

	expectedStrings := []string{
		"type EchoTool struct",
		`"text"`,
		`"no-newline"`,
		"echo-mcp",
	}

	for _, expected := range expectedStrings {
		if !contains(contentStr, expected) {
			t.Errorf("Generated code does not contain expected string: %s", expected)
		}
	}
}

func TestFlagExtractor(t *testing.T) {
	// Create a test Go file with flag definitions
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	
	testCode := `package main

import "flag"

func main() {
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	port := flag.Int("port", 8080, "Server port")
	host := flag.String("host", "localhost", "Server host")
	timeout := flag.Float64("timeout", 30.0, "Request timeout in seconds")
	
	flag.Parse()
}
`
	
	err := os.WriteFile(testFile, []byte(testCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	extractor := cmd2mcpserver.NewFlagExtractor(tmpDir)
	flags, err := extractor.ExtractFlags()
	if err != nil {
		t.Fatalf("Failed to extract flags: %v", err)
	}
	
	// Check extracted flags
	if len(flags) < 3 {
		t.Errorf("Expected at least 3 flags, got %d", len(flags))
	}
	
	// Look for specific flags
	flagMap := make(map[string]cmd2mcpserver.FlagDef)
	for _, flag := range flags {
		flagMap[flag.Name] = flag
	}
	
	// Check verbose flag
	if verbose, ok := flagMap["verbose"]; ok {
		if verbose.Type != "boolean" {
			t.Errorf("Expected verbose type to be boolean, got %s", verbose.Type)
		}
	} else {
		t.Error("Flag 'verbose' not found")
	}
	
	// Check port flag
	if port, ok := flagMap["port"]; ok {
		if port.Type != "integer" {
			t.Errorf("Expected port type to be integer, got %s", port.Type)
		}
	} else {
		t.Error("Flag 'port' not found")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestGenerateTxtar(t *testing.T) {
	config := &cmd2mcpserver.Config{
		BinaryPath:  "/bin/echo",
		OutputDir:   "echo-server",
		ModuleName:  "github.com/test/echo-server",
		ServerName:  "Echo",
		ToolName:    "echo",
		Description: "Echo command wrapper",
	}

	generator := cmd2mcpserver.NewGenerator(config)

	// Set some test flags
	flags := []cmd2mcpserver.FlagDef{
		{
			Name:        "text",
			Type:        "string",
			Default:     `"hello"`,
			Description: "Text to echo",
			Required:    true,
		},
		{
			Name:        "no-newline",
			Type:        "boolean",
			Default:     "false",
			Description: "Do not print newline",
			Required:    false,
		},
	}
	generator.SetFlags(flags)

	// Generate txtar
	txtar, err := generator.GenerateTxtar()
	if err != nil {
		t.Fatalf("Failed to generate txtar: %v", err)
	}

	// Check that txtar contains expected sections
	expectedStrings := []string{
		"-- go.mod --",
		"-- main.go --",
		"-- README.md --",
		"module github.com/test/echo-server",
		"type EchoTool struct",
		`"text"`,
		`"no-newline"`,
		"echo-mcp",
	}

	for _, expected := range expectedStrings {
		if !contains(txtar, expected) {
			t.Errorf("Generated txtar does not contain expected string: %s", expected)
		}
	}

	// Print for manual inspection if needed
	t.Logf("Generated txtar:\n%s", txtar)
}

// TestCmd2mcpserverScripts runs script-based tests using mcpscripttest
func TestCmd2mcpserverScripts(t *testing.T) {
	// Skip if no test scripts exist
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		t.Skip("No testdata directory found")
	}

	// Run all script tests in testdata/ directory
	mcpscripttest.Test(t, "testdata/*.txt", nil)
}