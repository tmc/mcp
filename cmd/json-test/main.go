package main

import (
	"encoding/json"
	"fmt"

	"golang.org/x/exp/jsonrpc2"
)

func main() {
	// Test how jsonrpc2 handles IDs

	// Create an empty ID
	emptyID := jsonrpc2.ID{}
	fmt.Printf("Empty ID IsValid: %v\n", emptyID.IsValid())

	// Marshal it
	data, _ := json.Marshal(emptyID)
	fmt.Printf("Empty ID JSON: %s\n", string(data))

	// Create a numeric ID
	numID := jsonrpc2.Int64ID(1)
	fmt.Printf("Numeric ID IsValid: %v\n", numID.IsValid())

	// Marshal it
	data, _ = json.Marshal(numID)
	fmt.Printf("Numeric ID JSON: %s\n", string(data))

	// Create a request with empty ID
	req := jsonrpc2.Request{
		Method: "test",
		ID:     jsonrpc2.ID{},
	}

	// Marshal the request
	data, _ = json.Marshal(req)
	fmt.Printf("Request with empty ID: %s\n", string(data))

	// Now with numeric ID
	req.ID = jsonrpc2.Int64ID(1)
	data, _ = json.Marshal(req)
	fmt.Printf("Request with numeric ID: %s\n", string(data))

	// Test the actual structure
	fmt.Printf("\nInternal ID structure: %+v\n", emptyID)
	fmt.Printf("Valid ID structure: %+v\n", numID)
}
