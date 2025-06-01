package toolgraph

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ScriptTestAnalyzer analyzes scripttest files for tool dependencies
type ScriptTestAnalyzer struct {
	builder *GraphBuilder
}

// NewScriptTestAnalyzer creates a new scripttest analyzer
func NewScriptTestAnalyzer(builder *GraphBuilder) *ScriptTestAnalyzer {
	return &ScriptTestAnalyzer{
		builder: builder,
	}
}

// AnalyzeScriptTest analyzes a scripttest file
func (a *ScriptTestAnalyzer) AnalyzeScriptTest(path string, content string, depth int) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	
	// Regular expressions for pattern matching
	execRe := regexp.MustCompile(`^exec\s+(\S+)(.*)`)
	stdinRe := regexp.MustCompile(`^stdin\s+(.*)`)
	stdoutRe := regexp.MustCompile(`^stdout\s+(.*)`)
	stderrRe := regexp.MustCompile(`^stderr\s+(.*)`)
	
	// Track current command context
	var currentCmd string
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		
		// Skip comments and empty lines
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		
		// Process exec commands
		if matches := execRe.FindStringSubmatch(trimmed); matches != nil {
			cmd := matches[1]
			args := matches[2]
			currentCmd = cmd
			
			// Create command node
			cmdNode := &Node{
				ID:    fmt.Sprintf("%s:cmd:%d:%s", path, lineNum, cmd),
				Type:  NodeTypeCommand,
				Label: cmd,
				Metadata: map[string]interface{}{
					"line": lineNum,
					"args": strings.TrimSpace(args),
				},
			}
			a.builder.graph.Nodes[cmdNode.ID] = cmdNode
			a.builder.addEdge(path, cmdNode.ID, "executes", fmt.Sprintf("line %d", lineNum))
			
			// Analyze the command
			a.analyzeCommand(cmdNode, cmd, args, depth)
		}
		
		// Process stdin with current command context
		if matches := stdinRe.FindStringSubmatch(trimmed); matches != nil {
			input := matches[1]
			if currentCmd != "" {
				a.analyzeInput(path, currentCmd, input, lineNum, depth)
			}
		}
		
		// Process stdout/stderr for tool outputs
		if matches := stdoutRe.FindStringSubmatch(trimmed); matches != nil {
			output := matches[1]
			a.analyzeOutput(path, currentCmd, output, lineNum, "stdout", depth)
		}
		
		if matches := stderrRe.FindStringSubmatch(trimmed); matches != nil {
			output := matches[1]
			a.analyzeOutput(path, currentCmd, output, lineNum, "stderr", depth)
		}
	}
	
	return scanner.Err()
}

func (a *ScriptTestAnalyzer) analyzeCommand(cmdNode *Node, cmd string, args string, depth int) {
	// Identify MCP tools
	if strings.HasPrefix(cmd, "mcp-") {
		toolNode := &Node{
			ID:    "tool:" + cmd,
			Type:  NodeTypeTool,
			Label: cmd,
			Metadata: map[string]interface{}{
				"mcp": true,
			},
		}
		a.builder.graph.Nodes[toolNode.ID] = toolNode
		a.builder.addEdge(cmdNode.ID, toolNode.ID, "invokes", "")
		
		// Parse tool subcommands
		if strings.Contains(args, "call") {
			a.parseToolCall(cmdNode, args, depth)
		}
	}
	
	// Identify file operations
	if isFileCommand(cmd) {
		a.analyzeFileOperation(cmdNode, cmd, args, depth)
	}
}

func (a *ScriptTestAnalyzer) parseToolCall(cmdNode *Node, args string, depth int) {
	// Extract tool name from call command
	// Example: mcp-tool call add '{"x": 1, "y": 2}'
	parts := strings.Fields(args)
	for i, part := range parts {
		if part == "call" && i+1 < len(parts) {
			toolName := parts[i+1]
			callNode := &Node{
				ID:    fmt.Sprintf("%s:call:%s", cmdNode.ID, toolName),
				Type:  NodeTypeTool,
				Label: fmt.Sprintf("call %s", toolName),
				Metadata: map[string]interface{}{
					"operation": toolName,
				},
			}
			a.builder.graph.Nodes[callNode.ID] = callNode
			a.builder.addEdge(cmdNode.ID, callNode.ID, "calls", toolName)
		}
	}
}

func (a *ScriptTestAnalyzer) analyzeInput(path string, cmd string, input string, line int, depth int) {
	// Look for JSON-RPC patterns
	if strings.Contains(input, "method") || strings.Contains(input, "jsonrpc") {
		a.parseJSONRPC(path, input, line, depth)
	}
	
	// Look for tool-specific patterns
	if strings.Contains(cmd, "mcp") {
		a.parseMCPInput(path, cmd, input, line, depth)
	}
}

func (a *ScriptTestAnalyzer) analyzeOutput(path string, cmd string, output string, line int, outputType string, depth int) {
	// Look for successful tool responses
	if strings.Contains(output, "result") || strings.Contains(output, "success") {
		resultNode := &Node{
			ID:    fmt.Sprintf("%s:result:%d", path, line),
			Type:  NodeTypeTool,
			Label: fmt.Sprintf("%s result", outputType),
			Metadata: map[string]interface{}{
				"line":   line,
				"type":   outputType,
				"output": output,
			},
		}
		a.builder.graph.Nodes[resultNode.ID] = resultNode
		a.builder.addEdge(path, resultNode.ID, "expects", outputType)
	}
}

func (a *ScriptTestAnalyzer) parseJSONRPC(path string, jsonStr string, line int, depth int) {
	// Extract method names from JSON-RPC
	methodRe := regexp.MustCompile(`"method"\s*:\s*"([^"]+)"`)
	if matches := methodRe.FindStringSubmatch(jsonStr); len(matches) > 1 {
		method := matches[1]
		rpcNode := &Node{
			ID:    fmt.Sprintf("%s:rpc:%d:%s", path, line, method),
			Type:  NodeTypeTool,
			Label: fmt.Sprintf("RPC: %s", method),
			Metadata: map[string]interface{}{
				"line":   line,
				"method": method,
				"rpc":    true,
			},
		}
		a.builder.graph.Nodes[rpcNode.ID] = rpcNode
		a.builder.addEdge(path, rpcNode.ID, "rpc_call", method)
	}
}

func (a *ScriptTestAnalyzer) parseMCPInput(path string, cmd string, input string, line int, depth int) {
	// Parse MCP-specific input patterns
	if strings.Contains(input, "{") && strings.Contains(input, "}") {
		// Try to extract tool/method names
		patterns := []string{
			`"tool"\s*:\s*"([^"]+)"`,
			`"name"\s*:\s*"([^"]+)"`,
			`"action"\s*:\s*"([^"]+)"`,
		}
		
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			if matches := re.FindStringSubmatch(input); len(matches) > 1 {
				toolName := matches[1]
				toolNode := &Node{
					ID:    fmt.Sprintf("%s:mcp:%d:%s", path, line, toolName),
					Type:  NodeTypeTool,
					Label: fmt.Sprintf("MCP: %s", toolName),
					Metadata: map[string]interface{}{
						"line": line,
						"tool": toolName,
					},
				}
				a.builder.graph.Nodes[toolNode.ID] = toolNode
				a.builder.addEdge(path, toolNode.ID, "mcp_operation", toolName)
			}
		}
	}
}

func (a *ScriptTestAnalyzer) analyzeFileOperation(cmdNode *Node, cmd string, args string, depth int) {
	// Extract file paths from command arguments
	files := extractFilePaths(args)
	
	for _, file := range files {
		fileNode := &Node{
			ID:    "file:" + file,
			Type:  NodeTypeFile,
			Label: filepath.Base(file),
			Path:  file,
			Metadata: map[string]interface{}{
				"operation": cmd,
			},
		}
		a.builder.graph.Nodes[fileNode.ID] = fileNode
		a.builder.addEdge(cmdNode.ID, fileNode.ID, getFileOperation(cmd), "")
	}
}

// Helper functions

func isFileCommand(cmd string) bool {
	fileCommands := []string{
		"cat", "ls", "cp", "mv", "rm", "mkdir", "touch",
		"grep", "sed", "awk", "find", "test", "stat",
	}
	
	for _, fc := range fileCommands {
		if cmd == fc {
			return true
		}
	}
	return false
}

func extractFilePaths(args string) []string {
	// Simple file path extraction
	// In production, use more sophisticated parsing
	paths := []string{}
	parts := strings.Fields(args)
	
	for _, part := range parts {
		// Skip flags
		if strings.HasPrefix(part, "-") {
			continue
		}
		
		// Check if it looks like a path
		if strings.Contains(part, "/") || strings.Contains(part, ".") {
			paths = append(paths, part)
		}
	}
	
	return paths
}

func getFileOperation(cmd string) string {
	switch cmd {
	case "cat", "less", "more":
		return "reads"
	case "cp":
		return "copies"
	case "mv":
		return "moves"
	case "rm":
		return "removes"
	case "mkdir":
		return "creates"
	case "touch":
		return "touches"
	case "grep":
		return "searches"
	default:
		return "uses"
	}
}