package fuzzing

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SmartGenerator uses binary introspection to generate valid commands
type SmartGenerator struct {
	*SpecializedGenerator
	introspector *BinaryIntrospector
	binaryCache  map[string]*BinaryInfo
	cooperativeCache map[string]bool  // Track which binaries support cooperative fuzzing
}

// SmartGeneratorConfig extends GeneratorConfig with introspection options
type SmartGeneratorConfig struct {
	GeneratorConfig
	
	// EnableIntrospection turns on binary analysis
	EnableIntrospection bool
	
	// BinaryPaths lists specific binaries to analyze
	BinaryPaths []string
	
	// CommonTestBinaries includes common test tools
	CommonTestBinaries bool
	
	// ValidateCommands checks if generated commands are valid
	ValidateCommands bool
	
	// MaxValidationAttempts limits validation retries
	MaxValidationAttempts int
}

// NewSmartGenerator creates a generator that uses binary introspection
func NewSmartGenerator(seed int64, config SmartGeneratorConfig) *SmartGenerator {
	// Set defaults
	if config.MaxValidationAttempts == 0 {
		config.MaxValidationAttempts = 3
	}
	
	sg := &SmartGenerator{
		SpecializedGenerator: NewSpecializedGenerator(seed, config.GeneratorConfig),
		introspector:        NewBinaryIntrospector(),
		binaryCache:         make(map[string]*BinaryInfo),
		cooperativeCache:    make(map[string]bool),
	}
	
	// Pre-introspect configured binaries
	if config.EnableIntrospection {
		sg.introspectBinaries(config)
	}
	
	// Override exec command generator with smart version
	if !config.DisabledCommands["exec"] {
		sg.overrideExecGenerator()
	}
	
	return sg
}

// introspectBinaries analyzes configured binaries
func (sg *SmartGenerator) introspectBinaries(config SmartGeneratorConfig) {
	// Add specific binaries
	for _, path := range config.BinaryPaths {
		if info, err := sg.introspector.IntrospectBinary(path); err == nil {
			sg.binaryCache[path] = info
		}
	}
	
	// Add common test binaries
	if config.CommonTestBinaries {
		commonBinaries := []string{
			"echo", "cat", "grep", "sed", "awk",
			"true", "false", "test", "date", "pwd",
			"wc", "sort", "uniq", "head", "tail",
		}
		
		for _, binary := range commonBinaries {
			if info, err := sg.introspector.IntrospectBinary(binary); err == nil {
				sg.binaryCache[binary] = info
			}
		}
	}
}

// overrideExecGenerator replaces the exec command generator with a smart version
func (sg *SmartGenerator) overrideExecGenerator() {
	for i, cmd := range sg.commands {
		if cmd.Name == "exec" {
			sg.commands[i].Generator = sg.generateSmartExecCommand
			break
		}
	}
}

// generateSmartExecCommand generates exec commands using introspection
func (sg *SmartGenerator) generateSmartExecCommand(g *SpecializedGenerator) string {
	// If we have introspected binaries, use them
	if len(sg.binaryCache) > 0 {
		// Pick a random binary
		binaries := make([]string, 0, len(sg.binaryCache))
		for path := range sg.binaryCache {
			binaries = append(binaries, path)
		}
		
		binary := binaries[sg.rng.Intn(len(binaries))]
		info := sg.binaryCache[binary]
		
		// Generate a command based on introspection
		return sg.generateCommandFromInfo(info)
	}
	
	// Fallback to traditional generation
	return sg.generateTraditionalExecCommand()
}

// generateCommandFromInfo creates a command based on binary info
func (sg *SmartGenerator) generateCommandFromInfo(info *BinaryInfo) string {
	// First check if this binary supports cooperative fuzzing
	if sg.isCooperativeBinary(info.Path) {
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
	
	return strings.Join(parts, " ")
}

// generateFlagValue generates a value for a flag type
func (sg *SmartGenerator) generateFlagValue(flagType string) string {
	switch flagType {
	case "string":
		values := []string{"test", "value", "output.txt", "/tmp/test"}
		return values[sg.rng.Intn(len(values))]
	case "int":
		return fmt.Sprintf("%d", sg.rng.Intn(100))
	case "bool":
		return "" // Bool flags don't need values
	default:
		return "value"
	}
}

// shouldAddPositionalArgs determines if a command needs positional args
func (sg *SmartGenerator) shouldAddPositionalArgs(binary string) bool {
	base := filepath.Base(binary)
	
	// Commands that typically need arguments
	needsArgs := map[string]bool{
		"echo": true,
		"cat":  true,
		"grep": true,
		"sed":  true,
		"awk":  true,
		"wc":   true,
		"sort": true,
	}
	
	return needsArgs[base]
}

// generatePositionalArgs generates appropriate positional arguments
func (sg *SmartGenerator) generatePositionalArgs(binary string) []string {
	base := filepath.Base(binary)
	
	switch base {
	case "echo":
		messages := []string{"test", "hello world", "done", "success"}
		return []string{messages[sg.rng.Intn(len(messages))]}
		
	case "cat", "wc", "sort":
		files := []string{"test.txt", "data.log", "output.json"}
		return []string{files[sg.rng.Intn(len(files))]}
		
	case "grep":
		patterns := []string{"error", "success", "test", "[0-9]+"}
		files := []string{"test.txt", "data.log", "output.json"}
		return []string{patterns[sg.rng.Intn(len(patterns))], files[sg.rng.Intn(len(files))]}
		
	case "sed":
		commands := []string{"s/old/new/", "1d", "/pattern/p"}
		files := []string{"test.txt", "data.log"}
		return []string{commands[sg.rng.Intn(len(commands))], files[sg.rng.Intn(len(files))]}
		
	default:
		return []string{}
	}
}

// isCooperativeBinary checks if a binary supports cooperative fuzzing
func (sg *SmartGenerator) isCooperativeBinary(path string) bool {
	// Check cache first
	if supported, ok := sg.cooperativeCache[path]; ok {
		return supported
	}

	// Try to introspect for cooperative support
	if info, err := sg.introspector.IntrospectBinary(path); err == nil {
		// Binary supports cooperative fuzzing if it has MCP_SCRIPTTEST_GENERATE capability
		supported := info.SupportsCooperativeFuzzing
		sg.cooperativeCache[path] = supported
		return supported
	}

	// Default to false
	sg.cooperativeCache[path] = false
	return false
}

// generateCooperativeCommand uses the binary's generate mode
func (sg *SmartGenerator) generateCooperativeCommand(path string) string {
	// Generate a seed for the binary
	seed := sg.rng.Int63()

	// Call the binary with generate mode to get valid command
	command, err := sg.introspector.GenerateCommand(path, seed)
	if err != nil {
		// Fallback to traditional generation
		return sg.generateTraditionalExecCommand()
	}

	return fmt.Sprintf("exec %s", command)
}

// generateTraditionalExecCommand falls back to traditional generation
func (sg *SmartGenerator) generateTraditionalExecCommand() string {
	commands := []string{
		"echo test",
		"echo 'hello world'",
		"true",
		"false",
		"date",
		"pwd",
		"cat test.txt",
		"grep pattern file.txt",
	}
	return fmt.Sprintf("exec %s", commands[sg.rng.Intn(len(commands))])
}

// ValidateAndRegenerate generates a command and validates it
func (sg *SmartGenerator) ValidateAndRegenerate(maxAttempts int) string {
	for i := 0; i < maxAttempts; i++ {
		script := sg.Generate()
		
		// Extract exec commands and validate them
		lines := strings.Split(script, "\n")
		allValid := true
		
		for _, line := range lines {
			if strings.HasPrefix(line, "exec ") {
				command := strings.TrimPrefix(line, "exec ")
				if valid, _ := sg.introspector.ValidateCommand(command); !valid {
					allValid = false
					break
				}
			}
		}
		
		if allValid {
			return script
		}
	}
	
	// Fallback to non-validated generation
	return sg.Generate()
}

// MCPSmartGenerator combines MCP focus with smart command generation
type MCPSmartGenerator struct {
	*SmartGenerator
}

// NewMCPSmartGenerator creates an MCP-focused smart generator
func NewMCPSmartGenerator(seed int64) *MCPSmartGenerator {
	config := SmartGeneratorConfig{
		GeneratorConfig: GeneratorConfig{
			DisabledCommands: map[string]bool{
				"exec": false, // Keep exec but make it smart
			},
			CommandWeights: map[string]float64{
				"mcp-trace": 5.0,
				"mcp-send":  3.0,
				"mcp-recv":  3.0,
				"exec":      2.0, // Lower weight for exec
			},
			FocusArea: "mcp",
		},
		EnableIntrospection: true,
		CommonTestBinaries:  true,
		ValidateCommands:    true,
	}
	
	return &MCPSmartGenerator{
		SmartGenerator: NewSmartGenerator(seed, config),
	}
}