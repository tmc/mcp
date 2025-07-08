package tests

import (
	"strings"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest/testcallgraph"
)

func TestStitchingConcept(t *testing.T) {
	// Demonstrate the concept of stitching

	// 1. We have a test script
	testScript := `
exec mcpdiff --help
stdout 'Usage'
exec echo "test"
`

	// 2. Standard callgraph can't see these connections
	// (It only analyzes static Go code)

	// 3. Our simple stitcher finds the connections
	stitcher := &testcallgraph.SimpleStitcher{}
	connections := stitcher.AnalyzeAndStitch("demo_test.txt", testScript)

	// 4. Verify we found the connections
	if len(connections) != 2 {
		t.Errorf("expected 2 connections, got %d", len(connections))
	}

	// Check we found mcpdiff
	foundMcpdiff := false
	for _, conn := range connections {
		if conn.Program == "mcpdiff" {
			foundMcpdiff = true
			t.Logf("Found connection: %s:%d -> %s (%s)",
				conn.TestFile, conn.TestLine, conn.Program, conn.MainPath)
		}
	}

	if !foundMcpdiff {
		t.Error("did not find mcpdiff connection")
	}
}

func TestStitchingDemonstration(t *testing.T) {
	// Show what standard callgraph misses
	t.Log("Standard callgraph analyzes static Go code:")
	t.Log("  - It sees function calls within Go programs")
	t.Log("  - It CANNOT see which external programs are executed")
	t.Log("")
	t.Log("Our testcallgraph adds these missing connections:")

	// Use the enhanced stitcher
	stitcher := testcallgraph.NewEnhancedStitcher()
	// For demo, manually populate executions
	stitcher.TestToProgramMap["demo.txt"] = []testcallgraph.ProgramExecution{
		{Program: "mcpdiff", Line: 2, ExecutedBy: "exec"},
		{Program: "mcp-spy", Line: 3, ExecutedBy: "exec"},
		{Program: "mcpdiff", Line: 3, ExecutedBy: "mcp-spy"},
		{Program: "server", Line: 4, ExecutedBy: "mcp-server-start", IsServer: true},
	}

	edges := stitcher.CreateCallGraphConnections("demo.txt")

	t.Log("Found connections:")
	foundMcpdiff := false
	for _, edge := range edges {
		t.Logf("  %s", edge)
		if strings.Contains(edge.To, "mcpdiff") {
			foundMcpdiff = true
		}
	}

	if !foundMcpdiff {
		t.Error("demo did not show mcpdiff connection")
	}
}
