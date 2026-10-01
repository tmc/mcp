package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/mcpscripttest"
	"github.com/tmc/mcp/mcpscripttest/coverage"
)

// TestMCPConformance runs all the MCP protocol conformance tests
func TestMCPConformance(t *testing.T) {
	t.Skip("Conformance tests require mcp-scripttest-server with --stdio support which is not yet implemented")

	// Set up test environment
	coverage.SetupCoverageEnvironment(t)

	// Create options for the test
	options := mcpscripttest.DefaultOptions()

	// Add custom environment variables if needed
	options.AdditionalEnvVars = []string{"MCP_TEST_MODE", "MCP_CONFORMANCE"}

	// Create test configuration
	coverageOpts := mcpscripttest.DefaultCoverageOptions()
	coverageOpts.PerTestSubdir = true
	coverageOpts.VerboseOutput = testing.Verbose()
	coverageOpts.Enabled = true

	// Define the test path pattern (relative to this package)
	testPath := filepath.Join("testdata", "*.txt")

	// Log info about the tests
	if testing.Verbose() {
		t.Logf("Running MCP conformance tests from: %s", testPath)
		t.Logf("Coverage enabled: %v", coverageOpts.Enabled)
	}

	// Run the conformance tests with coverage
	mcpscripttest.TestWithCoverageOptions(t, testPath, coverageOpts, options)
}

// TestMCPConformanceIndividual runs each conformance test category separately
// This allows developers to focus on specific test categories
func TestMCPConformanceIndividual(t *testing.T) {
	t.Skip("Individual conformance tests require mcp-scripttest-server with --stdio support which is not yet implemented")

	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping individual conformance tests in short mode")
	}

	// Set up test environment
	coverage.SetupCoverageEnvironment(t)

	// Create options for the test
	options := mcpscripttest.DefaultOptions()

	// Add custom environment variables if needed
	options.AdditionalEnvVars = []string{"MCP_TEST_MODE", "MCP_CONFORMANCE"}

	// Create test configuration
	coverageOpts := mcpscripttest.DefaultCoverageOptions()
	coverageOpts.PerTestSubdir = true
	coverageOpts.VerboseOutput = testing.Verbose()
	coverageOpts.Enabled = true

	// Define the directory containing tests (relative to this package)
	testDir := "testdata"

	// Get test files
	files, err := filepath.Glob(filepath.Join(testDir, "*.txt"))
	if err != nil {
		t.Fatalf("Failed to find test files: %v", err)
	}

	// Run each test file separately
	for _, file := range files {
		testName := filepath.Base(file)
		testName = testName[:len(testName)-4] // Remove .txt extension

		t.Run(testName, func(t *testing.T) {
			// Run the conformance test with coverage
			mcpscripttest.TestWithCoverageOptions(t, file, coverageOpts, options)
		})
	}
}

// init sets up the test environment based on the MCP_CONFORMANCE environment variable
func init() {
	// Setup any global test environment
	os.Setenv("MCP_CONFORMANCE", "true")
}
