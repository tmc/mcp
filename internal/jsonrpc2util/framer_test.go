package jsonrpc2util_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tmc/mcp/internal/jsonrpc2util"
	"golang.org/x/exp/jsonrpc2"
)

// Test the WorkingFramer behavior
func TestWorkingFramer(t *testing.T) {
	framer := &jsonrpc2util.WorkingFramer{}

	// Create test message
	msg, err := jsonrpc2.NewCall(jsonrpc2.Int64ID(1), "initialize", json.RawMessage(`{"test": "data"}`))
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Test write and read
	rwc := &mockRWC{}
	writer := framer.Writer(rwc)
	reader := framer.Reader(rwc)

	// Write message
	n, err := writer.Write(context.Background(), msg)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	t.Logf("Wrote %d bytes", n)
	t.Logf("Written data: %s", rwc.data)

	// Read message back
	readMsg, readBytes, err := reader.Read(context.Background())
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	t.Logf("Read %d bytes", readBytes)

	// Check if it's a request
	if req, ok := readMsg.(*jsonrpc2.Request); ok {
		t.Logf("Read request: method=%s, id=%v", req.Method, req.ID)
		if req.Method != "initialize" {
			t.Errorf("Expected method 'initialize', got '%s'", req.Method)
		}
	} else {
		t.Errorf("Expected request, got %T", readMsg)
	}
}

// Mock ReadWriteCloser for testing
type mockRWC struct {
	data []byte
	pos  int
}

func (m *mockRWC) Read(p []byte) (n int, err error) {
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockRWC) Write(p []byte) (n int, err error) {
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *mockRWC) Close() error {
	return nil
}
