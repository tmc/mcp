package mcpscripttest

import (
	"os"
	"testing"
)

// TestMain for the mcpscripttest package itself
// This is the actual TestMain used by Go when running tests for this package
func TestMain(m *testing.M) {
	// Set up options for the test run
	opts := DefaultTestMainOptions()
	
	// We want to run deadcode check only once after all tests
	opts.RunDeadcodeCheck = true
	
	// Run all tests and then deadcode check
	code := RunTestMain(m, opts)
	
	// Exit with appropriate code
	os.Exit(code)
}