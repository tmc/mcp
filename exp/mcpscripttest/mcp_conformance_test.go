package mcpscripttest

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMCPConformance runs all the MCP protocol conformance tests
func TestMCPConformance(t *testing.T) {
	// Set up test environment
	SetupCoverageEnvironment(t)

	// Create options for the test
	options := DefaultOptions()

	// Add custom environment variables if needed
	options.AdditionalEnvVars = []string{"MCP_TEST_MODE", "MCP_CONFORMANCE"}

	// Create test configuration
	coverageOpts := &CoverageOptions{
		PerTestSubdir:    true,
		VerboseOutput:    testing.Verbose(),
		Enabled: true,
	}

	// Define the test path pattern
	testPath := filepath.Join("testdata", "mcp_conformance", "*.txt")

	// Log info about the tests
	if testing.Verbose() {
		t.Logf("Running MCP conformance tests from: %s", testPath)
		t.Logf("Coverage enabled: %v", coverageOpts.Enabled)
	}

	// Run the conformance tests with coverage
	testCoverageOpts := &TestCoverageOptions{
		CoverageDir: coverageOpts.OutputDir,
		Enabled:     coverageOpts.Enabled,
	}
	TestWithCoverageOptions(t, testPath, testCoverageOpts)
}

// TestMCPConformanceIndividual runs each conformance test category separately
// This allows developers to focus on specific test categories
func TestMCPConformanceIndividual(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping individual conformance tests in short mode")
	}

	// Set up test environment
	SetupCoverageEnvironment(t)

	// Create options for the test
	options := DefaultOptions()

	// Add custom environment variables if needed
	options.AdditionalEnvVars = []string{"MCP_TEST_MODE", "MCP_CONFORMANCE"}

	// Create test configuration
	coverageOpts := &CoverageOptions{
		PerTestSubdir:    true,
		VerboseOutput:    testing.Verbose(),
		Enabled: true,
	}

	// Define the directory containing tests
	testDir := filepath.Join("testdata", "mcp_conformance")

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
			testCoverageOpts := &TestCoverageOptions{
				CoverageDir: coverageOpts.OutputDir,
				Enabled:     coverageOpts.Enabled,
			}
			TestWithCoverageOptions(t, file, testCoverageOpts)
		})
	}
}

// init sets up the test environment based on the MCP_CONFORMANCE environment variable
func init() {
	// Setup any global test environment
	os.Setenv("MCP_CONFORMANCE", "true")
}

// disabledTestMain runs conformance tests with comprehensive tooling support
func disabledTestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Run deadcode check if enabled and not in coverage mode
	if os.Getenv("GOCOVERDIR") == "" && os.Getenv("MCP_SKIP_DEADCODE") == "" {
		// Run deadcode check after tests - we create a mock testing.T
		t := &testing.T{}
		RunDeadcodeCheck(t, nil)
		// Note: deadcode check errors are reported to the mock testing.T
		// but not propagated as test failures
	}

	// Clean up and exit with test status code
	os.Exit(code)
}