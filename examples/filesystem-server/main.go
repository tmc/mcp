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

	// Create server
	server := mcp.NewServer("filesystem-server", "1.0.0")

	// Register read_file tool
	err := server.RegisterTool(mcp.NewTool(
		"read_file",
		"Read the complete contents of a file from the file system. "+
			"Handles various text encodings and provides detailed error messages "+
			"if the file cannot be read.",
		func(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
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
	))
	if err != nil {
		log.Fatalf("Failed to register tool: %v", err)
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
