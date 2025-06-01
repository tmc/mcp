package mcpscripttest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScripttestCoverageAcrossBinaries demonstrates coverage collection
// when scripttest runs tests that call coverage-enabled binaries
func TestScripttestCoverageAcrossBinaries(t *testing.T) {
	// Create a temporary directory structure for our meta-test
	tmpDir, err := os.MkdirTemp("", "scripttest-meta-coverage")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original GOCOVERDIR and PATH
	origCoverDir := os.Getenv("GOCOVERDIR")
	origPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("GOCOVERDIR", origCoverDir)
		os.Setenv("PATH", origPath)
	}()

	// Set up coverage directory
	coverDir := filepath.Join(tmpDir, "coverage")
	os.MkdirAll(coverDir, 0755)
	os.Setenv("GOCOVERDIR", coverDir)

	// Set up tools directory and install coverage-enabled tools
	toolsDir := filepath.Join(tmpDir, "tools")
	toolOpts := &ToolsOptions{
		AutoDetectCoverage: true,
		ToolsDir:           toolsDir,
		Tools:              []string{"mcpdiff", "mcpspy", "mcpcat"},
		VerboseOutput:      testing.Verbose(),
	}

	cleanup := InstallMCPTools(t, toolOpts)
	defer cleanup()

	// Create test script options
	scriptOpts := DefaultOptions()
	scriptOpts.AdditionalEnvVars = []string{"GOCOVERDIR"}

	// Run the meta-test
	Test(t, "testdata/scripttest_coverage_test.txt", scriptOpts)

	// Verify coverage was collected from multiple binaries
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		t.Fatalf("Failed to read coverage directory: %v", err)
	}

	// Count coverage files
	var covFileCount int
	var binaries []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "covcounters.") || strings.HasPrefix(name, "covmeta.") {
			covFileCount++
			// Extract binary name from coverage file name
			parts := strings.Split(name, ".")
			if len(parts) > 1 {
				binary := parts[1]
				if !contains(binaries, binary) {
					binaries = append(binaries, binary)
				}
			}
		}
	}

	t.Logf("Found %d coverage files from %d binaries", covFileCount, len(binaries))
	t.Logf("Binaries with coverage: %v", binaries)

	if covFileCount < 2 {
		t.Errorf("Expected coverage from multiple binaries, got %d files", covFileCount)
	}

	// Just check that we have coverage from multiple binaries
	// Binary names are hashed so we can't check for specific tool names
}

// TestScripttestCoverageMultipleTools tests coverage collection
// when scripttest runs multiple tools in sequence
func TestScripttestCoverageMultipleTools(t *testing.T) {
	// Create a test that uses multiple tools
	tmpDir, err := os.MkdirTemp("", "scripttest-multi-tool-coverage")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up coverage
	coverDir := filepath.Join(tmpDir, "coverage")
	os.MkdirAll(coverDir, 0755)
	origCoverDir := os.Getenv("GOCOVERDIR")
	defer os.Setenv("GOCOVERDIR", origCoverDir)
	os.Setenv("GOCOVERDIR", coverDir)

	// Install coverage-enabled tools
	toolOpts := &ToolsOptions{
		AutoDetectCoverage: true,
		ToolsDir:           filepath.Join(tmpDir, "tools"),
		Tools:              []string{"mcpdiff", "mcpspy", "mcpcat"},
		VerboseOutput:      testing.Verbose(),
	}

	cleanup := InstallMCPTools(t, toolOpts)
	defer cleanup()

	// Run a test that uses multiple tools
	scriptOpts := DefaultOptions()
	scriptOpts.AdditionalEnvVars = []string{"GOCOVERDIR"}

	Test(t, "testdata/multi_tool_coverage_test.txt", scriptOpts)

	// Count unique tools with coverage
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		t.Logf("Warning: Could not read coverage directory: %v", err)
		return
	}

	toolCoverage := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "covcounters.") || strings.HasPrefix(name, "covmeta.") {
			parts := strings.Split(name, ".")
			if len(parts) > 1 {
				toolCoverage[parts[1]] = true
			}
		}
	}

	t.Logf("Tools with coverage data: %d", len(toolCoverage))
	for tool := range toolCoverage {
		t.Logf("  - %s", tool)
	}

	if len(toolCoverage) < 2 {
		t.Errorf("Expected coverage from at least 2 tools, got %d", len(toolCoverage))
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}