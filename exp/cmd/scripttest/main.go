package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/exp/scripttest"
)

func main() {
	var (
		dir             = flag.String("dir", ".", "Directory containing test scripts")
		pattern         = flag.String("pattern", "*.txt", "Pattern for test files")
		verbose         = flag.Bool("verbose", false, "Verbose output")
		continueOnError = flag.Bool("continue", false, "Continue on test failures")
		timeout         = flag.Duration("timeout", 0, "Timeout for each test (0 = no timeout)")
		work            = flag.Bool("work", false, "Keep temporary work directory")
		update          = flag.Bool("update", false, "Update test files with actual output")
		list            = flag.Bool("list", false, "List test files without running")
		env             = flag.String("env", "", "Additional environment variables (KEY=value,...)")
		bail            = flag.Int("bail", 0, "Stop after N failures (0 = no limit)")
	)
	flag.Parse()

	// Override with positional arguments
	if flag.NArg() > 0 {
		*pattern = flag.Arg(0)
	}

	// Find test files
	testFiles, err := findTestFiles(*dir, *pattern)
	if err != nil {
		log.Fatalf("Failed to find test files: %v", err)
	}

	if len(testFiles) == 0 {
		log.Fatalf("No test files found matching pattern: %s", *pattern)
	}

	// List mode
	if *list {
		for _, file := range testFiles {
			fmt.Println(file)
		}
		return
	}

	// Parse environment variables
	envVars := parseEnv(*env)

	// Create test runner
	runner := scripttest.NewRunner(scripttest.Options{
		Verbose:         *verbose,
		ContinueOnError: *continueOnError,
		Timeout:         *timeout,
		KeepWork:        *work,
		UpdateMode:      *update,
		Environment:     envVars,
		BailAfter:       *bail,
	})

	// Run tests
	results := runner.RunTests(testFiles)

	// Print summary
	printSummary(results)

	// Exit with appropriate code
	if results.Failed > 0 {
		os.Exit(1)
	}
}

func findTestFiles(dir, pattern string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}

		if matched {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func parseEnv(envStr string) map[string]string {
	if envStr == "" {
		return nil
	}

	env := make(map[string]string)
	for _, pair := range strings.Split(envStr, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	return env
}

func printSummary(results *scripttest.Results) {
	fmt.Printf("\n")
	fmt.Printf("Test Summary:\n")
	fmt.Printf("  Total:   %d\n", results.Total)
	fmt.Printf("  Passed:  %d\n", results.Passed)
	fmt.Printf("  Failed:  %d\n", results.Failed)
	fmt.Printf("  Skipped: %d\n", results.Skipped)

	if results.Failed > 0 {
		fmt.Printf("\nFailed tests:\n")
		for _, failure := range results.Failures {
			fmt.Printf("  %s: %s\n", failure.Test, failure.Error)
		}
	}

	fmt.Printf("\nDuration: %s\n", results.Duration)
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `scripttest - Run script-based tests

Usage:
  scripttest [options] [pattern]

Examples:
  scripttest                        # Run all *.txt files in current directory
  scripttest "test_*.txt"          # Run files matching pattern
  scripttest -dir tests/           # Run tests in specific directory
  scripttest -update failing.txt   # Update test with actual output
  scripttest -verbose -bail 1      # Stop after first failure with verbose output

Options:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Test File Format:
  # Comments start with #
  exec command args...    # Execute command
  stdin data             # Send data to stdin
  stdout expected        # Expect output on stdout
  stderr expected        # Expect output on stderr
  ! command              # Expect command to fail
  skip [message]         # Skip the test
  cd directory           # Change directory
  env KEY=value          # Set environment variable
  
Documentation:
  https://github.com/rogpeppe/go-internal/tree/master/testscript
`)
	}
}
