/*
Package mcp implements the Model Context Protocol (MCP).

Notifications

The MCP protocol supports various types of notifications that can be sent between
client and server. Notifications are capability-based, meaning both sides must
agree to support specific notification types during initialization.

Server Capabilities:
  - Tool list changes (tools.listChanged)
  - Resource list changes (resources.listChanged)
  - Progress updates
  - Logging messages

Client Capabilities:
  - Root list changes (roots.listChanged)

Example server with notifications:

    svc := mcp.NewService("example", "1.0.0")

    // Handle logging notifications
    svc.Handle(mcp.MethodLogging, func(method string, params json.RawMessage) error {
        var msg struct {
            Level  mcp.LoggingLevel `json:"level"`
            Logger string           `json:"logger"`
            Data   interface{}      `json:"data"`
        }
        if err := json.Unmarshal(params, &msg); err != nil {
            return err
        }
        log.Printf("[%s] %s: %v\n", msg.Level, msg.Logger, msg.Data)
        return nil
    })

Example client with notifications:

    client := mcp.NewClient(conn)

    // Handle tool list changes
    client.Handle(mcp.MethodToolListChanged, func(method string, params json.RawMessage) error {
        log.Println("Tool list changed")
        return nil
    })

    // Initialize with capabilities
    reply, err := client.Initialize(context.Background(), mcp.Implementation{
        Name:    "example-client",
        Version: "1.0.0",
    }, mcp.ClientCapabilities{
        Sampling: &struct{}{},
    })

For more examples, see the examples directory.
*/
package mcp