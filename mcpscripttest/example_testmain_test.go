package mcpscripttest

import (
	"os"
	"testing"
)

// Example of how to use TestMain
// The actual TestMain for this package is in main_test.go
func ExampleTestMain() {
	// This function is not executed, just an example
	// of how to use the TestMain function

	// m is *testing.M passed to the package's TestMain function
	var m *testing.M

	// Use default options
	opts := DefaultTestMainOptions()

	// Optionally customize options
	opts.RunDeadcodeCheck = true // This is the default, just showing how to customize
	opts.Cleanup = func() {
		// Any cleanup after all tests run
		// For example, removing temporary files or directories
	}

	// Run tests and get exit code
	// os.Exit expects an int as parameter
	code := RunTestMain(m, opts)
	os.Exit(code)
}
