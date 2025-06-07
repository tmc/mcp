package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"

	"github.com/tmc/mcp/exp/mcpscripttest/tools"
)

// TestExplicitCoverage demonstrates explicit control over coverage collection
func TestExplicitCoverage(t *testing.T) {
	// Create our main coverage directory
	mainCoverDir := t.TempDir()

	// Save original GOCOVERDIR
	origCoverDir := os.Getenv("GOCOVERDIR")
	defer os.Setenv("GOCOVERDIR", origCoverDir)

	// Set GOCOVERDIR for tool building
	os.Setenv("GOCOVERDIR", mainCoverDir)

	// Install tools with coverage
	toolsDir := filepath.Join(mainCoverDir, "tools")
	opts := &tools.ToolsOptions{
		AutoDetectCoverage: true,
		ToolsDir:           toolsDir,
		Tools:              []string{"mcpdiff"},
		VerboseOutput:      true,
	}
	cleanup := mcpscripttest.InstallMCPTools(t, opts)
	defer cleanup()

	// Create test files
	testDir := t.TempDir()
	file1 := filepath.Join(testDir, "file1.mcp")
	file2 := filepath.Join(testDir, "file2.mcp")

	os.WriteFile(file1, []byte(`mcp-send {"jsonrpc":"2.0","method":"test","id":1}`), 0644)
	os.WriteFile(file2, []byte(`mcp-send {"jsonrpc":"2.0","method":"test","id":1}`), 0644)

	// Run mcpdiff directly with explicit GOCOVERDIR
	cmd := exec.Command(filepath.Join(toolsDir, "mcpdiff"), file1, file2)
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOCOVERDIR=%s", mainCoverDir))

	output, err := cmd.CombinedOutput()
	t.Logf("mcpdiff output: %s", output)
	if err != nil {
		t.Logf("mcpdiff error (expected): %v", err)
	}

	// Check coverage files
	entries, err := os.ReadDir(mainCoverDir)
	if err != nil {
		t.Fatalf("Failed to read coverage directory: %v", err)
	}

	t.Logf("Contents of %s:", mainCoverDir)
	for _, entry := range entries {
		t.Logf("  %s", entry.Name())
	}

	// Count coverage files
	covFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if len(name) > 3 && (name[:3] == "cov") {
				covFiles++
			}
		}
	}

	t.Logf("Total coverage files: %d", covFiles)

	// Try to analyze coverage
	if covFiles > 0 {
		cmd = exec.Command("go", "tool", "covdata", "percent", "-i", mainCoverDir)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Logf("Coverage analysis error: %v", err)
		} else {
			t.Logf("Coverage analysis:\n%s", output)
		}
	}
}
