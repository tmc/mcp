package fuzzing

import (
	"strings"
	"testing"
)

// TestSmartGeneratorSimple tests the basic smart generator functionality
func TestSmartGeneratorSimple(t *testing.T) {
	// Create a simple config with just one test binary
	config := SmartGeneratorConfig{
		GeneratorConfig: GeneratorConfig{
			DisabledCommands: map[string]bool{
				"mcp-send":  true, // Disable MCP commands to focus on exec
				"mcp-recv":  true,
				"mcp-trace": true,
				"mcp-serve": true,
			},
			MinScriptLength: 3,
			MaxScriptLength: 5,
		},
		EnableIntrospection: true,
		BinaryPaths:         []string{"/bin/echo"}, // Use a simple binary
	}

	// Create smart generator
	sg := NewSmartGenerator(42, config)

	// Debug: check binary cache
	t.Logf("Binary cache has %d entries", len(sg.binaryCache))
	for path, info := range sg.binaryCache {
		t.Logf("  %s: SupportsCooperative=%v", path, info.SupportsCooperativeFuzzing)
	}

	// Debug: check commands
	t.Logf("Commands (%d):", len(sg.commands))
	for i, cmd := range sg.commands {
		t.Logf("  [%d] %s: weight=%.2f", i, cmd.Name, cmd.Weight)
	}

	// Generate a few scripts
	for i := 0; i < 3; i++ {
		script := sg.Generate()
		t.Logf("\nScript %d:\n%s\n", i, script)

		// Check if exec commands use the binary
		lines := strings.Split(script, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "exec ") {
				t.Logf("Found exec command: %s", line)
				if strings.Contains(line, "/bin/echo") {
					t.Logf("✓ Uses cached binary!")
				}
			}
		}
	}
}
