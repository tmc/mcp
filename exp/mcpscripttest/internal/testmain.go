package internal

import (
	"os"
	"strings"
	"testing"
)

// TestMainOptions provides configuration options for TestMain
type TestMainOptions struct {
	// Whether to run deadcode check
	RunDeadcodeCheck bool
	// Function to run after all tests
	Cleanup func()
}

// DefaultTestMainOptions returns the default options for TestMain
func DefaultTestMainOptions() *TestMainOptions {
	return &TestMainOptions{
		RunDeadcodeCheck: true,
		Cleanup:          nil,
	}
}

// RunTestMain is a helper function that can be used in a package's TestMain function
// to standardize test initialization and cleanup, including running deadcode checks
// exactly once after all tests complete.
//
// Example usage:
//
//	func TestMain(m *testing.M) {
//		opts := mcpscripttest.DefaultTestMainOptions()
//		// Optionally customize options
//		code := mcpscripttest.RunTestMain(m, opts)
//		os.Exit(code)
//	}
func RunTestMain(m *testing.M, opts *TestMainOptions) int {
	// Use default options if none provided
	if opts == nil {
		opts = DefaultTestMainOptions()
	}

	// Run all tests
	code := m.Run()

	// If all tests passed and deadcode check is enabled, run it
	if code == 0 && opts.RunDeadcodeCheck {
		// Create a test instance just for running deadcode check
		deadcodeTest := &testing.T{}
		RunDeadcodeCheck(deadcodeTest, nil)

		// Convert test failure to exit code
		deadcodeOutput := captureTestOutput(deadcodeTest)
		if strings.Contains(deadcodeOutput, "FAIL") {
			// If deadcode check failed, return non-zero exit code
			code = 1
			// Print the output to stderr
			os.Stderr.WriteString(deadcodeOutput)
		}
	}

	// Run cleanup function if provided
	if opts.Cleanup != nil {
		opts.Cleanup()
	}

	return code
}

// captureTestOutput captures the output of a testing.T instance
// This is used to capture the output of the deadcode check
func captureTestOutput(t *testing.T) string {
	// The testing.T implementation doesn't expose a way to capture output directly,
	// but we can check if the test failed and assume it was due to deadcode
	if t.Failed() {
		return "FAIL: deadcode check found unused code\n"
	}
	return "PASS: no deadcode found\n"
}
