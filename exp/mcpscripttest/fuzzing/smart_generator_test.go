package fuzzing_test

import (
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest/fuzzing"
)

// TestSmartGenerator demonstrates binary introspection
func TestSmartGenerator(t *testing.T) {
	config := fuzzing.SmartGeneratorConfig{
		GeneratorConfig: fuzzing.GeneratorConfig{
			MinScriptLength: 5,
			MaxScriptLength: 10,
		},
		EnableIntrospection: true,
		CommonTestBinaries:  true,
		ValidateCommands:    false, // Skip validation for testing
	}

	generator := fuzzing.NewSmartGenerator(42, config)
	script := generator.Generate()

	t.Logf("Smart generated script:\n%s\n", script)

	// Analyze exec commands
	lines := strings.Split(script, "\n")
	execCommands := []string{}

	for _, line := range lines {
		if strings.HasPrefix(line, "exec ") {
			execCommands = append(execCommands, strings.TrimPrefix(line, "exec "))
		}
	}

	t.Logf("Generated exec commands: %v", execCommands)
}

// TestBinaryIntrospection tests the introspection capabilities
func TestBinaryIntrospection(t *testing.T) {
	introspector := fuzzing.NewBinaryIntrospector()

	// Test with common binaries
	binaries := []string{"echo", "cat", "ls"}

	for _, binary := range binaries {
		info, err := introspector.IntrospectBinary(binary)
		if err != nil {
			t.Logf("Failed to introspect %s: %v", binary, err)
			continue
		}

		t.Logf("Binary: %s", binary)
		t.Logf("  Supports help: %v", info.SupportsHelp)
		t.Logf("  Accepts stdin: %v", info.AcceptsStdin)
		t.Logf("  Flags found: %d", len(info.Flags))

		for _, flag := range info.Flags {
			t.Logf("    %s (%s): %s", flag.Name, flag.Type, flag.Description)
		}
	}
}

// TestSmartGeneratorWithValidation demonstrates validated generation
func TestSmartGeneratorWithValidation(t *testing.T) {
	config := fuzzing.SmartGeneratorConfig{
		GeneratorConfig: fuzzing.GeneratorConfig{
			CommandWeights: map[string]float64{
				"exec": 3.0, // Higher weight for exec commands
			},
		},
		EnableIntrospection:   true,
		CommonTestBinaries:    true,
		ValidateCommands:      true,
		MaxValidationAttempts: 3,
	}

	generator := fuzzing.NewSmartGeneratorWithEngine(123, config)
	script := generator.GenerateWithValidation()

	t.Logf("Validated script:\n%s\n", script)
}

// TestMCPSmartGenerator tests the MCP-focused smart generator
func TestMCPSmartGenerator(t *testing.T) {
	generator := fuzzing.NewMCPSmartGenerator(999)
	script := generator.Generate()

	t.Logf("MCP smart generated script:\n%s\n", script)

	// Count command types
	commandCounts := make(map[string]int)
	lines := strings.Split(script, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) > 0 {
			commandCounts[parts[0]]++
		}
	}

	t.Log("Command distribution:")
	for cmd, count := range commandCounts {
		t.Logf("  %s: %d", cmd, count)
	}
}

// Example fuzzing with smart generator
func FuzzWithSmartGenerator(f *testing.F) {
	f.Add(int64(42))

	f.Fuzz(func(t *testing.T, seed int64) {
		config := fuzzing.SmartGeneratorConfig{
			GeneratorConfig: fuzzing.GeneratorConfig{
				DisabledCommands: map[string]bool{
					"rm": true, // Still disable dangerous commands
				},
			},
			EnableIntrospection: true,
			CommonTestBinaries:  true,
			ValidateCommands:    true,
		}

		generator := fuzzing.NewSmartGeneratorWithEngine(seed, config)
		script := generator.GenerateWithValidation()

		// Run the generated script
		// In real use, this would go through mcpscripttest
		t.Logf("Generated validated script with %d lines", len(strings.Split(script, "\n")))
	})
}
