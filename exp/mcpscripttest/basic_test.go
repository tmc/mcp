package mcpscripttest

import (
	"testing"
)

func TestBasicFunctionality(t *testing.T) {
	// Test that we can create default options
	opts := DefaultOptions()
	if opts == nil {
		t.Fatal("DefaultOptions returned nil")
	}

	// Test that we can create an engine
	engine := NewEngine(opts)
	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}

	// Test that the engine has basic commands
	// Note: Default commands from scripttest may vary, let's check for our specific commands
	// when IncludeDefaultMCPCommands is enabled
	opts.IncludeDefaultMCPCommands = true
	engine = NewEngine(opts)

	if _, ok := engine.Cmds["mcp-server-start"]; !ok {
		t.Error("Engine missing mcp-server-start command")
	}

	// Test that the engine has basic conditions
	if _, ok := engine.Conds["stdio"]; !ok {
		t.Error("Engine missing stdio condition")
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
	SetupCoverageEnvironment(t)
	t.Log("Coverage environment setup completed")

	// Test that the function runs without error
	// The actual coverage functionality depends on GOCOVERDIR being set
	t.Log("Coverage environment setup test passed")
}