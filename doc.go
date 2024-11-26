/*
Package mcp implements the Model Context Protocol (MCP).

MCP is a protocol for AI model interaction that enables structured communication
between AI models and external tools/resources. This implementation follows the
specification at https://github.com/modelcontextprotocol/specification.

Basic usage:

    // Create a service
    svc := mcp.NewService("example", "1.0.0")

    // Register a tool
    svc.RegisterTool(mcp.Tool{
        Name: "echo",
        Description: "Echo the input",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "message": map[string]any{"type": "string"},
            },
        },
        Handler: func(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
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
        },
    })

    // Create server and serve connections
    server := mcp.NewServer(svc)
    listener, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Fatal(err)
    }
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("accept error: %v", err)
            continue
        }
        go server.ServeConn(conn)
    }

The client can then connect and use the tools:

    conn, err := net.Dial("tcp", ":8080")
    if err != nil {
        log.Fatal(err)
    }
    client := mcp.NewClient(conn)
    defer client.Close()

    // Initialize
    _, err = client.Initialize(context.Background(), mcp.Implementation{
        Name:    "example-client",
        Version: "1.0.0",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Call tool
    result, err := client.CallTool(context.Background(), "echo", map[string]string{
        "message": "Hello, World!",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result.Content[0].Text) // Prints: Hello, World!

For more examples, see the examples directory.
*/
package mcp
