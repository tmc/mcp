package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Simple MCP server that handles initialize and responds to probe
func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()

		// Look for Content-Length header
		if strings.HasPrefix(line, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			length, err := strconv.Atoi(lengthStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing Content-Length: %v\n", err)
				continue
			}

			// Read empty line
			scanner.Scan()

			// Read the JSON payload
			payload := make([]byte, length)
			n, err := io.ReadFull(os.Stdin, payload)
			if err != nil || n != length {
				fmt.Fprintf(os.Stderr, "Error reading payload: %v\n", err)
				continue
			}

			// Parse the request
			var req map[string]interface{}
			if err := json.Unmarshal(payload, &req); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
				continue
			}

			// Handle the request
			method := req["method"].(string)
			id := req["id"]

			switch method {
			case "initialize":
				// Send initialize response
				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]interface{}{
						"protocolVersion": "2025-03-26",
						"serverInfo": map[string]interface{}{
							"name":    "test-server",
							"version": "1.0.0",
						},
						"capabilities": map[string]interface{}{
							"tools": map[string]interface{}{
								"listChanged": false,
							},
						},
					},
				}
				sendResponse(response)

			case "ping":
				// Send ping response
				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      id,
					"result":  map[string]interface{}{},
				}
				sendResponse(response)

			case "tools/list":
				// Send tools list
				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]interface{}{
						"tools": []map[string]interface{}{
							{
								"name":        "test-tool",
								"description": "A test tool",
							},
						},
					},
				}
				sendResponse(response)

			default:
				// Send error for unsupported methods
				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      id,
					"error": map[string]interface{}{
						"code":    -32601,
						"message": "Method not found",
					},
				}
				sendResponse(response)
			}
		}
	}
}

func sendResponse(response map[string]interface{}) {
	data, err := json.Marshal(response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Content-Length: %d\r\n\r\n%s", len(data), data)
	os.Stdout.Sync()
}
