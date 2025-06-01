package mcpscripttest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest/testcallgraph"
)

func TestBashCallGraphIntegration(t *testing.T) {
	// Test that BashStitcher properly analyzes bash scripts
	stitcher := testcallgraph.NewBashStitcher()
	
	// Create test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	
	// Write a simple test with bash script
	content := `# Test bash scripts
env MCP_BASH_COVERAGE=1
bash 'echo "Direct command"'

exec chmod +x script.sh
bash './script.sh'

-- script.sh --
#!/bin/bash
echo "From script"
mcpdiff file1.mcp file2.mcp
`
	
	if err := writeTestFile(testFile, content); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Analyze the test
	if err := stitcher.AnalyzeScriptTest(testFile); err != nil {
		t.Fatalf("Failed to analyze test: %v", err)
	}
	
	// Get bash executions
	executions := stitcher.BashScriptMap[testFile]
	if len(executions) == 0 {
		t.Fatal("Expected to find bash executions")
	}
	
	// Create call graph
	edges := stitcher.CreateBashCallGraph(testFile)
	if len(edges) == 0 {
		t.Fatal("Expected to find call graph edges")
	}
	
	// Check for bash edges
	foundBashEdge := false
	for _, edge := range edges {
		if edge.EdgeType == "bash:bash" || edge.EdgeType == "bash:exec" {
			foundBashEdge = true
			break
		}
	}
	
	if !foundBashEdge {
		t.Fatal("Expected to find bash edges in call graph")
	}
}

func writeTestFile(path, content string) error {
	// Write the test file
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}