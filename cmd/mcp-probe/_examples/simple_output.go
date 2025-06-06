//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"

	"golang.org/x/exp/jsonrpc2"
)

func printSampleRequests() {
	// Initialize request
	initReq := &jsonrpc2.Request{
		ID:     jsonrpc2.Int64ID(1),
		Method: "initialize",
		Params: json.RawMessage(`{
			"protocolVersion": "2025-03-26",
			"clientInfo": {
				"name": "mcp-probe",
				"version": "0.1.0"
			},
			"capabilities": {}
		}`),
	}

	initData, _ := json.Marshal(initReq)
	fmt.Println(string(initData))

	// Sample tool call
	toolReq := &jsonrpc2.Request{
		ID:     jsonrpc2.Int64ID(2),
		Method: "tools/call",
		Params: json.RawMessage(`{
			"name": "example-tool",
			"arguments": {
				"message": "Hello, MCP!"
			}
		}`),
	}

	toolData, _ := json.Marshal(toolReq)
	fmt.Println(string(toolData))
}
