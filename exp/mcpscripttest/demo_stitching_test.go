package mcpscripttest

import (
	"fmt"
	"os"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest/testcallgraph"
)

func TestDemoStitching(t *testing.T) {
	// Create a simple test script
	testScript := `# Demo test script
exec echo "Hello"
exec mcpdiff --help
mcpdiff --version
mcp-spy -- mcpdiff test
mcp-server-start demo -- go run ./cmd/server/main.go
`

	// Create temp file
	tmpfile := t.TempDir() + "/demo_test.txt"
	if err := os.WriteFile(tmpfile, []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}

	// Use enhanced stitcher to analyze the test
	stitcher := testcallgraph.NewEnhancedStitcher()
	if err := stitcher.AnalyzeScriptTest(tmpfile); err != nil {
		t.Fatal(err)
	}

	// Get the connections
	edges := stitcher.CreateCallGraphConnections(tmpfile)

	// Print results
	fmt.Println("\n=== Stitching Demo Results ===")
	fmt.Printf("\nTest file: %s\n", tmpfile)
	fmt.Println("\nConnections found:")
	fmt.Println("-----------------")
	
	for _, edge := range edges {
		fmt.Printf("%s\n", edge)
	}

	fmt.Println("\nSummary:")
	fmt.Printf("- Found %d connections from test to programs\n", len(edges))
	
	// Count different types
	execCount := 0
	customCount := 0
	serverCount := 0
	
	for _, edge := range edges {
		switch edge.EdgeType {
		case "exec":
			execCount++
		default:
			customCount++
		}
		if edge.IsServer {
			serverCount++
		}
	}
	
	fmt.Printf("- %d regular exec commands\n", execCount)
	fmt.Printf("- %d custom MCP commands\n", customCount)
	fmt.Printf("- %d server processes\n", serverCount)

	// Verify we found expected programs
	expectedPrograms := map[string]bool{
		"echo": false,
		"mcpdiff": false,
		"mcp-spy": false,
		"server": false,
	}
	
	for _, edge := range edges {
		program := ""
		// Extract program name from the To field (e.g., "cmd/mcpdiff/main.go:main")
		if len(edge.To) > 4 {
			start := 4 // after "cmd/"
			end := len(edge.To)
			for i := start; i < end; i++ {
				if edge.To[i] == '/' {
					program = edge.To[start:i]
					break
				}
			}
		}
		if _, ok := expectedPrograms[program]; ok {
			expectedPrograms[program] = true
		}
	}
	
	fmt.Println("\nPrograms found:")
	for prog, found := range expectedPrograms {
		status := "✓"
		if !found {
			status = "✗"
		}
		fmt.Printf("  %s %s\n", status, prog)
	}
	
	// Assert at least some connections were found
	if len(edges) == 0 {
		t.Error("No connections found")
	}
}