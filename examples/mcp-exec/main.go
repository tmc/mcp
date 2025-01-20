package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"os/exec"
	"time"

	"github.com/tmc/mcp"
)

func main() {
	// Define and parse command-line flags
	var (
		name    = flag.String("name", "exec-server", "name of the server")
		version = flag.String("version", "0.1.0", "version of the server")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Create a new MCP service
	svc := mcp.NewService(*name, *version)

	// Register the exec tool
	err := svc.RegisterTool(NewExecTool())
	if err != nil {
		log.Fatalf("Failed to register exec tool: %v", err)
	}

	// Create a new RPC server and register the service
	server := rpc.NewServer()
	err = server.RegisterName("MCP", svc)
	if err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}

	// Serve on stdin/stdout
	transport := mcp.NewStdioTransport(context.Background())
	server.ServeConn(transport)
}

// ExecTool implements a Tool that executes shell commands.
type ExecTool struct {
	name        string
	description string
}

// NewExecTool creates a new ExecTool instance.
func NewExecTool() *ExecTool {
	return &ExecTool{
		name:        "exec",
		description: "Executes a shell command and returns the output.",
	}
}

// Name returns the name of the tool.
func (t *ExecTool) Name() string {
	return t.name
}

// Description returns the description of the tool.
func (t *ExecTool) Description() string {
	return t.description
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *ExecTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {"type": "string"},
			"args": {"type": "array", "items": {"type": "string"}},
			"timeout": {"type": "number", "description": "Timeout in seconds", "default": 60}
		},
		"required": ["command"]
	}`)
}

// Handler executes the shell command and returns the output.
func (t *ExecTool) Handler(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
	var params struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Timeout *float64 `json:"timeout"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{{
				Type: "text",
				Text: fmt.Sprintf("Error: invalid arguments: %v", err),
			}},
			IsError: true,
		}, nil
	}

	if params.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	timeout := 60 * time.Second
	if params.Timeout != nil {
		timeout = time.Duration(*params.Timeout * float64(time.Second))
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, params.Command, params.Args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{{
				Type: "text",
				Text: fmt.Sprintf("Error: %v\nOutput: %s", err, output),
			}},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{{
			Type: "text",
			Text: string(output),
		}},
	}, nil
}
