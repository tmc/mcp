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
