package testcallgraph

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ProgramExecution represents a program execution detected in a test
type ProgramExecution struct {
	Program    string // Program name (e.g., "mcpdiff", "server")
	Line       int    // Line number in test file
	Command    string // Full command line
	ExecutedBy string // How it's executed (e.g., "exec", "mcpdiff", "mcp-server-start")
	IsServer   bool   // Whether this is a server process
}

// CallGraphConnection represents a connection between test and program
type CallGraphConnection struct {
	TestFile string // Test file name
	TestLine int    // Line in test file
	Program  string // Program being called
	MainPath string // Path to main function
}

// CallGraphEdge represents an edge in the call graph
type CallGraphEdge struct {
	From     string // Source (e.g., "test:basic_test.txt:line5")
	To       string // Target (e.g., "cmd/mcpdiff/main.go:main")
	EdgeType string // Type of edge (e.g., "exec", "mcpdiff", "server")
	IsServer bool   // Whether target is a server process
}

func (e CallGraphEdge) String() string {
	serverMarker := ""
	if e.IsServer {
		serverMarker = " [SERVER]"
	}
	return fmt.Sprintf("%s -> %s (%s)%s", e.From, e.To, e.EdgeType, serverMarker)
}

// SimpleStitcher handles basic "exec" command analysis
type SimpleStitcher struct{}

// AnalyzeAndStitch analyzes a test file and creates connections for exec commands only
func (s *SimpleStitcher) AnalyzeAndStitch(filename, content string) []CallGraphConnection {
	var connections []CallGraphConnection
	lines := strings.Split(content, "\n")

	execRegex := regexp.MustCompile(`^\s*exec\s+(\S+)`)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if matches := execRegex.FindStringSubmatch(line); matches != nil {
			program := matches[1]
			connections = append(connections, CallGraphConnection{
				TestFile: filename,
				TestLine: i + 1,
				Program:  program,
				MainPath: fmt.Sprintf("cmd/%s/main.go:main", program),
			})
		}
	}

	return connections
}

// EnhancedStitcher handles all types of command analysis
type EnhancedStitcher struct {
	TestToProgramMap map[string][]ProgramExecution
}

// NewEnhancedStitcher creates a new enhanced stitcher
func NewEnhancedStitcher() *EnhancedStitcher {
	return &EnhancedStitcher{
		TestToProgramMap: make(map[string][]ProgramExecution),
	}
}

// AnalyzeScriptTest analyzes a scripttest file and extracts program executions
func (s *EnhancedStitcher) AnalyzeScriptTest(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var executions []ProgramExecution
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Regular expressions for different command patterns
	execRegex := regexp.MustCompile(`^\s*exec\s+(\S+)`)
	mcpCommandRegex := regexp.MustCompile(`^\s*(mcp-\S+|mcpdiff|mcpspy|mcpcat)\s*`)
	serverStartRegex := regexp.MustCompile(`^\s*mcp-server-start\s+.*--\s*(.+)`)
	goRunRegex := regexp.MustCompile(`go\s+run\s+(.+)`)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Handle exec commands
		if matches := execRegex.FindStringSubmatch(line); matches != nil {
			program := filepath.Base(matches[1])
			executions = append(executions, ProgramExecution{
				Program:    program,
				Line:       lineNum,
				Command:    line,
				ExecutedBy: "exec",
				IsServer:   false,
			})
			continue
		}

		// Handle MCP commands (mcpdiff, mcp-spy, etc.)
		if matches := mcpCommandRegex.FindStringSubmatch(line); matches != nil {
			program := matches[1]
			executions = append(executions, ProgramExecution{
				Program:    program,
				Line:       lineNum,
				Command:    line,
				ExecutedBy: program,
				IsServer:   false,
			})
			continue
		}

		// Handle server start commands
		if matches := serverStartRegex.FindStringSubmatch(line); matches != nil {
			serverCmd := strings.TrimSpace(matches[1])
			program := "server"

			// Extract actual program from go run commands
			if goMatches := goRunRegex.FindStringSubmatch(serverCmd); goMatches != nil {
				program = filepath.Base(filepath.Dir(goMatches[1]))
			}

			executions = append(executions, ProgramExecution{
				Program:    program,
				Line:       lineNum,
				Command:    line,
				ExecutedBy: "mcp-server-start",
				IsServer:   true,
			})
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	s.TestToProgramMap[filename] = executions
	return nil
}

// CreateCallGraphConnections creates call graph edges from analyzed programs
func (s *EnhancedStitcher) CreateCallGraphConnections(filename string) []CallGraphEdge {
	var edges []CallGraphEdge

	executions, exists := s.TestToProgramMap[filename]
	if !exists {
		return edges
	}

	baseFilename := filepath.Base(filename)

	for _, exec := range executions {
		from := fmt.Sprintf("test:%s:line%d", baseFilename, exec.Line)
		to := s.generateTargetPath(exec.Program)

		edges = append(edges, CallGraphEdge{
			From:     from,
			To:       to,
			EdgeType: exec.ExecutedBy,
			IsServer: exec.IsServer,
		})
	}

	return edges
}

// generateTargetPath generates the target path for a program
func (s *EnhancedStitcher) generateTargetPath(program string) string {
	// Special handling for common programs
	switch program {
	case "echo", "cat", "grep", "sed", "awk":
		return fmt.Sprintf("system:%s", program)
	case "go":
		return "system:go"
	default:
		// Assume it's an MCP tool in cmd/
		return fmt.Sprintf("cmd/%s/main.go:main", program)
	}
}
