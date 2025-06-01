package fuzzing_test

import (
	"errors"
	"strings"
	"testing"
	
	"github.com/tmc/mcp/exp/mcpscripttest/fuzzing"
)

// TestCoverageVisualization demonstrates coverage-guided fuzzing with visualization
func TestCoverageVisualization(t *testing.T) {
	// Skip unless explicitly requested
	if testing.Short() {
		t.Skip("Skipping visualization test in short mode")
	}
	
	// Create a simple "server" that accepts certain commands
	acceptedCommands := map[string]bool{
		"exec echo hello":   true,
		"exec true":         true,
		"stdout hello":      true,
		"stderr error":      true,
		"mcp-send {":        true,
		"mcp-recv response": true,
	}
	
	// Create visualizer with custom options
	vizOpts := fuzzing.DefaultVisualizerOptions()
	vizOpts.Enabled = true
	vizOpts.ShowStats = true
	vizOpts.ShowScript = true
	vizOpts.ShowRejected = true // Show both accepted and rejected scripts
	vizOpts.MaxScriptLines = 10
	viz := fuzzing.NewVisualizer(vizOpts)
	
	// Create run options with visualizer
	opts := fuzzing.DefaultRunOptions()
	opts.Iterations = 50
	opts.Verbose = false // Let visualizer handle output
	opts.Visualizer = viz
	
	// Track which commands were tested
	testedCommands := make(map[string]int)
	
	// Run fuzzing with visualization
	err := fuzzing.Run(func(script string) error {
		// Parse the script and validate commands
		lines := strings.Split(script, "\n")
		validCommands := 0
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			
			// Track all commands
			testedCommands[line]++

			// Check if this is a recognized command
			for cmd := range acceptedCommands {
				if strings.HasPrefix(line, cmd) {
					validCommands++
					break
				}
			}

			// Count any exec command as valid for fuzzing
			if strings.HasPrefix(line, "exec ") {
				validCommands++
			}
		}
		
		// Accept scripts with at least 2 valid commands
		if validCommands >= 2 {
			return nil
		}
		
		return errors.New("not enough valid commands")
	}, opts)
	
	if err != nil {
		t.Fatalf("Fuzzing failed: %v", err)
	}
	
	// Report which commands were tested
	t.Logf("Commands tested during fuzzing:")
	for cmd, count := range testedCommands {
		t.Logf("  %s: %d times", cmd, count)
	}
}