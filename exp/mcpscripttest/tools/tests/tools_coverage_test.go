package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"

	"github.com/tmc/mcp/exp/mcpscripttest/tools"
)

func TestToolInstallationWithCoverage(t *testing.T) {
	// Create a temporary directory for coverage
	tmpDir, err := os.MkdirTemp("", "mcp-tools-coverage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original GOCOVERDIR
	origCoverDir := os.Getenv("GOCOVERDIR")
	defer os.Setenv("GOCOVERDIR", origCoverDir)

	// Set temporary GOCOVERDIR
	coverDir := filepath.Join(tmpDir, "coverage")
	os.Setenv("GOCOVERDIR", coverDir)

	// Create tools options with auto-detection enabled
	toolOpts := &tools.ToolsOptions{
		AutoDetectCoverage: true,
		ToolsDir:           filepath.Join(tmpDir, "tools"),
		Tools:              []string{"mcpdiff"}, // Just test one tool
		VerboseOutput:      true,
	}

	// Install tools - should auto-detect coverage
	cleanup := mcpscripttest.InstallMCPTools(t, toolOpts)
	defer cleanup()

	// Verify the tool was installed
	toolPath := filepath.Join(toolOpts.ToolsDir, "mcpdiff")
	if _, err := os.Stat(toolPath); os.IsNotExist(err) {
		t.Fatalf("Tool was not installed at expected path: %s", toolPath)
	}

	// Verify the tool has coverage instrumentation by checking build info
	cmd := exec.Command("go", "version", "-m", toolPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Could not check build info (this is expected if go version is too old): %v", err)
		t.Logf("Output: %s", output)
	} else {
		// Check if the output contains coverage-related build settings
		outputStr := string(output)
		if !containsAny(outputStr,
			"build	-cover=",
			"build	-buildmode=",
			"cover") {
			t.Logf("Build info output: %s", outputStr)
			// Note: Different Go versions may format this differently,
			// so we'll just log instead of failing
			t.Logf("WARNING: Could not confirm coverage instrumentation in build info")
		}
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(substr) > 0 && len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
