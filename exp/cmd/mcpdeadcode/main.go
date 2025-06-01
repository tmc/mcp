// Command mcpdeadcode provides a wrapper around the deadcode tool for MCP codebase
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	excludeDirs     = flag.String("exclude-dirs", "vendor,node_modules,.git", "Comma-separated list of directories to exclude")
	excludePatterns = flag.String("exclude-patterns", "*_test.go", "Comma-separated list of file patterns to exclude")
	verbose         = flag.Bool("v", false, "Enable verbose output")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] [packages]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Runs deadcode analysis on Go packages\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nIf no packages are specified, ./... is used\n")
	}
	flag.Parse()

	// Check if deadcode is installed
	if _, err := exec.LookPath("deadcode"); err != nil {
		fmt.Println("Deadcode tool not found, attempting to install...")
		cmd := exec.Command("go", "install", "golang.org/x/tools/cmd/deadcode@latest")
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Failed to install deadcode tool: %v\nOutput: %s\n", err, output)
			os.Exit(1)
		}
		fmt.Println("Deadcode tool installed successfully")
	}

	// Get packages to analyze
	packages := flag.Args()
	if len(packages) == 0 {
		packages = []string{"./..."}
	}

	// Build the command arguments
	args := []string{}

	// Add exclude patterns
	excludePatternsList := strings.Split(*excludePatterns, ",")
	for _, pattern := range excludePatternsList {
		if pattern != "" {
			args = append(args, "-exclude", pattern)
		}
	}

	// Add packages
	args = append(args, packages...)

	// Create and run the deadcode command
	if *verbose {
		fmt.Printf("Running deadcode %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("deadcode", args...)

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current working directory: %v\n", err)
		os.Exit(1)
	}

	// Set the command working directory
	cmd.Dir = cwd

	// Pipe stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Check if we should ignore the error (compile errors)
			if strings.Contains(string(exitErr.Stderr), "packages contain errors") {
				fmt.Println("Note: Some packages contain compile errors, which may affect deadcode analysis")
				os.Exit(0)
			}
			os.Exit(exitErr.ExitCode())
		}
		fmt.Printf("Error running deadcode: %v\n", err)
		os.Exit(1)
	}
}