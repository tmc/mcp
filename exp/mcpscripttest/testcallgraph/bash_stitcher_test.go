package testcallgraph

import (
	"os"
	"strings"
	"testing"
)

func TestBashStitcher(t *testing.T) {
	// Create a test script file
	testScript := `
# Test script that executes bash scripts
exec bash test_script.sh
exec sh helper.sh --verbose
bash ./scripts/deploy.sh prod
exec kcov --exclude-pattern=/usr coverage_out ./test.sh

# Custom command that runs a bash script  
mcp-server-start server -- ./server.sh
`

	// Write test file
	testFile := "test_bash_execution.txt"
	if err := os.WriteFile(testFile, []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)
	
	// Create bash stitcher and analyze
	bs := NewBashStitcher()
	if err := bs.AnalyzeScriptTest(testFile); err != nil {
		t.Fatalf("Failed to analyze test file: %v", err)
	}
	
	// Check that bash executions were detected
	executions := bs.BashScriptMap[testFile]
	if len(executions) != 5 {
		t.Errorf("Expected 5 bash executions, got %d", len(executions))
	}
	
	// Verify specific executions
	tests := []struct {
		scriptPath   string
		executedBy   string
		withCoverage bool
	}{
		{"test_script.sh", "bash", false},
		{"helper.sh", "sh", false},
		{"./scripts/deploy.sh", "bash", false},
		{"./test.sh", "exec", true},
		{"./server.sh", "mcp-server-start", false},
	}
	
	for i, test := range tests {
		if i >= len(executions) {
			break
		}
		exec := executions[i]
		if exec.ScriptPath != test.scriptPath {
			t.Errorf("Execution %d: expected script %s, got %s", i, test.scriptPath, exec.ScriptPath)
		}
		if exec.ExecutedBy != test.executedBy {
			t.Errorf("Execution %d: expected executed by %s, got %s", i, test.executedBy, exec.ExecutedBy)
		}
		if exec.WithCoverage != test.withCoverage {
			t.Errorf("Execution %d: expected coverage %v, got %v", i, test.withCoverage, exec.WithCoverage)
		}
	}
}

func TestBashCallGraph(t *testing.T) {
	// Create test bash script
	bashScript := `#!/bin/bash

function setup() {
    echo "Setting up..."
}

function deploy() {
    echo "Deploying..."
    setup
}

# Main execution
deploy
`
	
	// Write bash script
	scriptPath := "test_deploy.sh"
	if err := os.WriteFile(scriptPath, []byte(bashScript), 0755); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(scriptPath)
	
	// Create test file that executes the script
	testScript := `
exec bash test_deploy.sh
`
	
	testFile := "test_bash_callgraph.txt"
	if err := os.WriteFile(testFile, []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)
	
	// Analyze with bash stitcher
	bs := NewBashStitcher()
	if err := bs.AnalyzeScriptTest(testFile); err != nil {
		t.Fatalf("Failed to analyze test file: %v", err)
	}
	
	// Create call graph
	edges := bs.CreateBashCallGraph(testFile)
	
	// Should have at least one edge from test to script
	found := false
	for _, edge := range edges {
		if strings.Contains(edge.From, testFile) && strings.Contains(edge.To, scriptPath) {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected edge from test file to bash script not found")
	}
}

func TestBashCoverageReport(t *testing.T) {
	bs := NewBashStitcher()
	
	// Add mock coverage data
	bs.BashCoverageMap["script1.sh"] = &BashCoverage{
		ScriptPath:    "script1.sh",
		TotalLines:    100,
		ExecutedLines: map[int]bool{1: true, 5: true, 10: true},
		Functions: map[string]*BashFunction{
			"setup": {
				Name:      "setup",
				StartLine: 5,
				EndLine:   10,
				Called:    true,
				CallCount: 3,
			},
			"cleanup": {
				Name:      "cleanup",
				StartLine: 15,
				EndLine:   20,
				Called:    false,
			},
		},
	}
	
	report := bs.GetBashCoverageReport()
	
	// Check report contains expected information
	if !strings.Contains(report, "script1.sh") {
		t.Error("Report should contain script1.sh")
	}
	if !strings.Contains(report, "3/100") {
		t.Error("Report should show 3/100 lines covered")
	}
	if !strings.Contains(report, "called 3 times") {
		t.Error("Report should show setup function called 3 times")
	}
	if !strings.Contains(report, "not called") {
		t.Error("Report should show cleanup function not called")
	}
}