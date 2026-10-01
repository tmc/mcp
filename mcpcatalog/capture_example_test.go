package mcpcatalog_test

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcpcatalog"
)

func ExampleCapture() {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "example", Version: "1"}, nil)
	a, b := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, a, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "example", Version: "1"}, nil)
	cs, err := client.Connect(ctx, b, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cs.Close()
	catalog, err := mcpcatalog.Capture(ctx, cs, mcpcatalog.Context{Scope: "example", Principal: "local", Client: "default"})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(catalog.Initialize.ServerInfo.Name, len(catalog.Tools))
	// Output: example 0
}
