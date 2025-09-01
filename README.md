# mcp

Package mcp is a Go implementation of the Model Context Protocol: client,
server, and transports.

```
go get github.com/tmc/mcp
```

## Server

```go
server := mcp.NewServer("my-server", "1.0.0")

server.RegisterTool(mcp.Tool{
	Name:        "calculate",
	Description: "Perform calculations",
}, handler)

server.Serve(ctx, transport)
```

## Client

```go
client, err := mcp.NewClient(transport)
if err != nil {
	log.Fatal(err)
}
if err := client.Initialize(ctx); err != nil {
	log.Fatal(err)
}
result, err := client.CallTool(ctx, "calculate", args)
```

## Transports

The Transport interface is a single method:

```go
type Transport interface {
	Dial(context.Context) (io.ReadWriteCloser, error)
}
```

Four implementations are provided: stdio (default), SSE, streamable HTTP, and
websocket. All speak JSON-RPC 2.0.

## Commands

The cmd directory holds two stabilized tools:

	mcp        umbrella CLI
	mcp-probe  inspect a server's capabilities

Additional development tools live under exp/cmd while their interfaces settle.

## Examples

Runnable servers are in examples/servers — echo, time, filesystem, sqlite, and
more:

	go run ./examples/servers/mcp-time-server

## Documentation

Package documentation is at https://pkg.go.dev/github.com/tmc/mcp.

## License

MIT. See LICENSE.
