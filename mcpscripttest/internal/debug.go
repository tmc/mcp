package internal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"rsc.io/script"
)

// StartDebugShellOnFailure starts an interactive debug shell for a failed test
func StartDebugShellOnFailure(ctx context.Context, engine *script.Engine, env []string, file string, testErr error) {
	fmt.Printf("Debug shell for test file: %s\n", file)
	fmt.Printf("Test failed with error: %v\n\n", testErr)
	fmt.Println("Available commands:")
	fmt.Println("  help               - Show this help message")
	fmt.Println("  list               - List test file contents")
	fmt.Println("  env                - Show environment variables")
	fmt.Println("  run <line-number>  - Run the test from the specified line")
	fmt.Println("  cmds               - List available commands")
	fmt.Println("  conds              - List available conditions")
	fmt.Println("  exit/quit          - Exit the debug shell")
	fmt.Println("")

	workdir, _ := os.Getwd()
	_, err := script.NewState(ctx, workdir, env)
	if err != nil {
		fmt.Printf("Error creating script state: %v\n", err)
		return
	}

	// Read the test file for debugging reference
	content, err := os.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading test file: %v\n", err)
		return
	}
	lines := strings.Split(string(content), "\n")

	// Start the interactive shell
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("debug> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  help               - Show this help message")
			fmt.Println("  list               - List test file contents")
			fmt.Println("  env                - Show environment variables")
			fmt.Println("  run <line-number>  - Run the test from the specified line")
			fmt.Println("  cmds               - List available commands")
			fmt.Println("  conds              - List available conditions")
			fmt.Println("  exit/quit          - Exit the debug shell")

		case "list":
			for i, line := range lines {
				fmt.Printf("%3d: %s\n", i+1, line)
			}

		case "env":
			for _, e := range env {
				fmt.Println(e)
			}

		case "cmds":
			fmt.Println("Available commands:")
			for name := range engine.Cmds {
				fmt.Printf("  %s\n", name)
			}

		case "conds":
			fmt.Println("Available conditions:")
			for name := range engine.Conds {
				fmt.Printf("  %s\n", name)
			}

		case "run":
			if len(args) != 1 {
				fmt.Println("Usage: run <line-number>")
				continue
			}
			// TODO: Implement partial test execution from specific line
			fmt.Println("Partial test execution not yet implemented")

		case "exit", "quit":
			fmt.Println("Exiting debug shell")
			return

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			fmt.Println("Type 'help' for available commands")
		}
	}
}
