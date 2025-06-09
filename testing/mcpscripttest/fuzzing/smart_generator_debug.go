package fuzzing

import (
	"fmt"
	"strings"
)

// generateSmartExecCommandDebug generates exec commands using introspection with debug info
func generateSmartExecCommandDebug(sg *SmartGenerator) func(*SpecializedGenerator) string {
	return func(g *SpecializedGenerator) string {
		fmt.Printf("DEBUG: generateSmartExecCommand called\n")
		fmt.Printf("DEBUG: binary cache size: %d\n", len(sg.binaryCache))

		// If we have introspected binaries, use them
		if len(sg.binaryCache) > 0 {
			// Pick a random binary
			binaries := make([]string, 0, len(sg.binaryCache))
			for path := range sg.binaryCache {
				binaries = append(binaries, path)
			}

			binary := binaries[sg.rng.Intn(len(binaries))]
			info := sg.binaryCache[binary]

			fmt.Printf("DEBUG: selected binary: %s\n", binary)
			fmt.Printf("DEBUG: binary supports cooperative: %v\n", info.SupportsCooperativeFuzzing)

			// Generate a command based on introspection
			result := sg.generateCommandFromInfo(info)
			fmt.Printf("DEBUG: generated command: %s\n", result)
			return result
		}

		// Fallback to traditional generation
		fmt.Printf("DEBUG: falling back to traditional generation\n")
		return sg.generateTraditionalExecCommand()
	}
}

// generateCommandFromInfoDebug creates a command based on binary info with debug info
func generateCommandFromInfoDebug(sg *SmartGenerator, info *BinaryInfo) string {
	fmt.Printf("DEBUG: generateCommandFromInfo called for %s\n", info.Path)

	// First check if this binary supports cooperative fuzzing
	if sg.isCooperativeBinary(info.Path) {
		fmt.Printf("DEBUG: binary supports cooperative fuzzing\n")
		return sg.generateCooperativeCommand(info.Path)
	}

	// Start with the binary path
	parts := []string{"exec", info.Path}

	// Add flags based on what we know
	if len(info.Flags) > 0 && sg.rng.Float64() < 0.7 {
		// 70% chance to add flags
		numFlags := sg.rng.Intn(3) + 1 // 1-3 flags

		for i := 0; i < numFlags && i < len(info.Flags); i++ {
			flag := info.Flags[sg.rng.Intn(len(info.Flags))]

			// Use short or long form
			if flag.ShortName != "" && sg.rng.Float64() < 0.5 {
				parts = append(parts, flag.ShortName)
			} else if flag.Name != "" {
				parts = append(parts, flag.Name)
			}

			// Add value for non-bool flags
			if flag.Type != "bool" {
				parts = append(parts, sg.generateFlagValue(flag.Type))
			}
		}
	}

	// Add positional arguments for some commands
	if sg.shouldAddPositionalArgs(info.Path) {
		parts = append(parts, sg.generatePositionalArgs(info.Path)...)
	}

	result := strings.Join(parts, " ")
	fmt.Printf("DEBUG: final command: %s\n", result)
	return result
}
