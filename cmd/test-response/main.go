package main

import (
	"encoding/json"
	"fmt"

	"golang.org/x/exp/jsonrpc2"
)

func main() {
	// Test how jsonrpc2 marshals IDs
	id := jsonrpc2.Int64ID(1)
	data, _ := json.Marshal(id)
	fmt.Printf("Marshaled ID: %s\n", string(data))

	// Test unmarshaling
	var id2 jsonrpc2.ID
	json.Unmarshal([]byte("1"), &id2)
	fmt.Printf("Unmarshaled from 1: %+v\n", id2)

	// Test response
	resp := jsonrpc2.Response{
		ID:     jsonrpc2.Int64ID(1),
		Result: json.RawMessage(`{"test": "ok"}`),
	}
	respData, _ := json.Marshal(resp)
	fmt.Printf("Response: %s\n", string(respData))

	// Try unmarshaling server response
	serverResp := `{"jsonrpc":"2.0","id":1,"result":{"test":"ok"}}`
	var resp2 jsonrpc2.Response
	err := json.Unmarshal([]byte(serverResp), &resp2)
	fmt.Printf("Unmarshal server response: err=%v, id=%+v\n", err, resp2.ID)

	// Try with string ID
	serverResp2 := `{"jsonrpc":"2.0","id":"1","result":{"test":"ok"}}`
	var resp3 jsonrpc2.Response
	err = json.Unmarshal([]byte(serverResp2), &resp3)
	fmt.Printf("Unmarshal string ID: err=%v, id=%+v\n", err, resp3.ID)
}
