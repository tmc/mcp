package mcpscripttest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ToolsOptions configures how MCP tools are installed and managed
type ToolsOptions struct {
	// InstallCoverage determines if tools should be installed with coverage instrumentation
	// If not explicitly set, will auto-detect based on GOCOVERDIR environment variable
	InstallCoverage bool

	// AutoDetectCoverage enables automatic detection of coverage based on GOCOVERDIR
	// Defaults to true
	AutoDetectCoverage bool

	// ToolsDir is the directory where tools will be installed
	// If empty, a temporary directory will be created
	ToolsDir string

	// Tools is the list of tools to install
	// If empty, default MCP tools will be installed
	Tools []string

	// VerboseOutput enables detailed logging about tool operations
	VerboseOutput bool
}

// DefaultToolsOptions returns default options for tool installation
func DefaultToolsOptions() *ToolsOptions {
	return &ToolsOptions{
		InstallCoverage: false,
		AutoDetectCoverage: true,
		ToolsDir:        "",
		Tools:           []string{"mcp-replay", "mcp-spy", "mcp-start", "mcp-test", "mcp-verify", "mcp-send", "mcp-recv", "mcpdiff", "mcp-probe"},
		VerboseOutput:   false,
	}
}

// InstallMCPTools installs MCP tools with or without coverage instrumentation
// It returns a cleanup function that should be deferred to restore the original PATH
func InstallMCPTools(t *testing.T, opts *ToolsOptions) func() {
	t.Helper()

	if opts == nil {
		opts = DefaultToolsOptions()
	}

	// Determine if coverage should be enabled
	coverageEnabled := opts.InstallCoverage
	if opts.AutoDetectCoverage && !opts.InstallCoverage {
		// Auto-detect coverage based on GOCOVERDIR
		if os.Getenv("GOCOVERDIR") != "" {
			coverageEnabled = true
			if opts.VerboseOutput {
				t.Logf("Auto-detected coverage enabled (GOCOVERDIR set)")
			}
		}
	}

	// Create a temporary directory for tools if not specified
	toolsDir := opts.ToolsDir
	if toolsDir == "" {
		var err error
		toolsDir, err = os.MkdirTemp("", "mcp-tools-*")
		if err != nil {
			t.Fatalf("Failed to create temporary directory for tools: %v", err)
		}
		if opts.VerboseOutput {
			t.Logf("Created temporary directory for tools: %s", toolsDir)
		}
	} else {
		// Ensure the directory exists
		if err := os.MkdirAll(toolsDir, 0755); err != nil {
			t.Fatalf("Failed to create tools directory: %v", err)
		}
	}

	// Save the original PATH to restore later
	originalPath := os.Getenv("PATH")

	// Prepend the tools directory to PATH
	newPath := fmt.Sprintf("%s%c%s", toolsDir, os.PathListSeparator, originalPath)
	os.Setenv("PATH", newPath)
	if opts.VerboseOutput {
		t.Logf("Updated PATH to include tools directory: %s", toolsDir)
	}

	// Prepare the go install command
	installCmd := []string{"go", "install"}
	if coverageEnabled {
		installCmd = append(installCmd, "-cover")
	}

	// Install each tool
	for _, tool := range opts.Tools {
		// Clean the tool name (in case it has path separators)
		toolName := filepath.Base(tool)

		// Build the full import path
		// Use relative path for local installation instead of @latest
		importPath := fmt.Sprintf("../../cmd/%s", toolName)

		// Execute the install command
		cmd := exec.Command(installCmd[0], append(installCmd[1:], importPath)...)

		// Set the binary output directory
		cmd.Env = append(os.Environ(), fmt.Sprintf("GOBIN=%s", toolsDir))

		if opts.VerboseOutput {
			t.Logf("Installing %s with coverage: %v", toolName, coverageEnabled)
		}

		// Run the command
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Failed to install %s: %v\nOutput: %s", toolName, err, output)
			// Continue to the next tool rather than failing everything
			continue
		}

		if opts.VerboseOutput && len(output) > 0 {
			t.Logf("Install output for %s: %s", toolName, output)
		}
	}

	// Return a cleanup function
	return func() {
		// Restore the original PATH
		os.Setenv("PATH", originalPath)

		// Clean up the temporary directory if we created one
		if opts.ToolsDir == "" {
			if err := os.RemoveAll(toolsDir); err != nil && opts.VerboseOutput {
				t.Logf("Warning: Failed to remove temporary tools directory: %v", err)
			}
		}
	}
}

// SetupMCPToolsPath sets up the PATH to include MCP tools without installing them
// This is useful when you've already installed the tools elsewhere and just need to include them in the PATH
func SetupMCPToolsPath(t *testing.T, toolsDir string) func() {
	t.Helper()

	// Ensure the directory exists
	if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		t.Fatalf("Tools directory does not exist: %s", toolsDir)
	}

	// Save the original PATH to restore later
	originalPath := os.Getenv("PATH")

	// Check if the directory is already in PATH
	if strings.Contains(originalPath, toolsDir) {
		return func() {} // No change needed
	}

	// Prepend the tools directory to PATH
	newPath := fmt.Sprintf("%s%c%s", toolsDir, os.PathListSeparator, originalPath)
	os.Setenv("PATH", newPath)

	// Return a cleanup function
	return func() {
		// Restore the original PATH
		os.Setenv("PATH", originalPath)
	}
}
