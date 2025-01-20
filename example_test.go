package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/tmc/mcp"
)

type testTransport struct {
	conn net.Conn
	ctx  context.Context
}

func (t *testTransport) Read(p []byte) (n int, err error)  { return t.conn.Read(p) }
func (t *testTransport) Write(p []byte) (n int, err error) { return t.conn.Write(p) }
func (t *testTransport) Close() error                      { return t.conn.Close() }
func (t *testTransport) Context() context.Context          { return t.ctx }

func Example() {
	// Create a server
	server := mcp.NewServer("example", "1.0.0")

	// Register a tool
	err := server.RegisterTool(mcp.NewTool("echo", "Echo the input", func(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
		var params struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, err
		}
		return &mcp.ToolResult{
			Content: []mcp.Content{{
				Type: "text",
				Text: params.Message,
			}},
		}, nil
	}))
	if err != nil {
		log.Fatal(err)
	}

	// Set up connection (example uses pipe)
	clientConn, serverConn := net.Pipe()

	// Serve in background
	go func() {
		for {
			msg := make([]byte, 4096)
			n, err := serverConn.Read(msg)
			if err != nil {
				return
			}
			resp, err := server.Handle(context.Background(), msg[:n])
			if err != nil {
				log.Printf("Server error: %v", err)
				return
			}
			_, err = serverConn.Write(append(resp, '\n'))
			if err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		}
	}()

	// Create client
	c := mcp.NewClient("example-client", "1.0.0", &testTransport{
		conn: clientConn,
		ctx:  context.Background(),
	})

	// Initialize
	_, err = c.Initialize(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Call tool
	args, _ := json.Marshal(map[string]string{
		"message": "Hello, World!",
	})
	result, err := c.CallTool(context.Background(), "echo", args)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Content[0].Text)
	// Output: Hello, World!
}
