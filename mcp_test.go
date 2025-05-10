package mcp

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
)

type testInput struct {
	Message string `json:"message"`
}

type testOutput struct {
	Result string `json:"result"`
}

// A very simple test that just confirms we can create a server and client
func TestServerClientCreation(t *testing.T) {
	// Create a new server
	server := NewServer("test-server", "1.0.0")

	if server.name != "test-server" {
		t.Errorf("Expected server name 'test-server', got %s", server.name)
	}

	// Version might be overwritten by default options that infer from build info
	// So don't check for exact version match

	// Create client and server connections via net.Pipe
	clientConn, serverConn := net.Pipe()

	// Create a transport for the client
	transport := &ReadWriteCloserTransport{clientConn}

	// Start the server in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// Create a transport that just returns our server connection
		serverTransport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
			return serverConn, nil
		})

		// This will start serving requests on the server side of the pipe
		server.Serve(ctx, serverTransport)
	}()

	// Create the client
	client, err := NewClient(transport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client is nil")
	}

	// Close connections to avoid resource leaks
	client.Close()
}

// Fake transport for testing that just returns predetermined responses
type fakeTransport struct {
	reader io.Reader
	writer *bytes.Buffer
}

func newFakeTransport(response string) *fakeTransport {
	return &fakeTransport{
		reader: bytes.NewBufferString(response),
		writer: &bytes.Buffer{},
	}
}

func (f *fakeTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return &fakeReadWriteCloser{
		reader: f.reader,
		writer: f.writer,
	}, nil
}

type fakeReadWriteCloser struct {
	reader io.Reader
	writer *bytes.Buffer
}

func (f *fakeReadWriteCloser) Read(p []byte) (n int, err error) {
	return f.reader.Read(p)
}

func (f *fakeReadWriteCloser) Write(p []byte) (n int, err error) {
	return f.writer.Write(p)
}

func (f *fakeReadWriteCloser) Close() error {
	return nil
}
