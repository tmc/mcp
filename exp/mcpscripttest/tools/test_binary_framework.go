package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

// TestBinaryMode defines the mode for test binary operation
type TestBinaryMode int

const (
	// Normal execution mode
	ModeNormal TestBinaryMode = iota
	// Validation mode - check if flags are valid
	ModeValidate
	// Generation mode - generate valid command line
	ModeGenerate
	// Introspection mode - output binary capabilities
	ModeIntrospect
)

// TestBinaryConfig configures the test binary behavior
type TestBinaryConfig struct {
	// BinaryName is the name of this binary
	BinaryName string

	// SupportedFlags defines available flags
	SupportedFlags []FlagDefinition

	// AcceptsStdin indicates if the binary accepts stdin
	AcceptsStdin bool

	// RequiredArgs defines required positional arguments
	RequiredArgs []ArgDefinition

	// OptionalArgs defines optional positional arguments
	OptionalArgs []ArgDefinition

	// GenerateFunc is called to generate valid command lines
	GenerateFunc func(seed int64) (string, error)

	// ValidateFunc is called to validate a command line
	ValidateFunc func(args []string) error

	// ExecuteFunc is the actual program logic
	ExecuteFunc func() error
}

// FlagDefinition describes a command-line flag
type FlagDefinition struct {
	Name        string
	ShortName   string
	Type        string // "bool", "string", "int"
	Default     interface{}
	Description string
	Required    bool
}

// ArgDefinition describes a positional argument
type ArgDefinition struct {
	Name        string
	Description string
	Pattern     string // regex pattern for validation
}

// TestMainWithFuzzing provides a test-aware main function
func TestMainWithFuzzing(config TestBinaryConfig) {
	// Determine mode from environment
	mode := ModeNormal

	if os.Getenv("MCP_SCRIPTTEST_VALIDATE_ONLY") == "1" {
		mode = ModeValidate
	} else if os.Getenv("MCP_SCRIPTTEST_GENERATE") == "1" {
		mode = ModeGenerate
	} else if os.Getenv("MCP_SCRIPTTEST_INTROSPECT") == "1" {
		mode = ModeIntrospect
	}

	switch mode {
	case ModeValidate:
		handleValidateMode(config)
	case ModeGenerate:
		handleGenerateMode(config)
	case ModeIntrospect:
		handleIntrospectMode(config)
	default:
		handleNormalMode(config)
	}
}

// handleValidateMode validates command-line arguments
func handleValidateMode(config TestBinaryConfig) {
	// Parse flags without running the program
	flag.Parse()

	// Custom validation if provided
	if config.ValidateFunc != nil {
		if err := config.ValidateFunc(flag.Args()); err != nil {
			os.Exit(1) // Invalid
		}
	}

	// Basic validation
	if err := validateBasicRequirements(config, flag.Args()); err != nil {
		os.Exit(1) // Invalid
	}

	os.Exit(0) // Valid
}

// handleGenerateMode generates a valid command line
func handleGenerateMode(config TestBinaryConfig) {
	// Get seed from environment or args
	seed := int64(0)
	seedStr := os.Getenv("MCP_SCRIPTTEST_FUZZ_SEED")
	if seedStr != "" {
		fmt.Sscanf(seedStr, "%d", &seed)
	} else if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &seed)
	}

	// Use custom generation function if provided
	if config.GenerateFunc != nil {
		cmdLine, err := config.GenerateFunc(seed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(cmdLine)
		os.Exit(0)
	}

	// Default generation
	cmdLine := generateDefaultCommand(config, seed)
	fmt.Print(cmdLine)
	os.Exit(0)
}

// handleIntrospectMode outputs binary capabilities as JSON
func handleIntrospectMode(config TestBinaryConfig) {
	info := BinaryCapabilities{
		BinaryName:   config.BinaryName,
		AcceptsStdin: config.AcceptsStdin,
		Flags:        config.SupportedFlags,
		RequiredArgs: config.RequiredArgs,
		OptionalArgs: config.OptionalArgs,
	}

	if err := json.NewEncoder(os.Stdout).Encode(info); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding capabilities: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// handleNormalMode runs the program normally
func handleNormalMode(config TestBinaryConfig) {
	flag.Parse()

	if config.ExecuteFunc != nil {
		if err := config.ExecuteFunc(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// BinaryCapabilities describes what a binary can do
type BinaryCapabilities struct {
	BinaryName   string           `json:"binary_name"`
	AcceptsStdin bool             `json:"accepts_stdin"`
	Flags        []FlagDefinition `json:"flags"`
	RequiredArgs []ArgDefinition  `json:"required_args"`
	OptionalArgs []ArgDefinition  `json:"optional_args"`
}

// validateBasicRequirements checks basic command-line requirements
func validateBasicRequirements(config TestBinaryConfig, args []string) error {
	// Check required arguments
	if len(args) < len(config.RequiredArgs) {
		return fmt.Errorf("missing required arguments")
	}

	// Additional validation can be added here
	return nil
}

// generateDefaultCommand generates a command line using default logic
func generateDefaultCommand(config TestBinaryConfig, seed int64) string {
	rng := rand.New(rand.NewSource(seed))
	parts := []string{config.BinaryName}

	// Add some flags randomly
	for _, flag := range config.SupportedFlags {
		if rng.Float64() < 0.5 { // 50% chance to include each flag
			if flag.ShortName != "" && rng.Float64() < 0.5 {
				parts = append(parts, flag.ShortName)
			} else if flag.Name != "" {
				parts = append(parts, flag.Name)
			}

			// Add value for non-bool flags
			if flag.Type != "bool" {
				parts = append(parts, generateFlagValue(flag.Type, rng))
			}
		}
	}

	// Add required arguments
	for _, arg := range config.RequiredArgs {
		parts = append(parts, generateArgValue(arg, rng))
	}

	// Add some optional arguments
	for _, arg := range config.OptionalArgs {
		if rng.Float64() < 0.3 { // 30% chance for optional args
			parts = append(parts, generateArgValue(arg, rng))
		}
	}

	return strings.Join(parts, " ")
}

// generateFlagValue generates a value for a flag type
func generateFlagValue(flagType string, rng *rand.Rand) string {
	switch flagType {
	case "string":
		values := []string{"test", "value", "output.txt", "/tmp/test"}
		return values[rng.Intn(len(values))]
	case "int":
		return fmt.Sprintf("%d", rng.Intn(100))
	default:
		return "value"
	}
}

// generateArgValue generates a value for an argument
func generateArgValue(arg ArgDefinition, rng *rand.Rand) string {
	// Generate based on pattern or name
	switch {
	case strings.Contains(arg.Name, "file"):
		files := []string{"test.txt", "data.log", "output.json"}
		return files[rng.Intn(len(files))]
	case strings.Contains(arg.Name, "pattern"):
		patterns := []string{"test", "[0-9]+", "error"}
		return patterns[rng.Intn(len(patterns))]
	default:
		return "test_value"
	}
}

// IsStdinAvailable checks if stdin has data available
func IsStdinAvailable() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
