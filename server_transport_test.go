package mcp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
)

// mockReadWriteCloser is a mock implementation for testing
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

// TestSingleConnListener tests the singleConnListener
func TestSingleConnListenerInternal(t *testing.T) {
	mockConn := &mockReadWriteCloser{
		readData:    []byte{},
		writtenData: []byte{},
	}
	listener := &singleConnListener{
		conn:   mockConn,
		done:   make(chan struct{}),
		logger: testLogger(t),
	}

	ctx := context.Background()

	// First accept should return the connection
	conn, err := listener.Accept(ctx)
	if err != nil {
		t.Errorf("First Accept() error = %v", err)
	}
	if conn != mockConn {
		t.Error("Expected same connection")
	}

	// Second accept should return EOF
	conn2, err := listener.Accept(ctx)
	if err != io.EOF {
		t.Errorf("Second Accept() error = %v, want io.EOF", err)
	}
	if conn2 != nil {
		t.Error("Expected nil connection on second accept")
	}

	// Test Close
	err = listener.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Test Accept with nil connection
	listener2 := &singleConnListener{
		conn:   nil,
		done:   make(chan struct{}),
		logger: testLogger(t),
	}

	conn3, err := listener2.Accept(ctx)
	if err != io.EOF {
		t.Errorf("Accept() with nil conn error = %v, want io.EOF", err)
	}
	if conn3 != nil {
		t.Error("Expected nil connection")
	}
}

// TestTemporaryError tests the temporaryError type
func TestTemporaryErrorInternal(t *testing.T) {
	err := &temporaryError{msg: "test error"}

	if err.Error() != "test error" {
		t.Errorf("Error() = %s, want test error", err.Error())
	}

	if !err.Temporary() {
		t.Error("Temporary() = false, want true")
	}
}

// TestStdioTransport tests the StdioTransport function
func TestStdioTransportInternal(t *testing.T) {
	// StdioTransport() returns an io.ReadWriteCloser with stdin/stdout
	result := StdioTransport()
	if result == nil {
		t.Error("StdioTransport() should return a non-nil ReadWriteCloser")
	}
}

// TestDialer tests the Dialer interface
func TestDialerInternal(t *testing.T) {
	// Test a mock dialer
	mockDialer := &mockDialer{
		conn: &mockReadWriteCloser{
			readData:    []byte{},
			writtenData: []byte{},
		},
	}

	ctx := context.Background()
	conn, err := mockDialer.Dial(ctx)
	if err != nil {
		t.Errorf("Dial() error = %v", err)
	}
	if conn != mockDialer.conn {
		t.Error("Expected same connection")
	}

	// Test with error
	mockDialer.err = errors.New("dial error")
	conn2, err := mockDialer.Dial(ctx)
	if err == nil || err.Error() != "dial error" {
		t.Errorf("Dial() error = %v, want dial error", err)
	}
	if conn2 != nil {
		t.Error("Expected nil connection on error")
	}

	// Test server Dialer method doesn't exist, skip this test
}

// mockDialer for testing
type mockDialer struct {
	conn io.ReadWriteCloser
	err  error
}

func (m *mockDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.conn, nil
}

// testWriter adapts testing.T to io.Writer
type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return len(p), nil
}

// testLogger creates a logger for testing
func testLogger(t *testing.T) *slog.Logger {
	// Set default level to INFO unless MCP_TEST_DEBUG is set
	level := slog.LevelInfo
	if os.Getenv("MCP_TEST_DEBUG") == "1" {
		level = slog.LevelDebug
	}

	// Use slog's text handler that writes to the test log
	return slog.New(slog.NewTextHandler(&testWriter{t: t}, &slog.HandlerOptions{Level: level}))
}
