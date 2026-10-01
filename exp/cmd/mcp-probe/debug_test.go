package main

import (
	"encoding/json"
	"testing"

	jsonrpc2 "github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

func TestJSONRPC2MessageTypes(t *testing.T) {
	req := jsonrpc2.Request{
		ID:     rpcID(1),
		Method: "test",
		Params: json.RawMessage(`{}`),
	}

	data, err := jsonrpc2.EncodeMessage(&req)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"jsonrpc":"2.0","id":1,"method":"test","params":{}}`; got != want {
		t.Fatalf("request = %s, want %s", got, want)
	}

	resp := jsonrpc2.Response{
		ID:     rpcID(1),
		Result: json.RawMessage(`{}`),
	}

	data, err = jsonrpc2.EncodeMessage(&resp)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"jsonrpc":"2.0","id":1,"result":{}}`; got != want {
		t.Fatalf("response = %s, want %s", got, want)
	}
}

func TestLineReaderParsing(t *testing.T) {
	msg, err := jsonrpc2.DecodeMessage([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, ok := msg.(*jsonrpc2.Response)
	if !ok {
		t.Fatalf("DecodeMessage returned %T, want *jsonrpc.Response", msg)
	}
	if !resp.ID.IsValid() {
		t.Fatal("response ID is invalid")
	}
}
