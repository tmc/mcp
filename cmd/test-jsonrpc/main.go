package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"golang.org/x/exp/jsonrpc2"
)

type mockHandler struct{}

func (h *mockHandler) Handle(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if req.Method == "test" {
		return map[string]string{"result": "ok"}, nil
	}
	return nil, fmt.Errorf("unknown method")
}

type pipeTransport struct {
	reader io.Reader
	writer io.Writer
}

func (p *pipeTransport) Read(b []byte) (n int, err error) {
	return p.reader.Read(b)
}

func (p *pipeTransport) Write(b []byte) (n int, err error) {
	return p.writer.Write(b)
}

func (p *pipeTransport) Close() error {
	return nil
}

// transportDialer adapts pipeTransport to jsonrpc2.Dialer
type transportDialer struct {
	transport *pipeTransport
}

func (td transportDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return td.transport, nil
}

func main() {
	// Create pipes
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	// Start server
	go func() {
		transport := &pipeTransport{
			reader: serverReader,
			writer: serverWriter,
		}

		// Create server connection using Dial API
		serverCtx := context.Background()
		conn, err := jsonrpc2.Dial(serverCtx,
			transportDialer{transport: transport},
			jsonrpc2.ConnectionOptions{
				Handler: jsonrpc2.HandlerFunc((&mockHandler{}).Handle),
			})
		if err != nil {
			log.Fatalf("Failed to create server connection: %v", err)
		}

		conn.Wait()
	}()

	// Create client
	transport := &pipeTransport{
		reader: clientReader,
		writer: clientWriter,
	}

	// Create client connection using Dial API
	clientCtx := context.Background()
	conn, err := jsonrpc2.Dial(clientCtx,
		transportDialer{transport: transport},
		jsonrpc2.ConnectionOptions{})
	if err != nil {
		log.Fatalf("Failed to create client connection: %v", err)
	}

	// Make a call
	var result map[string]string
	asyncCall := conn.Call(context.Background(), "test", nil)

	// Check the ID that was generated
	id := asyncCall.ID()
	fmt.Printf("Request ID: %s\n", id)

	err = asyncCall.Await(context.Background(), &result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result: %v\n", result)
}
