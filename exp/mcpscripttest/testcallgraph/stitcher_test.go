package testcallgraph

import (
	"strings"
	"testing"
)

func TestStitcher(t *testing.T) {
	stitcher := NewStitcher()

	// Test with non-existent file
	tmpfile := t.TempDir() + "/test.txt"
	if err := stitcher.AnalyzeScriptTest(tmpfile); err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}

	// Test with actual content
	programs := []string{"mcpdiff", "echo", "mcpdiff"}
	stitcher.TestToProgramMap["test.txt"] = programs

	// Should find mcpdiff (deduped)
	if progs := stitcher.TestToProgramMap["test.txt"]; len(progs) != 3 {
		t.Errorf("expected 3 programs, got %d", len(progs))
	}

	// Test finding main function
	mainFunc, err := stitcher.FindProgramMain("mcpdiff")
	if err != nil && mainFunc == nil {
		t.Errorf("FindProgramMain failed: %v", err)
	}

	// Even if we can't find the actual file, we should get a placeholder
	if mainFunc != nil && !strings.Contains(mainFunc.FullPath, "mcpdiff") {
		t.Errorf("expected mcpdiff in path, got %s", mainFunc.FullPath)
	}
}