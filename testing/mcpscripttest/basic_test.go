package mcpscripttest

import (
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest/coverage"
)

func TestBasicFunctionality(t *testing.T) {
	// Test that we can create default options
	opts := DefaultOptions()
	if opts == nil {
		t.Fatal("DefaultOptions returned nil")
	}

	// Test that options have expected defaults
	if !opts.IncludeDefaultMCPCommands {
		t.Error("Expected IncludeDefaultMCPCommands to be true by default")
	}

	t.Log("Basic functionality test passed")
}

func TestCoverageOptionsFunction(t *testing.T) {
	// Test default coverage options
	opts := DefaultCoverageOptions()
	if opts == nil {
		t.Fatal("DefaultCoverageOptions returned nil")
	}

	if opts.Enabled {
		t.Error("Coverage should be disabled by default")
	}

	// Test setting coverage options
	opts.Enabled = true
	opts.VerboseOutput = true

	if !opts.Enabled {
		t.Error("Failed to enable coverage")
	}

	t.Log("Coverage options test passed")
}

func TestCoverageEnvironmentSetup(t *testing.T) {
	// Test coverage environment setup
	coverage.SetupCoverageEnvironment(t)
	t.Log("Coverage environment setup completed")

	// Test that the function runs without error
	// The actual coverage functionality depends on GOCOVERDIR being set
	t.Log("Coverage environment setup test passed")
}

// TestMinimalWorkingScripttest tests that scripttest works with our fixed minimal test
func TestMinimalWorkingScripttest(t *testing.T) {
	Test(t, "../../testdata/scripttest/minimal_working_test.txt")
}

// TestWorkingBasicScripttest tests that working_basic_test.txt works
func TestWorkingBasicScripttest(t *testing.T) {
	Test(t, "../../testdata/scripttest/working_basic_test.txt")
}
