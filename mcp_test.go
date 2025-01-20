package mcp

import (
	"context"
	"encoding/json"
	"net"
	"testing"
)

type testTransport struct {
	conn net.Conn
	ctx  context.Context
}

func newTestTransport(conn net.Conn) Transport {
	return &testTransport{
		conn: conn,
		ctx:  context.Background(),
	}
}

func (t *testTransport) Read(p []byte) (n int, err error)  { return t.conn.Read(p) }
func (t *testTransport) Write(p []byte) (n int, err error) { return t.conn.Write(p) }
func (t *testTransport) Close() error                      { return t.conn.Close() }
func (t *testTransport) Context() context.Context          { return t.ctx }

func TestService(t *testing.T) {
	svc := NewService("test", "1.0.0")

	err := svc.RegisterTool(NewTool("test", "A test tool", func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: "test",
			}},
		}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer("test", "1.0.0")
	clientConn, serverConn := net.Pipe()

	go func() {
		for {
			msg := make([]byte, 4096)
			n, err := serverConn.Read(msg)
			if err != nil {
				return
			}
			resp, err := server.Handle(context.Background(), msg[:n])
			if err != nil {
				t.Error(err)
				return
			}
			_, err = serverConn.Write(append(resp, '\n'))
			if err != nil {
				t.Error(err)
				return
			}
		}
	}()

	c := NewClient("test-client", "1.0.0", newTestTransport(clientConn))

	reply, err := c.Initialize(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if reply.ProtocolVersion != ProtocolVersion {
		t.Errorf("got protocol version %q, want %q", reply.ProtocolVersion, ProtocolVersion)
	}

	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(tools.Tools) != 1 {
		t.Errorf("got %d tools, want 1", len(tools.Tools))
	}

	result, err := c.CallTool(context.Background(), "test", nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Content) != 1 || result.Content[0].Text != "test" {
		t.Errorf("got result %+v, want text=test", result)
	}
}
