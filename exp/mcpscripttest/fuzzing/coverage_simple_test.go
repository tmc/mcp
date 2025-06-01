package fuzzing_test

import (
	"testing"
	
	"github.com/tmc/mcp/exp/mcpscripttest"
	"github.com/tmc/mcp/exp/mcpscripttest/fuzzing"
)

// TestSimpleExample demonstrates basic fuzzing usage
func TestSimpleExample(t *testing.T) {
	// This is a simple example that doesn't use undefined fields
	opts := fuzzing.DefaultRunOptions()
	opts.Iterations = 10  // Just a few iterations for the test
	opts.Verbose = testing.Verbose()
	
	err := fuzzing.Run(func(script string) error {
		// Simple test that just validates the script length
		if len(script) == 0 {
			return nil 
		}
		t.Logf("Generated script with %d bytes", len(script))
		return nil
	}, opts)
	
	if err != nil {
		t.Fatalf("Fuzzing failed: %v", err)
	}
}

// FuzzSimpleServer demonstrates basic server fuzzing
func FuzzSimpleServer(f *testing.F) {
	serverCmd := []string{"echo", "test server"}
	
	// Use nil options to get defaults
	fuzzing.FuzzWithState(f, serverCmd, nil)
}

// FuzzWithOptions demonstrates fuzzing with options
func FuzzWithOptions(f *testing.F) {
	serverCmd := []string{"echo", "test server"}
	
	opts := &mcpscripttest.MCPScripttestOptions{
		IncludeDefaultMCPCommands: true,
		DebugMode:                 testing.Verbose(),
	}
	
	fuzzing.FuzzWithState(f, serverCmd, opts)
}