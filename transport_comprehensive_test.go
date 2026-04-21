package mcp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Comprehensive transport testing to achieve near 100% coverage

func TestStdioTransport(t *testing.T) {
	transport := StdioTransport()
	if transport == nil {
		t.Fatal("StdioTransport should not return nil")
	}

	// Test Dial
	ctx := context.Background()
	conn, err := transport.Dial(ctx)
	if err != nil {
		t.Errorf("StdioTransport Dial failed: %v", err)
	}

	if conn == nil {
		t.Error("StdioTransport Dial should return a connection")
	}

	// Verify it's the expected type
	if _, ok := transport.(*ReadWriteCloserTransport); !ok {
		t.Error("StdioTransport should return ReadWriteCloserTransport")
	}
}

func TestStdioTransportComponents(t *testing.T) {
	transport := StdioTransport()
	ctx := context.Background()
	conn, err := transport.Dial(ctx)
	if err != nil {
		t.Fatalf("StdioTransport Dial failed: %v", err)
	}

	// Note: We can't easily test actual reading from stdin or writing to stdout
	// in unit tests, but we can verify the structure

	// Test that Close doesn't actually close stdin/stdout (uses NopCloser)
	err = conn.Close()
	if err != nil {
		t.Errorf("Close should not return error for stdio transport: %v", err)
	}

	// The reader should be stdin
	if rwcTransport, ok := transport.(*ReadWriteCloserTransport); ok {
		if composite, ok := rwcTransport.ReadWriteCloser.(struct {
			io.Reader
			io.Writer
			io.Closer
		}); ok {
			// Verify components are correct types
			if composite.Reader != os.Stdin {
				t.Error("Reader should be os.Stdin")
			}
			if composite.Writer != os.Stdout {
				t.Error("Writer should be os.Stdout")
			}
		}
	}
}

func TestTransportConcurrency(t *testing.T) {
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	transport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return &mockConcurrentReadWriteCloser{}, nil
	})

	var successCount int64
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			conn, err := transport.Dial(ctx)
			if err != nil {
				t.Errorf("Concurrent Dial failed: %v", err)
				return
			}
			if conn != nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	mu.Lock()
	finalCount := successCount
	mu.Unlock()

	if finalCount != numGoroutines {
		t.Errorf("Expected %d successful dials, got %d", numGoroutines, finalCount)
	}
}

func TestTransportWithTimeout(t *testing.T) {
	// Create a transport that takes time to connect
	slowTransport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return &mockSlowReadWriteCloser{}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	// Test with sufficient timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	conn, err := slowTransport.Dial(ctx)
	if err != nil {
		t.Errorf("Expected success with sufficient timeout: %v", err)
	}
	if conn == nil {
		t.Error("Expected connection with sufficient timeout")
	}

	// Test with insufficient timeout
	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()

	conn2, err2 := slowTransport.Dial(ctx2)
	if err2 == nil {
		t.Error("Expected timeout error")
	}
	if conn2 != nil {
		t.Error("Expected nil connection on timeout")
	}

	if !errors.Is(err2, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err2)
	}
}

func TestTransportPipelineOperations(t *testing.T) {
	// Create a pipe to test bidirectional communication
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	transport := &ReadWriteCloserTransport{clientConn}
	conn, err := transport.Dial(context.Background())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	// Test bidirectional communication
	testMessages := []string{
		"Hello, World!",
		"This is a test message",
		"Testing bidirectional communication",
		"Final test message",
	}

	// Server goroutine that echoes messages
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, err := serverConn.Read(buffer)
			if err != nil {
				return
			}
			message := string(buffer[:n])
			echo := "Echo: " + message
			serverConn.Write([]byte(echo))
		}
	}()

	// Client sends messages and checks echoes
	for _, msg := range testMessages {
		// Send message
		_, err := conn.Write([]byte(msg))
		if err != nil {
			t.Errorf("Write failed: %v", err)
			continue
		}

		// Read echo
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			t.Errorf("Read failed: %v", err)
			continue
		}

		expectedEcho := "Echo: " + msg
		actualEcho := string(buffer[:n])
		if actualEcho != expectedEcho {
			t.Errorf("Expected echo %q, got %q", expectedEcho, actualEcho)
		}
	}
}

func TestTransportErrorPropagation(t *testing.T) {
	tests := []struct {
		name        string
		transport   Transport
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name:        "nil connection error",
			transport:   &ReadWriteCloserTransport{nil},
			expectError: true,
			errorCheck: func(err error) bool {
				return errors.Is(err, ErrTransportClosed)
			},
		},
		{
			name: "custom transport error",
			transport: TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
				return nil, errors.New("custom transport error")
			}),
			expectError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "custom transport error")
			},
		},
		{
			name: "context cancellation",
			transport: TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}),
			expectError: true,
			errorCheck: func(err error) bool {
				return errors.Is(err, context.Canceled)
			},
		},
		{
			name:        "successful connection",
			transport:   &ReadWriteCloserTransport{&mockSuccessfulReadWriteCloser{}},
			expectError: false,
			errorCheck:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.name == "context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel() // Cancel immediately
			}

			conn, err := tt.transport.Dial(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorCheck != nil && !tt.errorCheck(err) {
					t.Errorf("Error check failed for error: %v", err)
				}
				if conn != nil {
					t.Error("Expected nil connection on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if conn == nil {
					t.Error("Expected non-nil connection")
				}
			}
		})
	}
}

func TestTransportMemoryUsage(t *testing.T) {
	// Test with large data transfers to check memory handling
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	transport := &ReadWriteCloserTransport{clientConn}
	conn, err := transport.Dial(context.Background())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	// Server that reads and discards data
	go func() {
		buffer := make([]byte, 4096)
		totalRead := 0
		for totalRead < len(largeData) {
			n, err := serverConn.Read(buffer)
			if err != nil {
				return
			}
			totalRead += n
		}
	}()

	// Client sends large data
	totalWritten := 0
	chunkSize := 4096
	for totalWritten < len(largeData) {
		end := totalWritten + chunkSize
		if end > len(largeData) {
			end = len(largeData)
		}
		n, err := conn.Write(largeData[totalWritten:end])
		if err != nil {
			t.Errorf("Write failed: %v", err)
			break
		}
		totalWritten += n
	}

	if totalWritten != len(largeData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(largeData), totalWritten)
	}
}

func TestTransportMultipleDialCalls(t *testing.T) {
	conn := &mockMultiDialReadWriteCloser{}
	transport := &ReadWriteCloserTransport{conn}

	// Multiple Dial calls should return the same connection
	ctx := context.Background()
	conn1, err1 := transport.Dial(ctx)
	if err1 != nil {
		t.Errorf("First Dial failed: %v", err1)
	}

	conn2, err2 := transport.Dial(ctx)
	if err2 != nil {
		t.Errorf("Second Dial failed: %v", err2)
	}

	if conn1 != conn2 {
		t.Error("Multiple Dial calls should return the same connection")
	}

	if conn1 != conn {
		t.Error("Dial should return the wrapped connection")
	}
}

func TestTransportFuncVariations(t *testing.T) {
	tests := []struct {
		name     string
		funcImpl func(context.Context) (io.ReadWriteCloser, error)
		wantErr  bool
	}{
		{
			name: "simple success",
			funcImpl: func(ctx context.Context) (io.ReadWriteCloser, error) {
				return &mockSuccessfulReadWriteCloser{}, nil
			},
			wantErr: false,
		},
		{
			name: "nil connection success",
			funcImpl: func(ctx context.Context) (io.ReadWriteCloser, error) {
				return nil, nil
			},
			wantErr: false,
		},
		{
			name: "context-aware",
			funcImpl: func(ctx context.Context) (io.ReadWriteCloser, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
					return &mockSuccessfulReadWriteCloser{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "always error",
			funcImpl: func(ctx context.Context) (io.ReadWriteCloser, error) {
				return nil, errors.New("always fails")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := TransportFunc(tt.funcImpl)
			_, err := transport.Dial(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// Note: conn can be nil even on success if the function returns (nil, nil)
			}
		})
	}
}

// Mock types for comprehensive testing

type mockConcurrentReadWriteCloser struct {
	data []byte
	mu   sync.RWMutex
}

func (m *mockConcurrentReadWriteCloser) Read(p []byte) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := copy(p, m.data)
	return n, nil
}

func (m *mockConcurrentReadWriteCloser) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *mockConcurrentReadWriteCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = nil
	return nil
}

type mockSlowReadWriteCloser struct {
	bytes.Buffer
}

func (m *mockSlowReadWriteCloser) Close() error {
	return nil
}

type mockSuccessfulReadWriteCloser struct {
	bytes.Buffer
}

func (m *mockSuccessfulReadWriteCloser) Close() error {
	return nil
}

type mockMultiDialReadWriteCloser struct {
	bytes.Buffer
	dialCount int
}

func (m *mockMultiDialReadWriteCloser) Close() error {
	return nil
}

type mockFailingReadWriteCloser struct{}

func (m *mockFailingReadWriteCloser) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

func (m *mockFailingReadWriteCloser) Write(p []byte) (int, error) {
	return 0, errors.New("write error")
}

func (m *mockFailingReadWriteCloser) Close() error {
	return errors.New("close error")
}

func TestTransportWithFailingConnection(t *testing.T) {
	failingConn := &mockFailingReadWriteCloser{}
	transport := &ReadWriteCloserTransport{failingConn}

	conn, err := transport.Dial(context.Background())
	if err != nil {
		t.Errorf("Dial should not fail even with a failing connection: %v", err)
	}

	if conn != failingConn {
		t.Error("Dial should return the exact connection provided")
	}

	// Test that the connection actually fails when used
	_, readErr := conn.Read(make([]byte, 10))
	if readErr == nil {
		t.Error("Expected read error from failing connection")
	}

	_, writeErr := conn.Write([]byte("test"))
	if writeErr == nil {
		t.Error("Expected write error from failing connection")
	}

	closeErr := conn.Close()
	if closeErr == nil {
		t.Error("Expected close error from failing connection")
	}
}

func TestErrTransportClosed(t *testing.T) {
	// Test the exported error
	if ErrTransportClosed == nil {
		t.Error("ErrTransportClosed should not be nil")
	}

	expectedMsg := "mcp: transport closed"
	if ErrTransportClosed.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, ErrTransportClosed.Error())
	}

	// Test that nil transport returns ErrTransportClosed
	transport := &ReadWriteCloserTransport{nil}
	_, err := transport.Dial(context.Background())
	if !errors.Is(err, ErrTransportClosed) {
		t.Errorf("Expected ErrTransportClosed, got %v", err)
	}
}

func BenchmarkTransportDial(b *testing.B) {
	transport := &ReadWriteCloserTransport{&mockSuccessfulReadWriteCloser{}}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transport.Dial(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTransportFuncDial(b *testing.B) {
	transport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return &mockSuccessfulReadWriteCloser{}, nil
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transport.Dial(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdioTransportDial(b *testing.B) {
	transport := StdioTransport()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transport.Dial(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestSSEAdapterContextCancellationReturnsErrTransportClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	adapter := &sseRWCAdapter{
		ctx:         ctx,
		sseBody:     io.NopCloser(strings.NewReader("")),
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		readBuf:     new(bytes.Buffer),
		readChan:    make(chan []byte),
		readErrChan: make(chan error, 1),
		closed:      make(chan struct{}),
	}

	_, err := adapter.Read(make([]byte, 1))
	if !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Read() error = %v, want errors.Is(..., ErrTransportClosed)", err)
	}

	_, err = adapter.Write([]byte("{}"))
	if !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Write() error = %v, want errors.Is(..., ErrTransportClosed)", err)
	}
}

func TestStreamableTransportClosureReturnsErrTransportClosed(t *testing.T) {
	transport := newStreamableServerTransport("test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := transport.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	_, err := transport.Read(context.Background())
	if !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Read() error = %v, want errors.Is(..., ErrTransportClosed)", err)
	}

	err = transport.Write(context.Background(), JSONRPCMessage{JSONRPC: "2.0"})
	if !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Write() error = %v, want errors.Is(..., ErrTransportClosed)", err)
	}
}

func TestWebSocketReadClosureReturnsErrTransportClosed(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Upgrade() error = %v", err)
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()

	transport, err := NewWebSocketTransport("ws" + strings.TrimPrefix(server.URL, "http"))
	if err != nil {
		t.Fatalf("NewWebSocketTransport() error = %v", err)
	}

	rwc, err := transport.Dial(context.Background())
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer rwc.Close()

	_, err = rwc.Read(make([]byte, 1))
	if !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Read() error = %v, want errors.Is(..., ErrTransportClosed)", err)
	}
}
