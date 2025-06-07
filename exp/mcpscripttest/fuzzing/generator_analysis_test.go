package fuzzing_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest/fuzzing"
)

// TestGeneratorOutput examines what kinds of scripts the fuzzer generates
func TestGeneratorOutput(t *testing.T) {
	// Generate several scripts and analyze them
	seeds := []int64{42, 123, 999, 1337, 8888}

	for _, seed := range seeds {
		t.Run(fmt.Sprintf("Seed_%d", seed), func(t *testing.T) {
			generator := fuzzing.NewFuzzGenerator(seed)
			script := generator.Generate()

			t.Logf("Script generated with seed %d:\n%s\n", seed, script)

			// Analyze the script
			lines := strings.Split(script, "\n")
			var execCount, mcpCount, fileOps, dirOps int

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				switch {
				case strings.HasPrefix(line, "exec "):
					execCount++
				case strings.HasPrefix(line, "mcp-"):
					mcpCount++
				case strings.HasPrefix(line, "cat ") ||
					strings.HasPrefix(line, "cp ") ||
					strings.HasPrefix(line, "mv ") ||
					strings.HasPrefix(line, "rm "):
					fileOps++
				case strings.HasPrefix(line, "mkdir ") ||
					strings.HasPrefix(line, "cd "):
					dirOps++
				}
			}

			t.Logf("Analysis:")
			t.Logf("  - exec commands: %d", execCount)
			t.Logf("  - MCP commands: %d", mcpCount)
			t.Logf("  - File operations: %d", fileOps)
			t.Logf("  - Directory operations: %d", dirOps)
			t.Logf("  - Total lines: %d", len(lines))
		})
	}
}

// TestCurrentCommands explores what commands are available
func TestCurrentCommands(t *testing.T) {
	// We'll analyze generated scripts

	// We can analyze what the generator knows about by looking at its output
	t.Log("Analyzing generated scripts to understand available commands...")

	// Generate multiple scripts to see patterns
	commands := make(map[string]int)
	for seed := int64(1); seed <= 20; seed++ {
		generator := fuzzing.NewFuzzGenerator(seed)
		script := generator.Generate()

		lines := strings.Split(script, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Extract command name
			parts := strings.Fields(line)
			if len(parts) > 0 {
				commands[parts[0]]++
			}
		}
	}

	t.Log("Commands used in generated scripts:")
	for cmd, count := range commands {
		t.Logf("  - %s: used %d times", cmd, count)
	}
}
