package testcallgraph

import (
	"os"
	"strings"
	"testing"
)

func TestEnhancedStitcher(t *testing.T) {
	stitcher := NewEnhancedStitcher()
	
	testContent := `
# Test with various command types
exec mcpdiff --help
mcpdiff trace1.json trace2.json
mcp-spy -- mcpdiff file1 file2
mcp-server-start myserver -- go run ./cmd/server/main.go
mcp-serve -- python mcp_server.py
mcp-server-send myserver {"method":"test"}
exec echo "test"
`
	
	// Create a temp file with content
	tmpfile := t.TempDir() + "/enhanced_test.txt"
	if err := os.WriteFile(tmpfile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Analyze the test
	if err := stitcher.AnalyzeScriptTest(tmpfile); err != nil {
		t.Fatal(err)
	}
	
	executions := stitcher.TestToProgramMap[tmpfile]
	if len(executions) != 6 {
		t.Errorf("expected 6 executions, got %d", len(executions))
		for i, exec := range executions {
			t.Logf("  [%d] %s -> %s (by %s)", i, exec.Command, exec.Program, exec.ExecutedBy)
		}
	}
	
	// Check specific programs were found
	programs := map[string]bool{}
	for _, exec := range executions {
		programs[exec.Program] = true
		t.Logf("Found: %s (line %d) executed by %s", exec.Program, exec.Line, exec.ExecutedBy)
	}
	
	expectedPrograms := []string{"mcpdiff", "mcp-spy", "server", "mcp-serve", "echo"}
	for _, prog := range expectedPrograms {
		if !programs[prog] {
			t.Errorf("expected to find program %s", prog)
		}
	}
}

func TestCustomCommandMapping(t *testing.T) {
	stitcher := NewEnhancedStitcher()
	
	tests := []struct {
		line     string
		expected string
		isServer bool
	}{
		{"mcp-spy -- mcpdiff test", "mcp-spy", false},
		{"mcpdiff file1 file2", "mcpdiff", false},
		{"mcp-server-start test -- go run ./cmd/myserver/main.go", "myserver", true},
		{"mcp-server-start test -- ./mcpd -- node server.js", "mcpd", true},
		{"mcp-serve -- python server.py", "mcp-serve", true},
		{"mcp-server-send test data", "", false}, // Doesn't execute new program
	}
	
	for _, test := range tests {
		t.Run(test.line, func(t *testing.T) {
			parts := strings.Fields(test.line)
			if len(parts) == 0 {
				return
			}
			
			cmdName := parts[0]
			mapping, ok := stitcher.CustomCommandMap[cmdName]
			if !ok {
				t.Errorf("no mapping for command %s", cmdName)
				return
			}
			
			prog := stitcher.extractProgramFromCustomCommand(cmdName, test.line, mapping)
			if prog != test.expected {
				t.Errorf("expected program %s, got %s", test.expected, prog)
			}
			
			if mapping.IsServer != test.isServer {
				t.Errorf("expected isServer=%v, got %v", test.isServer, mapping.IsServer)
			}
		})
	}
}

func TestServerCommandParsing(t *testing.T) {
	stitcher := NewEnhancedStitcher()
	
	tests := []struct {
		cmd      string
		expected string
	}{
		{"mcp-server-start test -- go run ./cmd/myapp/main.go", "myapp"},
		{"mcp-server-start test -- python mcp_server.py", "mcp_server"},
		{"mcp-server-start test -- node server.js", "server"},
		{"mcp-server-start test -- ./my-server --port 8080", "my-server"},
		{"server -- go run ./cmd/demo/main.go", "demo"},
		{"mcp-server-start multi -- ./mcpd -- go run server.go", "mcpd"},
	}
	
	for _, test := range tests {
		t.Run(test.cmd, func(t *testing.T) {
			prog := stitcher.extractProgramFromServerCommand(test.cmd)
			if prog != test.expected {
				t.Errorf("expected %s, got %s", test.expected, prog)
			}
		})
	}
}

func TestCallGraphEdgeCreation(t *testing.T) {
	stitcher := NewEnhancedStitcher()
	
	// Manually set up some executions
	stitcher.TestToProgramMap["test.txt"] = []ProgramExecution{
		{Program: "mcpdiff", Line: 3, ExecutedBy: "exec", IsServer: false},
		{Program: "server", Line: 5, ExecutedBy: "mcp-server-start", IsServer: true},
		{Program: "mcp-spy", Line: 7, ExecutedBy: "mcp-spy", IsServer: false},
	}
	
	edges := stitcher.CreateCallGraphConnections("test.txt")
	
	if len(edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(edges))
	}
	
	// Check edge properties
	for i, edge := range edges {
		t.Logf("Edge %d: %s", i, edge)
		
		if !strings.Contains(edge.From, "test.txt:") {
			t.Errorf("edge.From should contain test.txt:, got %s", edge.From)
		}
		
		if !strings.Contains(edge.To, "/main.go:main") {
			t.Errorf("edge.To should contain /main.go:main, got %s", edge.To)
		}
	}
	
	// Check server edge
	if edges[1].IsServer != true {
		t.Error("expected second edge to be a server")
	}
}

