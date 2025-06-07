package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

type DiffProxyServer struct{}

func NewDiffProxyServer() *DiffProxyServer {
	return &DiffProxyServer{}
}

func (s *DiffProxyServer) handleTextDiff(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	text1, ok1 := args["text1"].(string)
	text2, ok2 := args["text2"].(string)

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("text1 and text2 parameters are required and must be strings")
	}

	// Use system diff command
	cmd := exec.Command("diff", "-u", "/dev/stdin", "/dev/stdin")

	// Create temp input with both texts
	input := fmt.Sprintf("%s\n---SEPARATOR---\n%s", text1, text2)
	cmd.Stdin = strings.NewReader(input)

	// Better approach: use diff with temporary files
	return s.diffTexts(text1, text2, "text1", "text2")
}

func (s *DiffProxyServer) handleFileDiff(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	file1, ok1 := args["file1"].(string)
	file2, ok2 := args["file2"].(string)

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("file1 and file2 parameters are required and must be strings")
	}

	// Use diff command for files
	cmd := exec.Command("diff", "-u", file1, file2)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	// diff returns 1 when files differ, which is normal
	if err != nil && cmd.ProcessState.ExitCode() > 1 {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Error running diff: %v\n%s", err, stderr.String()),
				},
			},
			IsError: true,
		}, nil
	}

	diffOutput := stdout.String()
	if diffOutput == "" {
		diffOutput = "Files are identical"
	}

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: diffOutput,
			},
		},
	}, nil
}

func (s *DiffProxyServer) handleMCPDiff(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	file1, ok1 := args["file1"].(string)
	file2, ok2 := args["file2"].(string)

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("file1 and file2 parameters are required and must be strings")
	}

	// Use mcpdiff command
	cmd := exec.Command("mcpdiff", file1, file2)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Error running mcpdiff: %v\n%s", err, stderr.String()),
				},
			},
			IsError: true,
		}, nil
	}

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: stdout.String(),
			},
		},
	}, nil
}

func (s *DiffProxyServer) diffTexts(text1, text2, name1, name2 string) (*modelcontextprotocol.CallToolResult, error) {
	// Write texts to temp files for diff
	cmd1 := exec.Command("mktemp")
	var stdout1 bytes.Buffer
	cmd1.Stdout = &stdout1
	if err := cmd1.Run(); err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpFile1 := strings.TrimSpace(stdout1.String())

	cmd2 := exec.Command("mktemp")
	var stdout2 bytes.Buffer
	cmd2.Stdout = &stdout2
	if err := cmd2.Run(); err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpFile2 := strings.TrimSpace(stdout2.String())

	// Write content to temp files
	writeCmd1 := exec.Command("sh", "-c", fmt.Sprintf("cat > %s", tmpFile1))
	writeCmd1.Stdin = strings.NewReader(text1)
	if err := writeCmd1.Run(); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %v", err)
	}

	writeCmd2 := exec.Command("sh", "-c", fmt.Sprintf("cat > %s", tmpFile2))
	writeCmd2.Stdin = strings.NewReader(text2)
	if err := writeCmd2.Run(); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %v", err)
	}

	// Run diff
	cmd := exec.Command("diff", "-u", "--label", name1, "--label", name2, tmpFile1, tmpFile2)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Clean up temp files
	exec.Command("rm", tmpFile1).Run()
	exec.Command("rm", tmpFile2).Run()

	// diff returns 1 when files differ, which is normal
	if err != nil && cmd.ProcessState.ExitCode() > 1 {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Error running diff: %v\n%s", err, stderr.String()),
				},
			},
			IsError: true,
		}, nil
	}

	diffOutput := stdout.String()
	if diffOutput == "" {
		diffOutput = "Texts are identical"
	}

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: diffOutput,
			},
		},
	}, nil
}

func main() {
	server := NewDiffProxyServer()
	mcpServer := mcp.NewServer("mcp-diff-proxy-server", "1.0.0")

	// Add tools
	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "text_diff",
		Description: "Generate a unified diff between two text strings",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text1": map[string]interface{}{
					"type":        "string",
					"description": "First text to compare",
				},
				"text2": map[string]interface{}{
					"type":        "string",
					"description": "Second text to compare",
				},
			},
			"required": []string{"text1", "text2"},
		},
	})

	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "file_diff",
		Description: "Generate a unified diff between two files",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file1": map[string]interface{}{
					"type":        "string",
					"description": "Path to first file",
				},
				"file2": map[string]interface{}{
					"type":        "string",
					"description": "Path to second file",
				},
			},
			"required": []string{"file1", "file2"},
		},
	})

	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "mcp_diff",
		Description: "Generate a diff between two MCP trace files using mcpdiff",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file1": map[string]interface{}{
					"type":        "string",
					"description": "Path to first MCP trace file",
				},
				"file2": map[string]interface{}{
					"type":        "string",
					"description": "Path to second MCP trace file",
				},
			},
			"required": []string{"file1", "file2"},
		},
	})

	// Add tool handlers
	mcpServer.OnToolCall("text_diff", server.handleTextDiff)
	mcpServer.OnToolCall("file_diff", server.handleFileDiff)
	mcpServer.OnToolCall("mcp_diff", server.handleMCPDiff)

	if err := mcpServer.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
