package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func main() {
	// Run mcp-time-server and see what it outputs
	// go run ./examples/servers/mcp-time-server

	// Send initialization request
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1, // Try with a numeric ID
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"clientInfo": map[string]interface{}{
				"name":    "test",
				"version": "1.0",
			},
			"capabilities": map[string]interface{}{},
		},
	}

	data, _ := json.Marshal(req)
	fmt.Printf("Sending: %s\n", string(data))
	fmt.Println(string(data)) // Output to stdout

	// Read response
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(os.Stderr, "Response: %s\n", line)

			// Parse response
			var resp map[string]interface{}
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "ID: %v (type: %T)\n", resp["id"], resp["id"])
			}
		}
	}()

	time.Sleep(2 * time.Second)
}
