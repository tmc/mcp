package mcp

import (
    "context"
    "encoding/json"
    "net"
    "testing"
)

func TestService(t *testing.T) {
    svc := NewService("test", "1.0.0")

    err := svc.RegisterTool(Tool{
        Name: "test",
        Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
            return &ToolResult{
                Content: []Content{{
                    Type: "text",
                    Text: "test",
                }},
            }, nil
        },
    })
    if err != nil {
        t.Fatal(err)
    }

    server := NewServer(svc)
    clientConn, serverConn := net.Pipe()

    go server.ServeConn(serverConn)

    c := NewClient(clientConn)
    defer c.Close()

    reply, err := c.Initialize(context.Background(), Implementation{
        Name:    "test-client",
        Version: "1.0.0",
    })
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

    if len(tools) != 1 {
        t.Errorf("got %d tools, want 1", len(tools))
    }

    result, err := c.CallTool(context.Background(), "test", nil)
    if err != nil {
        t.Fatal(err)
    }

    if len(result.Content) != 1 || result.Content[0].Text != "test" {
        t.Errorf("got result %+v, want text=test", result)
    }
}