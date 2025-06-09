package fuzzing_test

import (
	"os"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
	"github.com/tmc/mcp/testing/mcpscripttest/fuzzing"
)

// FuzzMCPTraces demonstrates fuzzing with MCP-focused test generation
func FuzzMCPTraces(f *testing.F) {
	// Add seed corpus
	f.Add(int64(42))

	f.Fuzz(func(t *testing.T, seed int64) {
		// Use the MCP trace generator
		generator := fuzzing.NewMCPTraceGenerator(seed)
		script := generator.Generate()

		// Create temp file
		tmpfile := createTempScript(t, script)
		defer tmpfile.Close()

		// Run the test
		mcpscripttest.Test(t, tmpfile.Name())
	})
}

// FuzzSafeFileOperations demonstrates fuzzing without exec/rm commands
func FuzzSafeFileOperations(f *testing.F) {
	f.Add(int64(123))

	f.Fuzz(func(t *testing.T, seed int64) {
		// Use the safe file operations generator
		generator := fuzzing.NewSafeFileOperationsGenerator(seed)
		script := generator.Generate()

		tmpfile := createTempScript(t, script)
		defer tmpfile.Close()

		// This test will be safer as it doesn't execute arbitrary commands
		mcpscripttest.Test(t, tmpfile.Name())
	})
}

// FuzzCustomConfiguration demonstrates fuzzing with custom config
func FuzzCustomConfiguration(f *testing.F) {
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

		tmpfile := createTempScript(t, script)
		defer tmpfile.Close()

		mcpscripttest.Test(t, tmpfile.Name())
	})
}

// createTempScript is a helper to create a temporary script file
func createTempScript(t *testing.T, script string) *os.File {
	tmpfile, err := os.CreateTemp("", "fuzz-*.txt")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpfile.WriteString(script); err != nil {
		t.Fatal(err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpfile
}
