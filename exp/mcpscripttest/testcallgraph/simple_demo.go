package testcallgraph

import (
	"fmt"
	"strings"
)

// SimpleStitcher demonstrates the concept without complex dependencies
type SimpleStitcher struct {
	connections []Connection
}

// Connection represents a test-to-program connection
type Connection struct {
	TestFile string
	TestLine int
	Program  string
	MainPath string
}

// AnalyzeAndStitch analyzes a test and creates connections
func (s *SimpleStitcher) AnalyzeAndStitch(testFile string, content string) []Connection {
	s.connections = []Connection{}
	
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "exec ") {
			cmd := strings.TrimPrefix(line, "exec ")
			parts := strings.Fields(cmd)
			if len(parts) > 0 {
				program := parts[0]
				// Create a connection
				conn := Connection{
					TestFile: testFile,
					TestLine: i + 1,
					Program:  program,
					MainPath: fmt.Sprintf("cmd/%s/main.go:1", program),
				}
				s.connections = append(s.connections, conn)
			}
		}
	}
	
	return s.connections
}

// Demo shows the concept
func Demo() {
	stitcher := &SimpleStitcher{}
	
	testContent := `
# Test that calls mcpdiff
exec mcpdiff --help
stderr 'Usage'
exec echo "test"
`
	
	connections := stitcher.AnalyzeAndStitch("test.txt", testContent)
	
	fmt.Println("Standard callgraph would NOT show these connections:")
	for _, conn := range connections {
		fmt.Printf("  %s:%d -> %s (%s)\n", 
			conn.TestFile, conn.TestLine, conn.Program, conn.MainPath)
	}
	fmt.Println("\nOur testcallgraph creates these missing edges!")
}