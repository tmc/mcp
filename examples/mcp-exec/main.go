package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
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

	// Create server
	server := mcp.NewServer(*name, *version)

	// Register the exec tool
	err := server.RegisterTool(NewExecTool())
	if err != nil {
		log.Fatalf("Failed to register exec tool: %v", err)
	}

	// Create transport
	transport := mcp.NewStdioTransport(context.Background())

	// Handle messages
	for {
		msg := make([]byte, 4096)
		n, err := transport.Read(msg)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Read error: %v", err)
			continue
		}

		resp, err := server.Handle(context.Background(), msg[:n])
		if err != nil {
			log.Printf("Handle error: %v", err)
			continue
		}

		_, err = transport.Write(append(resp, '\n'))
		if err != nil {
			log.Printf("Write error: %v", err)
			continue
		}
	}
}

// ExecTool implements a Tool that executes shell commands.
type ExecTool struct {
	name        string
	description string
}

// NewExecTool creates a new ExecTool instance.
func NewExecTool() mcp.Tool {
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
