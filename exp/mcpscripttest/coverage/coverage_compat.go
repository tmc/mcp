package coverage

import (
	"testing"
)

// SetupCoverageEnvironment is a compatibility function that sets up coverage environment
func SetupCoverageEnvironment(t *testing.T) {
	t.Helper()
	// This is a simplified implementation - the actual implementation might need more logic
	setupEnv := SetupTestCoverage(t, DefaultCoverageOptions())
	t.Cleanup(setupEnv)
}

// TestCoverageOptions provides coverage options for tests
type TestCoverageOptions struct {
	CoverageDir string
	Enabled     bool
}

// TestWithCoverageOptions is a compatibility function for tests with coverage options
func TestWithCoverageOptions(t *testing.T, pattern string, opts *TestCoverageOptions) {
	t.Helper()
	coverageOpts := &CoverageOptions{
		Enabled:   true,
		OutputDir: opts.CoverageDir,
	}
	cleanup := SetupTestCoverage(t, coverageOpts)
	defer cleanup()
	// Note: Test function needs to be implemented in internal package
}