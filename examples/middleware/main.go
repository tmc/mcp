package main

import (
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcpmiddleware"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "middleware-example", Version: "1.0.0"}, nil)
	server.AddReceivingMiddleware(mcpmiddleware.Log(slog.Default()))
}
