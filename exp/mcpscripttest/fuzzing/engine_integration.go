package fuzzing

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// EngineValidator interacts with test binaries in validation mode
type EngineValidator struct {
	// Environment variable to signal validation mode
	ValidationEnvVar string

	// Timeout for validation attempts
	ValidationTimeout time.Duration
}

// NewEngineValidator creates a new engine validator
func NewEngineValidator() *EngineValidator {
	return &EngineValidator{
		ValidationEnvVar:  "MCP_SCRIPTTEST_VALIDATE_ONLY",
		ValidationTimeout: 100 * time.Millisecond,
	}
}

// ValidateBinaryFlags checks if a binary accepts the given flags
func (ev *EngineValidator) ValidateBinaryFlags(binary string, args []string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ev.ValidationTimeout)
	defer cancel()

	// Create command with validation environment variable
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=1", ev.ValidationEnvVar))

	// Capture output
	_, err := cmd.CombinedOutput()

	// Check for specific validation signals
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Special exit codes for validation:
			// 0: Valid flags
			// 1: Invalid flags
			// 2: Validation not supported
			switch exitErr.ExitCode() {
			case 0:
				return true, nil
			case 1:
				return false, nil
			case 2:
				// Binary doesn't support validation mode
				return ev.fallbackValidation(binary, args)
			}
		}
		return false, err
	}

	// If no error, flags are valid
	return true, nil
}

// fallbackValidation uses heuristics when binary doesn't support validation mode
func (ev *EngineValidator) fallbackValidation(binary string, args []string) (bool, error) {
	// Try running with --help to see if flags are mentioned
	ctx, cancel := context.WithTimeout(context.Background(), ev.ValidationTimeout)
	defer cancel()

	helpCmd := exec.CommandContext(ctx, binary, "--help")
	helpOutput, _ := helpCmd.CombinedOutput()
	helpText := string(helpOutput)

	// Check if our flags appear in help text
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
			flag := strings.TrimLeft(arg, "-")
			if !strings.Contains(helpText, flag) {
				// Flag not found in help text, probably invalid
				return false, nil
			}
		}
	}

	// All flags found in help text
	return true, nil
}

// SmartGeneratorWithEngine extends SmartGenerator with engine validation
type SmartGeneratorWithEngine struct {
	*SmartGenerator
	validator *EngineValidator
}

// NewSmartGeneratorWithEngine creates a generator that validates with test binaries
func NewSmartGeneratorWithEngine(seed int64, config SmartGeneratorConfig) *SmartGeneratorWithEngine {
	sg := NewSmartGenerator(seed, config)

	return &SmartGeneratorWithEngine{
		SmartGenerator: sg,
		validator:      NewEngineValidator(),
	}
}

// generateValidatedExecCommand generates and validates exec commands
func (sge *SmartGeneratorWithEngine) generateValidatedExecCommand(g *SpecializedGenerator) string {
	maxAttempts := 5

	for i := 0; i < maxAttempts; i++ {
		// Generate a command
		command := sge.generateSmartExecCommand(g)

		// Parse the command
		parts := strings.Fields(strings.TrimPrefix(command, "exec "))
		if len(parts) == 0 {
			continue
		}

		binary := parts[0]
		args := parts[1:]

		// Validate with the engine
		if valid, err := sge.validator.ValidateBinaryFlags(binary, args); valid || err != nil {
			return command
		}

		// Invalid, try again
	}

	// Fallback to a known-good command
	return "exec echo 'validation test'"
}

// GenerateWithValidation creates a script with validated commands
func (sge *SmartGeneratorWithEngine) GenerateWithValidation() string {
	// Override exec generator with validated version
	for i, cmd := range sge.commands {
		if cmd.Name == "exec" {
			sge.commands[i].Generator = sge.generateValidatedExecCommand
		}
	}

	return sge.Generate()
}

// Example of how test binaries could implement validation mode:
/*
func main() {
	// Check for validation mode
	if os.Getenv("MCP_SCRIPTTEST_VALIDATE_ONLY") == "1" {
		// Parse flags but don't run the actual program
		flag.Parse()

		// Check if all flags are valid
		if err := validateFlags(); err != nil {
			os.Exit(1) // Invalid flags
		}

		os.Exit(0) // Valid flags
	}

	// Normal execution
	flag.Parse()
	runProgram()
}

func validateFlags() error {
	// Custom validation logic
	// Return error if flags are invalid
	return nil
}
*/
