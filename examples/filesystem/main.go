package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/tmc/mcp"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <allowed-directory> [additional-directories...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Create service
	svc := mcp.NewService("filesystem-server", "1.0.0")

	// Register read_file tool
	err := svc.RegisterTool(mcp.Tool{
		Name: "read_file",
		Description: "Read the complete contents of a file from the file system. " +
			"Handles various text encodings and provides detailed error messages " +
			"if the file cannot be read.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		},
		Handler: func(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
			var params struct {
				Path string `json:"path"`
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

			// TODO: Add path validation against allowed directories

			content, err := os.ReadFile(params.Path)
			if err != nil {
				return &mcp.ToolResult{
					Content: []mcp.Content{{
						Type: "text",
						Text: fmt.Sprintf("Error: failed to read file: %v", err),
					}},
					IsError: true,
				}, nil
			}

			return &mcp.ToolResult{
				Content: []mcp.Content{{
					Type: "text",
					Text: string(content),
				}},
			}, nil
		},
	})
	if err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}

	// Create server
	server := mcp.NewServer(svc)

	// Serve on stdin/stdout
	server.ServeConn(struct {
		io.Reader
		io.Writer
		io.Closer
	}{
		Reader: os.Stdin,
		Writer: os.Stdout,
		Closer: nopCloser{},
	})
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }
