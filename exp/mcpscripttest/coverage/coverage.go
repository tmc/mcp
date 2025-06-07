// Package mcpscripttest provides testing utilities for MCP tools using script-based testing.
package coverage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// CoverageOptions configures how coverage data is collected and managed
type CoverageOptions struct {
	// Enabled determines if coverage should be collected
	Enabled bool

	// OutputDir is the directory where coverage data will be written
	// If empty, a standard temporary directory will be used
	OutputDir string

	// PerTestSubdir determines if each test should get its own subdirectory for coverage
	// Disabled by default since it makes tests run serially instead of in parallel
	PerTestSubdir bool

	// VerboseOutput enables detailed logging about coverage operations
	VerboseOutput bool
}

// DefaultCoverageOptions returns the default coverage options
func DefaultCoverageOptions() *CoverageOptions {
	return &CoverageOptions{
		Enabled:       false,
		OutputDir:     "",
		PerTestSubdir: false,
		VerboseOutput: false,
	}
}

// SetupTestCoverage sets up coverage for a specific test
// It returns a cleanup function that should be deferred
func SetupTestCoverage(t *testing.T, opts *CoverageOptions) func() {
	t.Helper()

	if opts == nil {
		opts = DefaultCoverageOptions()
	}

	// Skip if coverage is disabled
	if !opts.Enabled {
		return func() {}
	}

	// Save original GOCOVERDIR to restore it later
	originalCoverDir := os.Getenv("GOCOVERDIR")
	if opts.VerboseOutput {
		t.Logf("Original GOCOVERDIR: %s", originalCoverDir)
		t.Logf("Coverage options: %+v", opts)
	}

	// Determine test-specific coverage directory
	testCoverDir := originalCoverDir
	if opts.PerTestSubdir {
		// Use the test name from options if provided, otherwise use t.Name()
		testName := t.Name()

		// Sanitize the test name for use as a directory name
		testName = strings.ReplaceAll(testName, "/", "_")
		testName = strings.ReplaceAll(testName, " ", "_")

		// Add timestamp to ensure uniqueness
		timestamp := time.Now().Format("20060102-150405.000")
		testSubdir := fmt.Sprintf("%s-%s", testName, timestamp)

		testCoverDir = filepath.Join(originalCoverDir, testSubdir)
		if opts.VerboseOutput {
			t.Logf("Test-specific coverage directory: %s", testCoverDir)
		}

		if err := os.MkdirAll(testCoverDir, 0755); err != nil {
			if opts.VerboseOutput {
				t.Logf("Warning: Failed to create test-specific coverage directory: %v", err)
			}
			// Fall back to the original coverage directory
			testCoverDir = originalCoverDir
		} else if opts.VerboseOutput {
			t.Logf("Created test-specific coverage directory: %s", testCoverDir)
		}

		// Set GOCOVERDIR to the test-specific directory
		if opts.VerboseOutput {
			t.Logf("Setting GOCOVERDIR to: %s", testCoverDir)
		}
		t.Setenv("GOCOVERDIR", testCoverDir)
	}

	// Return cleanup function
	return func() {
		if opts.VerboseOutput {
			t.Log("Cleaning up coverage data...")
		}

		// Restore original GOCOVERDIR
		if originalCoverDir != testCoverDir {
			//t.Setenv("GOCOVERDIR", originalCoverDir)
		}

		// Merge coverage data back to the original directory
		cmd := exec.Command("go", "tool", "covdata", "merge", "-pcombine", "-i", testCoverDir, "-o", originalCoverDir)
		if err := cmd.Run(); err != nil {
			if opts.VerboseOutput {
				t.Logf("Warning: Failed to merge coverage data: %v", err)
			}
		} else if opts.VerboseOutput {
			t.Logf("Merged coverage data from %s to %s", testCoverDir, originalCoverDir)
		}

		// Check for coverage data
		if testCoverDir != "" && opts.VerboseOutput {
			if entries, err := os.ReadDir(testCoverDir); err == nil {
				var covFiles int
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), "covcounters.") ||
						strings.HasPrefix(entry.Name(), "covmeta.") {
						covFiles++
					}
				}

				if covFiles > 0 {
					t.Logf("Found %d coverage data files for test %s", covFiles, t.Name())
					t.Logf("To analyze this test's coverage: go tool covdata percent -i %s", testCoverDir)
				} else {
					t.Logf("WARNING: No coverage data found for test %s in %s", t.Name(), testCoverDir)
				}
			}
		}
	}
}

// EnablePerTestCoverage modifies test options to save per-test coverage data
func EnablePerTestCoverage(t *testing.T, rootOutputDir string) {
	t.Helper()

	// Ensure the root output directory exists
	if err := os.MkdirAll(rootOutputDir, 0755); err != nil {
		if testing.Verbose() {
			t.Logf("Warning: Failed to create coverage root directory: %v", err)
		}
		return
	}

	// Set GOCOVERDIR to the root output directory
	t.Setenv("GOCOVERDIR", rootOutputDir)

	if testing.Verbose() {
		t.Logf("Enabled per-test coverage in directory: %s", rootOutputDir)
		t.Logf("Each test will save coverage data in a separate subdirectory")
		t.Logf("To analyze combined coverage: go tool covdata percent -i %s", rootOutputDir)
	}
}

// isGoTesting returns true if we're running inside the go test tool
// This is useful for customizing output in tests
func isGoTesting() bool {
	// Check if executable name contains "test"
	exe := filepath.Base(os.Args[0])
	return strings.Contains(exe, "test") || strings.Contains(exe, "Test") ||
		strings.Contains(exe, ".test") || strings.Contains(exe, "_test")
}

// CheckCoverageWarning checks if coverage is enabled but no data is being collected
// and logs a warning with troubleshooting tips if needed.
// It can be called at the end of tests to verify coverage is working.
func CheckCoverageWarning(t *testing.T) {
	t.Helper()

	// Only check in verbose mode
	if !testing.Verbose() {
		return
	}

	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir == "" {
		// No GOCOVERDIR set, nothing to check
		return
	}

	// Check if directory exists and has any content
	if info, err := os.Stat(coverDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(coverDir)
		if err == nil {
			// Count coverage data files
			var covFiles int
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "covcounters.") ||
					strings.HasPrefix(entry.Name(), "covmeta.") {
					covFiles++
				}
			}

			if covFiles == 0 {
				t.Logf("WARNING: Coverage is enabled (GOCOVERDIR=%s) but no coverage data files were found.", coverDir)
				t.Logf("This may indicate one of the following issues:")
				t.Logf("1. The binary being tested wasn't built with -cover flag")
				t.Logf("2. The test didn't execute the binary with coverage instrumentation")
				t.Logf("3. Coverage data files are being written to a different location")
				t.Logf("TIP: Use 'go install -cover <package>' to build a coverage-instrumented binary")
			}
		}
	}
}

// CheckWarning checks and warns about coverage status.
func CheckWarning(t *testing.T) {
	t.Helper()
	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir == "" {
		t.Logf("Coverage not enabled (GOCOVERDIR not set)")
	} else {
		t.Logf("Coverage data should be in: %s", coverDir)
	}
}
