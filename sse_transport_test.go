package mcp_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/tmc/mcp"
)

// TestSSEClientTransport tests the SSE client transport
func TestSSEClientTransport(t *testing.T) {
	t.Parallel()
	
	// Use a discarding logger for test to avoid log output
	testLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	t.Run("create_transport", func(t *testing.T) {
		transport, err := mcp.NewSSEClientTransport("http://example.com/sse", testLogger)
		if err != nil {
			t.Fatalf("failed to create SSE client transport: %v", err)
		}
		if transport == nil {
			t.Fatal("expected non-nil transport")
		}
	})

	t.Run("invalid_url", func(t *testing.T) {
		_, err := mcp.NewSSEClientTransport("://invalid-url", testLogger)
		if err == nil {
			t.Error("expected error for invalid URL")
		}
	})

	t.Run("nil_logger", func(t *testing.T) {
		transport, err := mcp.NewSSEClientTransport("http://example.com/sse", nil)
		if err != nil {
			t.Fatalf("failed to create SSE client transport with nil logger: %v", err)
		}
		if transport == nil {
			t.Fatal("expected non-nil transport")
		}
	})

	t.Run("dial_network_failure", func(t *testing.T) {
		transport, err := mcp.NewSSEClientTransport("http://nonexistent.example/sse", testLogger)
		if err != nil {
			t.Fatalf("failed to create SSE client transport: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err = transport.Dial(ctx)
		if err == nil {
			t.Error("expected error from dial with invalid URL")
		}
	})
}

func TestSSEServerTransport(t *testing.T) {
	t.Parallel()
	
	// Use a discarding logger for test to avoid log output
	testLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	t.Run("create_and_dial", func(t *testing.T) {
		// Create a mock ReadWriteCloser
		mockRWC := &mockReadWriteCloser{
			readData:    []byte{},
			writtenData: []byte{},
			readIndex:   0,
			closed:      false,
		}

		transport := mcp.NewSSEServerTransport(mockRWC, testLogger)
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
	})

	t.Run("nil_logger", func(t *testing.T) {
		mockRWC := &mockReadWriteCloser{}
		transport := mcp.NewSSEServerTransport(mockRWC, nil)
		if transport == nil {
			t.Fatal("expected non-nil transport even with nil logger")
		}
	})

	t.Run("nil_rwc", func(t *testing.T) {
		transport := mcp.NewSSEServerTransport(nil, testLogger)
		
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := transport.Dial(ctx)
		if err == nil {
			t.Error("expected error when dialing with nil RWC")
		}
	})
}

// mockReadWriteCloser implements io.ReadWriteCloser for testing
type mockReadWriteCloser struct {
	readData    []byte
	writtenData []byte
	readIndex   int
	closed      bool
}

func (m *mockReadWriteCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.EOF
	}
	if m.readIndex >= len(m.readData) {
		return 0, io.EOF
	}
	n = copy(p, m.readData[m.readIndex:])
	m.readIndex += n
	return n, nil
}

func (m *mockReadWriteCloser) Write(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	m.writtenData = append(m.writtenData, p...)
	return len(p), nil
}

func (m *mockReadWriteCloser) Close() error {
	m.closed = true
	return nil
}
