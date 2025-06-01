package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	// Start the time server
	cmd := exec.Command("go", "run", "./examples/servers/mcp-time-server")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	// Show server stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "SERVER: %s\n", scanner.Text())
		}
	}()

	// Start server
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	// Send initialization message
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
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
	fmt.Printf("SENDING: %s\n", string(data))
	fmt.Fprintf(stdin, "%s\n", data)

	// Read response
	scanner := bufio.NewScanner(stdout)
	if scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("RECEIVED: %s\n", line)

		// Parse response
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			fmt.Printf("PARSE ERROR: %v\n", err)
		} else {
			fmt.Printf("RESPONSE ID: %v (type: %T)\n", resp["id"], resp["id"])
			if result, ok := resp["result"]; ok {
				fmt.Printf("RESULT: %v\n", result)
			}
		}
	} else {
		fmt.Println("NO RESPONSE")
	}

	cmd.Process.Kill()
}
