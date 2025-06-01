package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	// Test parsing the server response
	response := `{"jsonrpc":"2.0","id":{},"result":{"protocolVersion":"2025-03-26","serverInfo":{"name":"mcp-time","version":"(devel)"},"capabilities":{},"instructions":"A simple time service that provides current time and conversions"}}`

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed response: %#v\n", resp)
	fmt.Printf("ID field: %#v\n", resp["id"])

	// Test the client
	client := `{"ID":{},"Method":"initialize","Params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"mcp-probe","version":"0.1.0"},"capabilities":{}}}`

	var req map[string]interface{}
	if err := json.Unmarshal([]byte(client), &req); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nParsed client request: %#v\n", req)
	fmt.Printf("ID field: %#v\n", req["ID"])

	// Check stdin reading
	fmt.Println("\nReading from stdin...")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("Got line: %s\n", line)

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			fmt.Printf("Parse error: %v\n", err)
		} else {
			fmt.Printf("Parsed: %#v\n", msg)
		}
	}
}
