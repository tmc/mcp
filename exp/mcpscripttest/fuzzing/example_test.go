package fuzzing_test

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
	"github.com/tmc/mcp/exp/mcpscripttest/fuzzing"
)

// FuzzEchoServer demonstrates basic fuzzing of an echo server
func FuzzEchoServer(f *testing.F) {
	serverCmd := []string{"go", "run", "../../examples/servers/mcp-echo-server"}
	
	// Use default options
	fuzzing.FuzzWithState(f, serverCmd, nil)
}

// FuzzTimeServerWithCoverage demonstrates coverage-guided fuzzing
func FuzzTimeServerWithCoverage(f *testing.F) {
	serverCmd := []string{"go", "run", "../../examples/servers/mcp-time-server"}
	
	opts := &mcpscripttest.MCPScripttestOptions{
		// Enable default MCP commands
		IncludeDefaultMCPCommands: true,
		// Enable debug mode for verbose output
		DebugMode: testing.Verbose(),
	}
	
	fuzzing.FuzzWithState(f, serverCmd, opts)
}

// Example of using Run() for direct fuzzing
func TestDirectFuzzing(t *testing.T) {
	opts := fuzzing.DefaultRunOptions()
	opts.Iterations = 100  // Fewer iterations for example
	opts.MinCoverage = 60.0
	opts.Verbose = true
	
	err := fuzzing.Run(func(script string) error {
		// This would be your actual test implementation
		t.Logf("Testing script with %d lines", len(script))
		return nil
	}, opts)
	
	if err != nil {
		t.Fatalf("Fuzzing failed: %v", err)
	}
}

// Example of custom fuzzing with specific patterns
func FuzzCustomPatterns(f *testing.F) {
	// Add specific seed patterns that you want to test
	f.Add(int64(42))  // Base seed
	f.Add(int64(123)) // Different pattern
	f.Add(int64(999)) // Edge case seed
	
	f.Fuzz(func(t *testing.T, seed int64) {
		// Create a generator with custom seed
		generator := fuzzing.NewFuzzGenerator(seed)
		script := generator.Generate()
		
		// Validate the generated script
		if len(script) == 0 {
			t.Skip("Empty script generated")
		}
		
		// You could run the script through your server here
		t.Logf("Generated script with seed %d: %d bytes", seed, len(script))
	})
}