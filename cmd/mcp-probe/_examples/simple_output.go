//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"

	jsonrpc2 "github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

func printSampleRequests() {
	// Initialize request
	initReq := &jsonrpc2.Request{
		ID:     rpcID(1),
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
		ID:     rpcID(2),
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

func rpcID(n int64) jsonrpc2.ID {
	id, _ := jsonrpc2.MakeID(float64(n))
	return id
}
