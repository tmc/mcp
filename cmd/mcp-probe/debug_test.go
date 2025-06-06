package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/tmc/mcp/jsonrpc2"
)

func TestJSONRPC2MessageTypes(t *testing.T) {
	// Test how Request is marshaled
	req := jsonrpc2.Request{
		ID:     jsonrpc2.Int64ID(1),
		Method: "test",
		Params: json.RawMessage(`{}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Request: %s\n", data)

	// Test how Response is marshaled
	resp := jsonrpc2.Response{
		ID:     jsonrpc2.Int64ID(1),
		Result: json.RawMessage(`{}`),
	}

	data, err = json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Response: %s\n", data)

	// Test message types
	var msg jsonrpc2.Message
	msg = &req
	fmt.Printf("Message type: %T\n", msg)
}

func TestLineReaderParsing(t *testing.T) {
	// Test parsing a response with empty ID
	response := `{"jsonrpc":"2.0","id":{},"result":{}}`

	var resp jsonrpc2.Response
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
	} else {
		fmt.Printf("Parsed response: %+v\n", resp)
		fmt.Printf("ID: %v\n", resp.ID)
	}

	// Test parsing a response with numeric ID
	response2 := `{"jsonrpc":"2.0","id":1,"result":{}}`
	var resp2 jsonrpc2.Response
	if err := json.Unmarshal([]byte(response2), &resp2); err != nil {
		fmt.Printf("Error parsing response2: %v\n", err)
	} else {
		fmt.Printf("Parsed response2: %+v\n", resp2)
		fmt.Printf("ID2: %v\n", resp2.ID)
	}
}
