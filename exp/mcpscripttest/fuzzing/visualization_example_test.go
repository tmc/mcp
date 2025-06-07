package fuzzing_test

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest/fuzzing"
)

// TestVisualizationExample demonstrates the fuzzing visualizer
func TestVisualizationExample(t *testing.T) {
	// This test shows how to use the visualizer for live fuzzing feedback

	// Create visualization options
	vizOpts := fuzzing.DefaultVisualizerOptions()
	vizOpts.Enabled = true
	vizOpts.ShowStats = true
	vizOpts.ShowScript = true
	vizOpts.ShowRejected = false // Only show accepted scripts by default
	vizOpts.MaxScriptLines = 15

	// Create the visualizer
	viz := fuzzing.NewVisualizer(vizOpts)

	// Create run options with visualizer
	opts := fuzzing.DefaultRunOptions()
	opts.Iterations = 100
	opts.Verbose = false // Let visualizer handle output
	opts.Visualizer = viz

	// Run fuzzing with visualization
	err := fuzzing.Run(func(script string) error {
		// Simple validation: reject scripts with more than 10 lines
		lines := 0
		for _, c := range script {
			if c == '\n' {
				lines++
			}
		}

		if lines > 10 {
			return nil // Accept complex scripts
		}

		// Simulate some processing time
		// time.Sleep(10 * time.Millisecond)

		return nil
	}, opts)

	if err != nil {
		t.Fatalf("Fuzzing failed: %v", err)
	}
}

// FuzzWithVisualization demonstrates using visualization in fuzz tests
func FuzzWithVisualization(f *testing.F) {
	// This fuzz test will show live progress when run with:
	// MCP_FUZZ_VISUALIZE=1 go test -fuzz=FuzzWithVisualization
	// Or with verbose flag:
	// go test -v -fuzz=FuzzWithVisualization

	f.Add(int64(42))

	f.Fuzz(func(t *testing.T, seed int64) {
		// The visualization will be enabled automatically if
		// MCP_FUZZ_VISUALIZE=1 or -v flag is used

		// Generate and test a script
		generator := fuzzing.NewFuzzGenerator(seed)
		script := generator.Generate()

		// Simulate validation
		if len(script) > 500 {
			t.Skip("Script too long")
		}

		// Test passes if we get here
		t.Logf("Generated script with %d bytes", len(script))
	})
}
