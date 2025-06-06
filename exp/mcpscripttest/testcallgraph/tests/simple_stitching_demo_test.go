package tests

import (
	"testing"

	"github.com/tmc/mcp/exp/callgraph/testcallgraph"
)

func TestSimpleStitchingDemo(t *testing.T) {
	// Create a simple stitcher
	stitcher := &testcallgraph.SimpleStitcher{}
	
	// Analyze a test script
	testContent := `
exec mcpdiff --help
exec echo "test"
exec mcp-spy -- trace.json
`
	
	connections := stitcher.AnalyzeAndStitch("demo.txt", testContent)
	
	t.Log("=== Simple Stitching Demo ===")
	t.Log("")
	t.Log("Test content:")
	t.Log(testContent)
	t.Log("")
	t.Log("Connections found:")
	for _, conn := range connections {
		t.Logf("  Line %d: %s -> %s (via %s)", 
			conn.TestLine, conn.TestFile, conn.Program, conn.MainPath)
	}
	t.Log("")
	t.Log("This shows how stitching connects test scripts to programs!")
	
	// Verify connections
	if len(connections) != 3 {
		t.Errorf("Expected 3 connections, got %d", len(connections))
	}
}

func TestMinimalStitchingExample(t *testing.T) {
	// Minimal example of stitching
	t.Log("=== Minimal Stitching Example ===")
	t.Log("")
	t.Log("Standard callgraph:")
	t.Log("  - Only sees static Go function calls")
	t.Log("  - Cannot see: test.txt -> exec mcpdiff")
	t.Log("")
	t.Log("Our stitching:")
	t.Log("  - Parses test scripts")
	t.Log("  - Finds 'exec mcpdiff' command")
	t.Log("  - Creates edge: test.txt:3 -> cmd/mcpdiff/main.go:main")
	t.Log("")
	t.Log("Result: Complete visibility into test->program relationships!")
}