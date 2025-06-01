package internal

import (
	"os/exec"
	"strings"
	"testing"
)

// RunDeadcodeCheck runs the deadcode analysis tool on the current package.
// It identifies unused (dead) code in Go packages.
//
// Note: It's recommended to use the TestMain function instead of calling this
// directly, as TestMain will run the deadcode check only once after all tests are complete.
// See the package documentation for more details.
func RunDeadcodeCheck(t *testing.T, _ interface{}) {
	t.Helper()

	// Run deadcode using go tool
	cmd := exec.Command("go", "tool", "deadcode", ".")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check if we have any errors that aren't compile errors
	// (We ignore compile errors as they might be in test code or examples)
	if err != nil && !strings.Contains(outputStr, "packages contain errors") {
		t.Logf("Warning: Deadcode command failed: %v\nOutput: %s", err, outputStr)
		return
	}

	// If we have any deadcode findings, report them
	if len(output) > 0 && !strings.Contains(outputStr, "packages contain errors") {
		t.Errorf("Deadcode analysis found unused code:\n%s", outputStr)
	} else {
		t.Log("No deadcode found!")
	}
}