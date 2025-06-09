package fuzzing_test

import (
	"strings"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest/fuzzing"
)

// FuzzMCPTracesFixed demonstrates fuzzing with MCP-focused test generation
func FuzzMCPTracesFixed(f *testing.F) {
	// Add seed corpus
	f.Add(int64(42))

	f.Fuzz(func(t *testing.T, seed int64) {
		// Use the MCP trace generator
		generator := fuzzing.NewMCPTraceGenerator(seed)
		script := generator.Generate()

		// Just verify the script contains expected patterns
		if !strings.Contains(script, "mcp-") {
			t.Error("Script doesn't contain MCP commands")
		}

		// Log the script for debugging
		t.Logf("Generated MCP script with %d bytes", len(script))
	})
}

// FuzzSafeFileOperationsFixed demonstrates fuzzing without exec/rm commands
func FuzzSafeFileOperationsFixed(f *testing.F) {
	f.Add(int64(123))

	f.Fuzz(func(t *testing.T, seed int64) {
		// Use the safe file operations generator
		generator := fuzzing.NewSafeFileOperationsGenerator(seed)
		script := generator.Generate()

		// Verify that dangerous commands are not present
		if strings.Contains(script, "rm -rf") {
			t.Error("Script contains dangerous rm -rf")
		}

		if strings.Contains(script, " rm ") && !strings.Contains(script, "! exec") {
			t.Error("Script contains unsafe rm command")
		}

		t.Logf("Generated safe script with %d bytes", len(script))
	})
}

// FuzzCustomConfigurationFixed demonstrates fuzzing with custom config
func FuzzCustomConfigurationFixed(f *testing.F) {
	f.Add(int64(999))

	f.Fuzz(func(t *testing.T, seed int64) {
		// Create a custom configuration
		config := fuzzing.GeneratorConfig{
			DisabledCommands: map[string]bool{
				"exec":  true, // No arbitrary execution
				"rm":    true, // No deletions
				"sleep": true, // No delays
			},
			CommandWeights: map[string]float64{
				"mcp-send": 3.0, // Focus on MCP protocol
				"mcp-recv": 3.0,
				"stdin":    2.0,
				"stdout":   2.0,
			},
			AllowDirectives: false, // No platform-specific tests
			MinScriptLength: 5,
			MaxScriptLength: 10,
		}

		generator := fuzzing.NewSpecializedGenerator(seed, config)
		script := generator.Generate()

		// Verify that disabled commands are not present
		for cmd := range config.DisabledCommands {
			if strings.Contains(script, cmd+" ") {
				t.Errorf("Script contains disabled command: %s", cmd)
			}
		}

		// Verify script length constraints
		lines := strings.Split(strings.TrimSpace(script), "\n")
		if len(lines) < config.MinScriptLength || len(lines) > config.MaxScriptLength {
			t.Errorf("Script length %d not within bounds [%d, %d]",
				len(lines), config.MinScriptLength, config.MaxScriptLength)
		}

		t.Logf("Generated custom script with %d lines", len(lines))
	})
}
