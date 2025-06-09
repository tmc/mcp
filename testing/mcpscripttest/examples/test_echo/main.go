package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"

	"github.com/tmc/mcp/testing/mcpscripttest/tools"
)

var (
	uppercase = flag.Bool("u", false, "Convert to uppercase")
	repeat    = flag.Int("n", 1, "Number of times to repeat")
	separator = flag.String("s", " ", "Separator between repetitions")
)

func main() {
	config := tools.TestBinaryConfig{
		BinaryName: "test_echo",

		SupportedFlags: []tools.FlagDefinition{
			{
				Name:        "--uppercase",
				ShortName:   "-u",
				Type:        "bool",
				Description: "Convert to uppercase",
			},
			{
				Name:        "--repeat",
				ShortName:   "-n",
				Type:        "int",
				Default:     1,
				Description: "Number of times to repeat",
			},
			{
				Name:        "--separator",
				ShortName:   "-s",
				Type:        "string",
				Default:     " ",
				Description: "Separator between repetitions",
			},
		},

		AcceptsStdin: true,

		RequiredArgs: []tools.ArgDefinition{
			{
				Name:        "message",
				Description: "Message to echo",
			},
		},

		GenerateFunc: generateCommand,
		ValidateFunc: validateArgs,
		ExecuteFunc:  execute,
	}

	tools.TestMainWithFuzzing(config)
}

// generateCommand generates a valid test_echo command
func generateCommand(seed int64) (string, error) {
	rng := rand.New(rand.NewSource(seed))

	parts := []string{"test_echo"}

	// Randomly add flags
	if rng.Float64() < 0.3 {
		parts = append(parts, "-u")
	}

	if rng.Float64() < 0.3 {
		parts = append(parts, fmt.Sprintf("-n %d", rng.Intn(5)+1))
	}

	if rng.Float64() < 0.2 {
		seps := []string{" ", ", ", " - ", " | "}
		parts = append(parts, fmt.Sprintf("-s '%s'", seps[rng.Intn(len(seps))]))
	}

	// Add message
	messages := []string{
		"hello world",
		"test message",
		"fuzzing test",
		"echo $(date)",
		"line1\nline2",
	}

	parts = append(parts, fmt.Sprintf("'%s'", messages[rng.Intn(len(messages))]))

	return strings.Join(parts, " "), nil
}

// validateArgs validates command-line arguments
func validateArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing required message argument")
	}

	// Check that repeat count is positive
	flag.Parse()
	if *repeat < 1 {
		return fmt.Errorf("repeat count must be positive")
	}

	return nil
}

// execute runs the actual echo functionality
func execute() error {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		return fmt.Errorf("missing message argument")
	}

	message := strings.Join(args, " ")

	// Apply transformations
	if *uppercase {
		message = strings.ToUpper(message)
	}

	// Repeat with separator
	parts := make([]string, *repeat)
	for i := 0; i < *repeat; i++ {
		parts[i] = message
	}

	fmt.Println(strings.Join(parts, *separator))

	// Also read and echo stdin if available
	if tools.IsStdinAvailable() {
		// Read from stdin
		var input string
		fmt.Scanln(&input)
		if input != "" {
			fmt.Printf("From stdin: %s\n", input)
		}
	}

	return nil
}

// Helper function to check if stdin is available
func init() {
	// This would be part of the framework
}
