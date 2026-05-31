package mcp

import (
	"bytes"
	"context"
	"testing"

	"golang.org/x/exp/jsonrpc2"
)

func TestLineFramerWrite(t *testing.T) {
	var buf bytes.Buffer
	w := LineFramer().Writer(&buf)
	msg, err := jsonrpc2.NewCall(jsonrpc2.Int64ID(1), "ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write(context.Background(), msg)
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if got == "" || got[len(got)-1] != '\n' {
		t.Fatalf("got %q, want newline-terminated JSON", got)
	}
}

func TestLineFramerRead(t *testing.T) {
	data := []byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n")
	r := LineFramer().Reader(bytes.NewReader(data))
	msg, _, err := r.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	resp, ok := msg.(*jsonrpc2.Response)
	if !ok {
		t.Fatalf("msg=%T", msg)
	}
	if !resp.ID.IsValid() {
		t.Fatal("invalid response id")
	}
}

func TestLineFramerReadRawValue(t *testing.T) {
	data := []byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}")
	r := LineFramer().Reader(bytes.NewReader(data))
	msg, _, err := r.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := msg.(*jsonrpc2.Response); !ok {
		t.Fatalf("msg=%T", msg)
	}
}
