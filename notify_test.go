package mcp

import (
    "encoding/json"
    "testing"

    "github.com/google/go-cmp/cmp"
)

func TestListChangeNotifications(t *testing.T) {
    tests := []struct {
        name     string
        method   string
        caps     Capabilities
        wantSent bool
    }{
        {
            name:   "tools list changed with capability",
            method: MethodToolListChanged,
            caps: Capabilities{
                Tools: &struct {
                    ListChanged bool `json:"listChanged,omitempty"`
                }{ListChanged: true},
            },
            wantSent: true,
        },
        {
            name:   "tools list changed without capability",
            method: MethodToolListChanged,
            caps:   Capabilities{},
            wantSent: false,
        },
        {
            name:   "prompts list changed with capability",
            method: MethodPromptListChanged,
            caps: Capabilities{
                Prompts: &struct {
                    ListChanged bool `json:"listChanged,omitempty"`
                }{ListChanged: true},
            },
            wantSent: true,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := NewService(ServiceOptions{
                Name:    "test",
                Version: "1.0.0",
                Capabilities: tt.caps,
            })

            var notified bool
            svc.Handle(tt.method, func(method string, params json.RawMessage) error {
                notified = true
                return nil
            })

            err := svc.NotifyListChanged(tt.method)
            if err != nil {
                t.Fatal(err)
            }

            if notified != tt.wantSent {
                t.Errorf("notification sent = %v, want %v", notified, tt.wantSent)
            }
        })
    }
}

func TestCapabilityNegotiation(t *testing.T) {
    clientCaps := ClientCapabilities{
        Roots: &struct {
            ListChanged bool `json:"listChanged,omitempty"`
        }{ListChanged: true},
    }

    serverCaps := Capabilities{
        Tools: &struct {
            ListChanged bool `json:"listChanged,omitempty"`
        }{ListChanged: true},
        Resources: &struct {
            ListChanged bool `json:"listChanged,omitempty"`
        }{ListChanged: true},
    }

    svc := NewService(ServiceOptions{
        Name:         "test",
        Version:      "1.0.0",
        Capabilities: serverCaps,
    })

    // Set up client and server
    server := NewServer(svc)
    clientConn, serverConn := net.Pipe()
    go server.ServeConn(serverConn)

    client := NewClient(clientConn)
    defer client.Close()

    // Initialize with capabilities
    reply, err := client.Initialize(context.Background(), Implementation{
        Name:    "test-client",
        Version: "1.0.0",
    }, clientCaps)
    if err != nil {
        t.Fatal(err)
    }

    // Verify server capabilities
    if !reply.Capabilities.Tools.ListChanged {
        t.Error("server should support tool list change notifications")
    }
    if !reply.Capabilities.Resources.ListChanged {
        t.Error("server should support resource list change notifications")
    }

    // Test notification flow
    var gotToolNotification bool
    client.Handle(MethodToolListChanged, func(method string, params json.RawMessage) error {
        gotToolNotification = true
        return nil
    })

    // Register a tool which should trigger a notification
    err = svc.RegisterTool(Tool{
        Name: "test",
        Handler: func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
            return &ToolResult{}, nil
        },
    })
    if err != nil {
        t.Fatal(err)
    }

    // Wait a bit for notification
    time.Sleep(100 * time.Millisecond)
    if !gotToolNotification {
        t.Error("did not receive tool list change notification")
    }
}