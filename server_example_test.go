// Copyright 2025 The MCP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/tmc/mcp"
)

func ExampleServer() {
	ctx := context.Background()
	// Create in-memory connection using pipes
	clientConn, serverConn := net.Pipe()
	clientTransport := &mcp.ReadWriteCloserTransport{ReadWriteCloser: clientConn}
	serverTransport := &mcp.ReadWriteCloserTransport{ReadWriteCloser: serverConn}

	server := mcp.NewServer("greeter", "v0.0.1")
	server.RegisterTool(mcp.Tool{
		Name:        "greet",
		Description: "say hi",
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []any{mcp.TextContent{Type: "text", Text: "Hi " + params.Name}},
		}, nil
	})

	go func() {
		// Serve will return an error when the connection closes, which is expected
		_ = server.Serve(ctx, serverTransport)
	}()

	client, err := mcp.NewClient(clientTransport)
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.Initialize(ctx, mcp.InitializeRequest{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo:      mcp.Implementation{Name: "client", Version: "v0.0.1"},
	})
	if err != nil {
		log.Fatal(err)
	}

	args, _ := json.Marshal(map[string]string{"name": "user"})
	res, err := client.CallTool(ctx, mcp.CallToolRequest{Name: "greet", Arguments: args})
	if err != nil {
		log.Fatal(err)
	}
	// Content is returned as a map, not a typed struct
	if content, ok := res.Content[0].(map[string]interface{}); ok {
		if text, ok := content["text"].(string); ok {
			fmt.Println(text)
		}
	}

	client.Close()

	// Output: Hi user
}
