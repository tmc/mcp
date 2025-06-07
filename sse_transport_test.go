package mcp_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/tmc/mcp"
)

// TestSSEClientTransport tests the SSE client transport
func TestSSEClientTransport(t *testing.T) {
	transport, err := mcp.NewSSEClientTransport("http://example.com/sse", slog.Default())
	if err != nil {
		t.Fatalf("failed to create SSE client transport: %v", err)
	}
	if transport == nil {
		t.Fatal("expected non-nil transport")
	}

	// Test dial - should fail with a test URL
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = transport.Dial(ctx)
	if err == nil {
		t.Error("expected error from dial with invalid URL")
	}
}

// TestSSEServerTransport tests the SSE server transport
func TestSSEServerTransport(t *testing.T) {
	// Create a mock ReadWriteCloser
	mockRWC := &mockReadWriteCloser{
		Buffer: bytes.NewBuffer(nil),
	}

	transport := mcp.NewSSEServerTransport(mockRWC, slog.Default())
	if transport == nil {
		t.Fatal("expected non-nil transport")
	}

	// Test dial - should return the RWC
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	rwc, err := transport.Dial(ctx)
	if err != nil {
		t.Errorf("unexpected error from server transport dial: %v", err)
	}
	if rwc != mockRWC {
		t.Error("expected dial to return the same RWC")
	}
}

// mockReadWriteCloser is a simple mock that satisfies io.ReadWriteCloser
type mockReadWriteCloser struct {
	*bytes.Buffer
	closed bool
}

func (m *mockReadWriteCloser) Close() error {
	m.closed = true
	return nil
}
