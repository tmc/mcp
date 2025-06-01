package mcpscripttest

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest/testcallgraph"
)

func TestEnhancedStitchingDemo(t *testing.T) {
	// Run the enhanced demo
	Test(t, "testdata/enhanced_stitching_concept.txt")
}

func TestEnhancedStitchingConcept(t *testing.T) {
	// Use enhanced stitcher
	stitcher := testcallgraph.NewEnhancedStitcher()
	
	// For demo purposes, we'll manually set up the analysis result
	// In real usage, you'd use stitcher.AnalyzeScriptTest(filename)
	
	// Manually analyze for demo
	stitcher.TestToProgramMap["demo.txt"] = []testcallgraph.ProgramExecution{
		{Program: "mcpdiff", Line: 3, ExecutedBy: "exec"},
		{Program: "mcpdiff", Line: 4, ExecutedBy: "mcpdiff"},
		{Program: "mcp-spy", Line: 5, ExecutedBy: "mcp-spy"},
		{Program: "server", Line: 6, ExecutedBy: "mcp-server-start", IsServer: true},
		{Program: "mcp-serve", Line: 7, ExecutedBy: "mcp-serve", IsServer: true},
	}
	
	edges := stitcher.CreateCallGraphConnections("demo.txt")
	
	t.Log("Enhanced Stitching Results:")
	t.Log("==========================")
	for _, edge := range edges {
		t.Logf("  %s", edge)
	}
	t.Log("")
	t.Log("Key Insights:")
	t.Log("- Handles exec commands (standard)")
	t.Log("- Handles custom MCP commands (mcpdiff, mcp-spy, etc)")
	t.Log("- Identifies server processes (IsServer flag)")
	t.Log("- Creates complete call graph edges")
}

// TestEnhancedStitchingSummary summarizes the enhancement
func TestEnhancedStitchingSummary(t *testing.T) {
	t.Log("=== Enhanced Stitching: Beyond exec Commands ===")
	t.Log("")
	t.Log("Standard stitching only handles: exec <program>")
	t.Log("")
	t.Log("Enhanced stitching also handles:")
	t.Log("  - Custom MCP commands (mcpdiff, mcp-spy, mcp-serve)")
	t.Log("  - Server start commands (mcp-server-start)")
	t.Log("  - Nested commands (mcp-server-start -- ./mcpd -- node)")
	t.Log("  - Interpreter commands (go run, python, node)")
	t.Log("")
	t.Log("Benefits:")
	t.Log("  - Complete visibility into ALL program executions")
	t.Log("  - Tracks server processes separately")
	t.Log("  - Handles complex command patterns")
	t.Log("  - Provides richer call graph data")
	t.Log("")
	t.Log("Implementation: testcallgraph/enhanced_stitcher.go")
}

