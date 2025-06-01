package fuzzing

import (
	"testing"
	"strings"
)

// TestSmartGeneratorOverride tests that the exec command override works
func TestSmartGeneratorOverride(t *testing.T) {
	// Create a simple config
	config := SmartGeneratorConfig{
		GeneratorConfig: GeneratorConfig{
			DisabledCommands: map[string]bool{},
		},
		EnableIntrospection: true,
		BinaryPaths:        []string{"echo"},  // Use a known binary
		CommonTestBinaries: true,
	}
	
	// Create smart generator
	sg := NewSmartGenerator(42, config)
	
	// Check if exec was overridden
	foundExec := false
	for _, cmd := range sg.commands {
		if cmd.Name == "exec" {
			foundExec = true
			t.Logf("Found exec command with generator: %p", cmd.Generator)
			
			// Test the generator
			result := cmd.Generator(sg.SpecializedGenerator)
			t.Logf("Generated exec command: %s", result)
			
			// Should use the binary cache if available
			if len(sg.binaryCache) > 0 && !strings.Contains(result, "echo") {
				t.Error("Exec generator not using binary cache")
			}
			break
		}
	}
	
	if !foundExec {
		t.Error("Exec command not found in commands list")
	}
}

// TestSmartGeneratorIntrospection tests the introspection behavior
func TestSmartGeneratorIntrospection(t *testing.T) {
	// Create config with echo binary
	config := SmartGeneratorConfig{
		GeneratorConfig: GeneratorConfig{
			DisabledCommands: map[string]bool{},
		},
		EnableIntrospection: true,
		BinaryPaths:        []string{"echo"},
		CommonTestBinaries: true,
	}
	
	// Create smart generator
	sg := NewSmartGenerator(42, config)
	
	// Check binary cache
	t.Logf("Binary cache size: %d", len(sg.binaryCache))
	for path, info := range sg.binaryCache {
		t.Logf("Binary: %s, SupportsHelp: %v", path, info.SupportsHelp)
	}
	
	// Generate a script to see if it uses cached binaries
	script := sg.Generate()
	t.Logf("Generated script:\n%s", script)
	
	// Count exec commands
	execCount := strings.Count(script, "exec ")
	t.Logf("Number of exec commands: %d", execCount)
}