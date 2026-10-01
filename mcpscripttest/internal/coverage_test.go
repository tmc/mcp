package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/mcpscripttest/coverage"
)

func TestWithCoverageOptionsSetsEnvironment(t *testing.T) {
	dir := t.TempDir()
	scriptFile := filepath.Join(dir, "coverage.txt")
	script := "exec sh -c 'test -n \"$GOCOVERDIR\"'\n"
	if err := os.WriteFile(scriptFile, []byte(script), 0644); err != nil {
		t.Fatal(err)
	}

	coverageDir := filepath.Join(dir, "coverage")
	TestWithCoverageOptions(t, scriptFile, &coverage.CoverageOptions{
		Enabled:   true,
		OutputDir: coverageDir,
	})
}
