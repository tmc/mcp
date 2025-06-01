package mcpscripttest

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rsc.io/script"
)

// TestRunner runs mcpscripttest tests independent of the Go testing framework
type TestRunner struct {
	Options      *MCPScripttestOptions
	CoverageOpts *CoverageOptions
	Verbose      bool
}

// RunTests runs the specified test files and returns the number of failures
func (r *TestRunner) RunTests(pattern string) int {
	// Find files matching the pattern
	files, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Printf("Error finding test files: %v\n", err)
		return 1
	}

	if len(files) == 0 {
		fmt.Printf("No script files found matching pattern: %s\n", pattern)
		return 1
	}

	// Set up the script engine with MCP commands
	engine := NewEngine(r.Options)

	// Get environment variables
	env := getTestEnvironment(r.Options.AdditionalEnvVars)

	// Create a context
	ctx := context.Background()
	if r.Options.RunDeadcodeCheck {
		// Add a timeout if deadcode check is enabled
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
	}

	// Run each test file
	var failures int
	for _, file := range files {
		baseName := filepath.Base(file)
		fmt.Printf("===== Running Test: %s =====\n", baseName)

		// Set up test-specific environment
		testEnv := make([]string, len(env))
		copy(testEnv, env)
		if r.CoverageOpts != nil && r.CoverageOpts.PerTestSubdir {
			testCoverageEnv := setupTestSpecificCoverage(baseName, r.CoverageOpts)
			testEnv = append(testEnv, testCoverageEnv...)
		}

		// Create a state for the test
		workdir := filepath.Dir(file)
		state, err := script.NewState(ctx, workdir, testEnv)
		if err != nil {
			fmt.Printf("Error creating state for %s: %v\n", baseName, err)
			continue
		}

		// Create a collector for test output
		var collector io.Writer = os.Stdout
		if !r.Verbose {
			collector = io.Discard
		}

		// Open the test file
		testFile, err := os.Open(file)
		if err != nil {
			fmt.Printf("Error opening %s: %v\n", baseName, err)
			state.CloseAndWait(collector)
			continue
		}
		defer testFile.Close()

		// Run the test
		startTime := time.Now()
		reader := bufio.NewReader(testFile)
		err = engine.Execute(state, file, reader, collector)
		duration := time.Since(startTime)

		// Close the state
		state.CloseAndWait(collector)

		// Report the result
		if err != nil {
			fmt.Printf("FAIL: %s (%.2fs)\n", baseName, duration.Seconds())
			fmt.Printf("Error: %v\n", err)
			failures++

			// If debug mode is enabled, start the debug shell
			if r.Options.DebugMode {
				fmt.Println("\nStarting debug shell for failed test...")
				StartDebugShellOnFailure(ctx, engine, testEnv, file, err)
			}
		} else {
			fmt.Printf("PASS: %s (%.2fs)\n", baseName, duration.Seconds())
		}
	}

	// Return the number of failures
	return failures
}

// outputCollector collects and optionally displays test output
type outputCollector struct {
	verbose bool
}

// Log implements the scripttest.Logger interface
func (c *outputCollector) Log(msg string) {
	if c.verbose {
		fmt.Println(msg)
	}
}

// setupTestSpecificCoverage sets up coverage for a specific test
func setupTestSpecificCoverage(testName string, opts *CoverageOptions) []string {
	// Check if we're running in coverage mode
	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir == "" {
		return nil
	}

	// Create a test-specific subdirectory
	testID := strings.TrimSuffix(testName, filepath.Ext(testName))
	testCoverDir := filepath.Join(coverDir, testID)
	
	// Create the directory
	err := os.MkdirAll(testCoverDir, 0755)
	if err != nil {
		fmt.Printf("Warning: Failed to create test-specific coverage directory: %v\n", err)
		return nil
	}

	if opts.VerboseOutput {
		fmt.Printf("Collecting coverage data for %s in %s\n", testName, testCoverDir)
	}

	// Return the new GOCOVERDIR setting
	return []string{"GOCOVERDIR=" + testCoverDir}
}