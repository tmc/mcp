package testcallgraph

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnhancedStitcher connects test scripts to programs, including custom commands
type EnhancedStitcher struct {
	// Map of test files to programs they execute
	TestToProgramMap map[string][]ProgramExecution
	
	// Map of custom commands to the programs they execute
	CustomCommandMap map[string]CommandMapping
}

// ProgramExecution represents how a program is executed from a test
type ProgramExecution struct {
	Program     string
	Command     string   // The full command that executes it (exec, mcp-server-start, etc)
	Line        int
	ExecutedBy  string   // "exec" or custom command name
	IsServer    bool     // Whether this starts a long-running server process
}

// CommandMapping maps a custom command to the program it executes
type CommandMapping struct {
	ExecutedProgram string   // The actual program that gets executed
	IsServer        bool     // Whether this starts a long-running process
	CommandPattern  string   // Pattern to extract program from command args
}

// NewEnhancedStitcher creates a new enhanced stitcher
func NewEnhancedStitcher() *EnhancedStitcher {
	return &EnhancedStitcher{
		TestToProgramMap: make(map[string][]ProgramExecution),
		CustomCommandMap: makeDefaultCommandMap(),
	}
}

// makeDefaultCommandMap creates the default mapping of custom commands to programs
func makeDefaultCommandMap() map[string]CommandMapping {
	return map[string]CommandMapping{
		// MCP tool commands
		"mcp-replay": {ExecutedProgram: "mcp-replay", IsServer: false},
		"mcp-spy": {ExecutedProgram: "mcp-spy", IsServer: false},
		"mcp-start": {ExecutedProgram: "mcp-start", IsServer: true},
		"mcp-test": {ExecutedProgram: "mcp-test", IsServer: false},
		"mcp-verify": {ExecutedProgram: "mcp-verify", IsServer: false},
		"mcp-send": {ExecutedProgram: "mcp-send", IsServer: false},
		"mcp-recv": {ExecutedProgram: "mcp-recv", IsServer: false},
		"mcp-serve": {ExecutedProgram: "mcp-serve", IsServer: true},
		"mcp-scripttest-server": {ExecutedProgram: "mcp-scripttest-server", IsServer: true},
		"mcpspy": {ExecutedProgram: "mcp-spy", IsServer: false}, // Alias
		"mcpdiff": {ExecutedProgram: "mcpdiff", IsServer: false},
		"mcpcat": {ExecutedProgram: "mcpcat", IsServer: false},
		"mcp-sort": {ExecutedProgram: "mcp-sort", IsServer: false},
		"mcp-shadow": {ExecutedProgram: "mcp-shadow", IsServer: false},
		
		// Server management commands
		"mcp-server-start": {
			ExecutedProgram: "dynamic", // Depends on command args
			IsServer:        true,
			CommandPattern:  "extract-from-args", // Special handling needed
		},
		"mcp-server-send": {ExecutedProgram: "none", IsServer: false}, // Sends to existing server
		"mcp-server-stop": {ExecutedProgram: "none", IsServer: false}, // Stops existing server
		"mcp-server-output": {ExecutedProgram: "none", IsServer: false}, // Reads from existing server
		
		// Special case: server command might execute arbitrary programs
		"server": {
			ExecutedProgram: "dynamic",
			IsServer:        true,
			CommandPattern:  "extract-from-args",
		},
	}
}

// AnalyzeScriptTest analyzes a test file to find all program executions
func (s *EnhancedStitcher) AnalyzeScriptTest(testFile string) error {
	file, err := os.Open(testFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	executions := []ProgramExecution{}
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Check for exec commands
		if strings.HasPrefix(line, "exec ") {
			cmd := strings.TrimPrefix(line, "exec ")
			if prog := extractProgram(cmd); prog != "" {
				executions = append(executions, ProgramExecution{
					Program:    prog,
					Command:    line,
					Line:       lineNum,
					ExecutedBy: "exec",
					IsServer:   false,
				})
			}
		} else {
			// Check for custom commands
			parts := strings.Fields(line)
			if len(parts) > 0 {
				cmdName := parts[0]
				if mapping, ok := s.CustomCommandMap[cmdName]; ok {
					prog := s.extractProgramFromCustomCommand(cmdName, line, mapping)
					if prog != "" {
						executions = append(executions, ProgramExecution{
							Program:    prog,
							Command:    line,
							Line:       lineNum,
							ExecutedBy: cmdName,
							IsServer:   mapping.IsServer,
						})
					}
				}
			}
		}
	}
	
	s.TestToProgramMap[testFile] = executions
	return scanner.Err()
}

// extractProgram extracts the program name from an exec command
func extractProgram(cmd string) string {
	// Handle various forms:
	// - exec mcpdiff --help
	// - exec ./mcpdiff --help
	// - exec /usr/bin/mcpdiff --help
	// - exec -- mcpdiff --help
	
	// Remove leading -- if present
	cmd = strings.TrimPrefix(cmd, "-- ")
	
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}
	
	prog := parts[0]
	return filepath.Base(prog)
}

// extractProgramFromCustomCommand extracts the program from a custom command
func (s *EnhancedStitcher) extractProgramFromCustomCommand(cmdName, fullCmd string, mapping CommandMapping) string {
	switch mapping.ExecutedProgram {
	case "dynamic":
		// For commands like mcp-server-start and server, extract from args
		return s.extractProgramFromServerCommand(fullCmd)
	case "none":
		// Commands that don't execute new programs
		return ""
	default:
		// Direct mapping
		return mapping.ExecutedProgram
	}
}

// extractProgramFromServerCommand extracts program from server start commands
func (s *EnhancedStitcher) extractProgramFromServerCommand(cmd string) string {
	// Handle commands like:
	// - mcp-server-start myserver -- go run ./server/main.go
	// - server -- python mcp_server.py
	// - mcp-server-start test -- ./mcpd -- node server.js

	// Find the -- separator
	parts := strings.Split(cmd, " -- ")
	if len(parts) < 2 {
		return ""
	}

	// For commands with multiple --, we want the program right after the first --
	// e.g., "mcp-server-start test -- ./mcpd -- node server.js"
	// We want "./mcpd", not "node server.js"
	actualCmd := parts[1]

	// But check if there's another -- (nested command)
	if strings.Contains(actualCmd, " -- ") {
		// Get just the part before the next --
		actualCmd = strings.Split(actualCmd, " -- ")[0]
	}

	// Parse the actual command
	cmdParts := strings.Fields(actualCmd)
	if len(cmdParts) == 0 {
		return ""
	}

	// Handle common patterns
	prog := cmdParts[0]

	// If it starts with ./ or /, it's already a program path
	if strings.HasPrefix(prog, "./") || strings.HasPrefix(prog, "/") {
		return filepath.Base(prog)
	}

	// If it's "go run", "python", "node", etc., look for the next meaningful part
	switch prog {
	case "go":
		if len(cmdParts) > 1 && cmdParts[1] == "run" {
			// For go run, try to extract package name from path
			if len(cmdParts) > 2 {
				path := cmdParts[2]
				// Extract meaningful name from path
				if strings.Contains(path, "cmd/") {
					parts := strings.Split(path, "cmd/")
					if len(parts) > 1 {
						name := strings.TrimSuffix(parts[1], "/main.go")
						name = strings.TrimSuffix(name, "/")
						return name
					}
				}
				return filepath.Base(filepath.Dir(path))
			}
		}
	case "python", "python3", "node", "deno":
		if len(cmdParts) > 1 {
			return strings.TrimSuffix(filepath.Base(cmdParts[1]), filepath.Ext(cmdParts[1]))
		}
	}

	// Otherwise, use the program name directly
	return filepath.Base(prog)
}

// CreateCallGraphConnections creates connections for a test file
func (s *EnhancedStitcher) CreateCallGraphConnections(testFile string) []CallGraphEdge {
	executions := s.TestToProgramMap[testFile]
	edges := []CallGraphEdge{}
	
	for _, exec := range executions {
		edge := CallGraphEdge{
			From:        fmt.Sprintf("%s:%d", testFile, exec.Line),
			To:          fmt.Sprintf("cmd/%s/main.go:main", exec.Program),
			EdgeType:    exec.ExecutedBy,
			IsServer:    exec.IsServer,
			Command:     exec.Command,
		}
		edges = append(edges, edge)
	}
	
	return edges
}

// CallGraphEdge represents an edge in the enhanced call graph
type CallGraphEdge struct {
	From     string // test.txt:line
	To       string // program/main.go:main
	EdgeType string // "exec", "mcp-server-start", etc.
	IsServer bool   // Whether this starts a long-running process
	Command  string // The full command line
}

// String returns a string representation of the edge
func (e CallGraphEdge) String() string {
	serverStr := ""
	if e.IsServer {
		serverStr = " [SERVER]"
	}
	return fmt.Sprintf("%s -> %s (%s)%s", e.From, e.To, e.EdgeType, serverStr)
}