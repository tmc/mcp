package fuzzing_test

import (
	"strings"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest/fuzzing"
)

// TestSpecializedGenerators demonstrates different specialized generators
func TestSpecializedGenerators(t *testing.T) {
	tests := []struct {
		name      string
		generator interface{ Generate() string }
		validate  func(t *testing.T, script string)
	}{
		{
			name:      "MCP Trace Generator",
			generator: fuzzing.NewMCPTraceGenerator(42),
			validate: func(t *testing.T, script string) {
				// Should have no exec commands
				if strings.Contains(script, "exec ") {
					t.Error("MCP trace generator should not include exec commands")
				}

				// Should have MCP commands
				if !strings.Contains(script, "mcp-") {
					t.Error("MCP trace generator should include MCP commands")
				}

				// Count MCP trace commands
				traceCount := strings.Count(script, "mcp-trace")
				t.Logf("MCP trace commands: %d", traceCount)
			},
		},
		{
			name:      "Safe File Operations",
			generator: fuzzing.NewSafeFileOperationsGenerator(123),
			validate: func(t *testing.T, script string) {
				// Should have no exec or rm commands
				if strings.Contains(script, "exec ") {
					t.Error("Safe file generator should not include exec commands")
				}
				if strings.Contains(script, "rm ") {
					t.Error("Safe file generator should not include rm commands")
				}

				// Should have file operations
				hasFileOps := strings.Contains(script, "cat ") ||
					strings.Contains(script, "cp ") ||
					strings.Contains(script, "mv ") ||
					strings.Contains(script, "mkdir ")

				if !hasFileOps {
					t.Error("Safe file generator should include file operations")
				}
			},
		},
		{
			name: "Custom Configuration",
			generator: fuzzing.NewSpecializedGenerator(999, fuzzing.GeneratorConfig{
				DisabledCommands: map[string]bool{
					"exec":  true,
					"sleep": true,
				},
				CommandWeights: map[string]float64{
					"stdin":  5.0,
					"stdout": 5.0,
				},
				AllowDirectives: true,
				MinScriptLength: 10,
				MaxScriptLength: 15,
			}),
			validate: func(t *testing.T, script string) {
				// Should have no exec or sleep
				if strings.Contains(script, "exec ") {
					t.Error("Custom generator should not include disabled exec")
				}
				if strings.Contains(script, "sleep ") {
					t.Error("Custom generator should not include disabled sleep")
				}

				// Should have high frequency of stdin/stdout
				stdinCount := strings.Count(script, "stdin ")
				stdoutCount := strings.Count(script, "stdout ")
				t.Logf("stdin: %d, stdout: %d", stdinCount, stdoutCount)

				// Should have directives
				hasDirectives := strings.Contains(script, "!") ||
					strings.Contains(script, "?") ||
					strings.Contains(script, "[")

				if hasDirectives {
					t.Log("Custom generator includes directives")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := tt.generator.Generate()
			t.Logf("Generated script:\n%s\n", script)
			tt.validate(t, script)

			// Analyze command distribution
			lines := strings.Split(script, "\n")
			commands := make(map[string]int)

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				parts := strings.Fields(line)
				if len(parts) > 0 {
					commands[parts[0]]++
				}
			}

			t.Log("Command distribution:")
			for cmd, count := range commands {
				t.Logf("  %s: %d", cmd, count)
			}
		})
	}
}

// TestGeneratorComparison compares original vs specialized generators
func TestGeneratorComparison(t *testing.T) {
	seed := int64(42)

	// Original generator
	original := fuzzing.NewFuzzGenerator(seed)
	originalScript := original.Generate()

	// Specialized generator with similar config
	specialized := fuzzing.NewSpecializedGenerator(seed, fuzzing.GeneratorConfig{
		AllowDirectives: true,
		MinScriptLength: 5,
		MaxScriptLength: 15,
	})
	specializedScript := specialized.Generate()

	t.Log("Original Generator Output:")
	t.Log(originalScript)
	t.Log("\nSpecialized Generator Output:")
	t.Log(specializedScript)

	// Analyze differences
	originalCommands := extractCommands(originalScript)
	specializedCommands := extractCommands(specializedScript)

	t.Log("\nCommand comparison:")
	t.Logf("Original commands: %v", originalCommands)
	t.Logf("Specialized commands: %v", specializedCommands)
}

func extractCommands(script string) map[string]int {
	commands := make(map[string]int)
	lines := strings.Split(script, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) > 0 {
			commands[parts[0]]++
		}
	}

	return commands
}
