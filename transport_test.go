package mcp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
)

func TestReadWriteCloserTransport(t *testing.T) {
	// Create a pipe for testing
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	transport := &ReadWriteCloserTransport{clientConn}

	// Test Dial
	conn, err := transport.Dial(context.Background())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	if conn != clientConn {
		t.Error("Dial should return the same connection")
	}
}

func TestReadWriteCloserTransportNilConnection(t *testing.T) {
	transport := &ReadWriteCloserTransport{nil}

	_, err := transport.Dial(context.Background())
	if err == nil {
		t.Error("Expected error for nil connection")
	}
}

func TestTransportFunc(t *testing.T) {
	conn := &testMockReadWriteCloser{}

	transport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return conn, nil
	})

	result, err := transport.Dial(context.Background())
	if err != nil {
		t.Fatalf("TransportFunc Dial failed: %v", err)
	}

	if result != conn {
		t.Error("TransportFunc should return the connection from the function")
	}
}

func TestTransportFuncError(t *testing.T) {
	expectedErr := errors.New("transport error")

	transport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return nil, expectedErr
	})

	_, err := transport.Dial(context.Background())
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestTransportFuncContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	transport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return &testMockReadWriteCloser{}, nil
		}
	})

	_, err := transport.Dial(ctx)
	if err == nil {
		t.Error("Expected cancellation error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestMultipleTransportTypes(t *testing.T) {
	tests := []struct {
		name      string
		transport Transport
		wantErr   bool
	}{
		{
			name: "ReadWriteCloserTransport",
			transport: &ReadWriteCloserTransport{
				ReadWriteCloser: &testMockReadWriteCloser{},
			},
			wantErr: false,
		},
		{
			name: "TransportFunc success",
			transport: TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
				return &testMockReadWriteCloser{}, nil
			}),
			wantErr: false,
		},
		{
			name: "TransportFunc error",
			transport: TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
				return nil, errors.New("test error")
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := tt.transport.Dial(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if conn == nil {
				t.Error("Expected connection but got nil")
			}
		})
	}
}

func TestTransportWithRealConnections(t *testing.T) {
	// Test with actual network connections
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// Test read/write operations
	transport := &ReadWriteCloserTransport{clientConn}
	conn, err := transport.Dial(context.Background())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	// Test writing to the connection
	testData := []byte("test message")
	go func() {
		conn.Write(testData)
	}()

	// Test reading from the other end
	buffer := make([]byte, len(testData))
	n, err := serverConn.Read(buffer)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Expected to read %d bytes, got %d", len(testData), n)
	}

	if string(buffer[:n]) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(buffer[:n]))
	}
}

func TestTransportInterfaceCompliance(t *testing.T) {
	// Test that our types implement the Transport interface
	var transport Transport

	// ReadWriteCloserTransport
	transport = &ReadWriteCloserTransport{&testMockReadWriteCloser{}}
	if transport == nil {
		t.Error("ReadWriteCloserTransport should implement Transport")
	}

	// TransportFunc
	transport = TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return nil, nil
	})
	if transport == nil {
		t.Error("TransportFunc should implement Transport")
	}
}

// Mock types for testing

type testMockReadWriteCloser struct {
	buf    bytes.Buffer
	closed bool
}

func (m *testMockReadWriteCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, errors.New("connection closed")
	}
	return m.buf.Read(p)
}

func (m *testMockReadWriteCloser) Write(p []byte) (n int, err error) {
	if m.closed {
		return 0, errors.New("connection closed")
	}
	return m.buf.Write(p)
}

func (m *testMockReadWriteCloser) Close() error {
	m.closed = true
	return nil
}

func TestMockReadWriteCloser(t *testing.T) {
	mock := &testMockReadWriteCloser{}

	// Test write
	data := []byte("test data")
	n, err := mock.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Test read
	buffer := make([]byte, len(data))
	n, err = mock.Read(buffer)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to read %d bytes, read %d", len(data), n)
	}
	if string(buffer) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(buffer))
	}

	// Test close
	err = mock.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Test operations after close
	_, err = mock.Write([]byte("after close"))
	if err == nil {
		t.Error("Expected error writing after close")
	}

	_, err = mock.Read(make([]byte, 10))
	if err == nil {
		t.Error("Expected error reading after close")
	}
}

func TestTransportErrorHandling(t *testing.T) {
	// Test various error conditions
	tests := []struct {
		name      string
		transport Transport
		wantErr   string
	}{
		{
			name:      "nil connection",
			transport: &ReadWriteCloserTransport{nil},
			wantErr:   "transport closed",
		},
		{
			name: "function returns error",
			transport: TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
				return nil, errors.New("custom error")
			}),
			wantErr: "custom error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.transport.Dial(context.Background())
			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestTransportDialMethodSignature(t *testing.T) {
	// Ensure the Dial method has the correct signature
	transport := &ReadWriteCloserTransport{&testMockReadWriteCloser{}}

	// This should compile and work
	ctx := context.Background()
	conn, err := transport.Dial(ctx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if conn == nil {
		t.Error("Expected connection, got nil")
	}

	// Verify the connection implements io.ReadWriteCloser
	var rwc io.ReadWriteCloser = conn
	if rwc == nil {
		t.Error("Connection should implement io.ReadWriteCloser")
	}
}
